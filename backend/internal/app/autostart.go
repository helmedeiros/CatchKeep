package app

import (
	"errors"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// ErrAutostartUnsupported means this computer cannot start CatchKeep at
// login from inside the app.
var ErrAutostartUnsupported = errors.New("starting at login is not available on this computer")

// AutostartState says whether CatchKeep can and will start at login.
type AutostartState struct {
	Supported, Enabled bool
}

// Autostart turns starting CatchKeep at login on or off.
type Autostart struct {
	agent ports.Autostart
}

// NewAutostart wires the use case; agent is nil where it is unsupported.
func NewAutostart(agent ports.Autostart) *Autostart {
	return &Autostart{agent: agent}
}

// State reports whether CatchKeep starts at login.
func (a *Autostart) State() (AutostartState, error) {
	if a.agent == nil {
		return AutostartState{}, nil
	}
	on, err := a.agent.Enabled()
	if err != nil {
		return AutostartState{}, fmt.Errorf("check login item: %w", err)
	}
	return AutostartState{Supported: true, Enabled: on}, nil
}

// Set turns starting at login on or off; it takes effect at the next login.
func (a *Autostart) Set(on bool) (AutostartState, error) {
	if a.agent == nil {
		return AutostartState{}, ErrAutostartUnsupported
	}
	change := a.agent.Disable
	if on {
		change = a.agent.Enable
	}
	if err := change(); err != nil {
		return AutostartState{}, fmt.Errorf("change login item: %w", err)
	}
	return a.State()
}
