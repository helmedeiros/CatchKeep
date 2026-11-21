package launchd_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/launchd"
)

const agentFile = "Library/LaunchAgents/" + launchd.Label + ".plist"

func TestAgent_EnableWritesALoginItemAndDisableRemovesIt(t *testing.T) {
	home := t.TempDir()
	agent := launchd.New(home, []string{"/Apps/catchkeep", "serve", "-addr", "0.0.0.0:8080", "-web-dir", "/Apps/web dist"})

	if on, err := agent.Enabled(); err != nil || on {
		t.Fatalf("Enabled = %v, %v before enabling", on, err)
	}
	if err := agent.Enable(); err != nil {
		t.Fatal(err)
	}
	raw, err := fs.ReadFile(os.DirFS(home), agentFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<string>" + launchd.Label + "</string>",
		"<string>/Apps/catchkeep</string>", "<string>serve</string>", "<string>0.0.0.0:8080</string>",
		"<string>/Apps/web dist</string>",
		"<key>RunAtLoad</key>\n\t<true/>",
		filepath.Join(home, "Library", "Logs", "CatchKeep.log"),
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("plist lacks %q:\n%s", want, raw)
		}
	}
	if on, _ := agent.Enabled(); !on {
		t.Fatal("Enabled = false after enabling")
	}

	if err := agent.Disable(); err != nil {
		t.Fatal(err)
	}
	if on, _ := agent.Enabled(); on {
		t.Fatal("Enabled = true after disabling")
	}
	if err := agent.Disable(); err != nil {
		t.Fatalf("disabling twice: %v", err)
	}
}

func TestAgent_EscapesArguments(t *testing.T) {
	home := t.TempDir()
	if err := launchd.New(home, []string{"/a&b/<catchkeep>", "serve"}).Enable(); err != nil {
		t.Fatal(err)
	}
	raw, _ := fs.ReadFile(os.DirFS(home), agentFile)
	if !strings.Contains(string(raw), "<string>/a&amp;b/&lt;catchkeep&gt;</string>") {
		t.Fatalf("arguments not escaped:\n%s", raw)
	}
}
