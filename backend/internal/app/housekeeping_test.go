package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSweep_RemovesWeekOldImportsAndIdleSessions(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	store, sessions := apptest.NewCollectionStore(), apptest.NewSessionStore()
	old, recent := now.Add(-domain.StagedImportLifetime-time.Second), now.Add(-time.Hour)
	for id, b := range map[string]domain.ImportBatch{
		"old":    {TrainerID: ash, Status: domain.ImportStaged, CreatedAt: old},
		"done":   {TrainerID: ash, Status: domain.ImportCommitted, CreatedAt: old},
		"recent": {TrainerID: ash, Status: domain.ImportStaged, CreatedAt: recent},
	} {
		b.ID = id
		_ = store.Save(t.Context(), b)
	}
	idle := now.Add(-domain.SessionIdleTimeout - time.Second)
	_ = sessions.Create(t.Context(), domain.Session{TokenHash: "idle", TrainerID: ash, LastSeenAt: idle})
	_ = sessions.Create(t.Context(), domain.Session{TokenHash: "used", TrainerID: ash, LastSeenAt: recent})

	swept, err := app.NewHousekeeping(store, sessions, apptest.NewFixedClock(now)).Sweep(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if swept != (app.Swept{Imports: 1, Sessions: 1}) {
		t.Fatalf("swept = %+v, want 1 import and 1 session", swept)
	}
	if _, err := store.Get(t.Context(), ash, "old"); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("old import: err = %v, want removed", err)
	}
	if _, err := sessions.Get(t.Context(), "used"); err != nil {
		t.Fatalf("used session: %v, want kept", err)
	}
}
