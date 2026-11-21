// Package launchd starts CatchKeep when the trainer logs in to their Mac,
// through a per-user launch agent (spec §3, opt-in launch at login).
package launchd

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Label names the launch agent.
const Label = "io.github.helmedeiros.catchkeep"

// Agent writes or removes the launch agent. macOS reads it at the next
// login, so changing it never stops or doubles the running CatchKeep.
type Agent struct {
	home string
	args []string
}

// New returns an agent for the user whose home folder is home, starting
// CatchKeep with args (the program first).
func New(home string, args []string) *Agent {
	return &Agent{home: home, args: args}
}

func (a *Agent) path() string {
	return filepath.Join(a.home, "Library", "LaunchAgents", Label+".plist")
}

// Enabled reports whether CatchKeep starts at login.
func (a *Agent) Enabled() (bool, error) {
	_, err := os.Stat(a.path())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Enable makes CatchKeep start at the next login, with the same settings
// as now.
func (a *Agent) Enable() error {
	if err := os.MkdirAll(filepath.Dir(a.path()), 0o750); err != nil {
		return fmt.Errorf("create launch agents folder: %w", err)
	}
	logFile := filepath.Join(a.home, "Library", "Logs", "CatchKeep.log")
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	`)
	writeString(&b, Label)
	b.WriteString("\n\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range a.args {
		b.WriteString("\t\t")
		writeString(&b, arg)
		b.WriteString("\n")
	}
	b.WriteString("\t</array>\n\t<key>RunAtLoad</key>\n\t<true/>\n\t<key>StandardOutPath</key>\n\t")
	writeString(&b, logFile)
	b.WriteString("\n\t<key>StandardErrorPath</key>\n\t")
	writeString(&b, logFile)
	b.WriteString("\n</dict>\n</plist>\n")
	if err := os.WriteFile(a.path(), b.Bytes(), 0o644); err != nil { //nolint:gosec // launchd requires a readable agent file
		return fmt.Errorf("write launch agent: %w", err)
	}
	return nil
}

// Disable stops CatchKeep from starting at the next login.
func (a *Agent) Disable() error {
	if err := os.Remove(a.path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove launch agent: %w", err)
	}
	return nil
}

func writeString(b *bytes.Buffer, s string) {
	b.WriteString("<string>")
	_ = xml.EscapeText(b, []byte(s))
	b.WriteString("</string>")
}
