package buildinfo_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/buildinfo"
)

func TestVersion_DefaultsToDevWhenNotStampedAtBuild(t *testing.T) {
	if got := buildinfo.Version(); got != "dev" {
		t.Fatalf("Version() = %q, want %q", got, "dev")
	}
}
