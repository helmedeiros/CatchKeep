package app_test

import (
	"errors"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
)

func TestAutostart_TurnsOnAndOff(t *testing.T) {
	a := app.NewAutostart(&apptest.Autostart{})
	if s, _ := a.State(); !s.Supported || s.Enabled {
		t.Fatalf("state = %+v", s)
	}
	if s, err := a.Set(true); err != nil || !s.Enabled {
		t.Fatalf("on: %+v, %v", s, err)
	}
	if s, err := a.Set(false); err != nil || s.Enabled {
		t.Fatalf("off: %+v, %v", s, err)
	}
}

func TestAutostart_UnsupportedComputers(t *testing.T) {
	a := app.NewAutostart(nil)
	if s, err := a.State(); err != nil || s.Supported {
		t.Fatalf("state = %+v, %v", s, err)
	}
	if _, err := a.Set(true); !errors.Is(err, app.ErrAutostartUnsupported) {
		t.Fatalf("err = %v", err)
	}
}
