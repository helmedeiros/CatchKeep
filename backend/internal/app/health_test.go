package app_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
)

func TestHealth_ReportsOKWithBuildVersion(t *testing.T) {
	got := app.NewHealth("1.2.3").Check(t.Context())

	if got.Status != app.HealthOK || got.Version != "1.2.3" {
		t.Fatalf("Check() = %+v, want status %q and version %q", got, app.HealthOK, "1.2.3")
	}
}
