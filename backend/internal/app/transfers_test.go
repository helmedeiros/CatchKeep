package app_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestTransfers_ConfirmUndoAndHistory(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nPidgey,4,5,6,120\nRattata,7,8,9,90\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	transfers := app.NewTransfers(f.store, f.store, f.store, f.storage, apptest.NewSequentialIDs(), apptest.NewFixedClock(time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)))
	list, _ := f.collection.List(t.Context(), ash)
	ids := []domain.PokemonID{list[1].ID, list[2].ID}
	rev, _ := transfers.Revision(t.Context(), ash)

	if _, err := transfers.Confirm(t.Context(), ash, ids, "r1", rev+1); !errors.Is(err, domain.ErrStaleRevision) {
		t.Fatalf("stale plan: err = %v", err)
	}
	e, err := transfers.Confirm(t.Context(), ash, ids, "r1", rev)
	if err != nil || e.Kind != domain.TransferBulk || len(e.Pokemon) != 2 {
		t.Fatalf("confirm: %+v, %v", e, err)
	}
	if again, err := transfers.Confirm(t.Context(), ash, ids, "r1", rev); err != nil || again.ID != e.ID {
		t.Fatalf("repeat: %+v, %v; counts once", again, err)
	}
	if left, _ := f.collection.List(t.Context(), ash); len(left) != 1 {
		t.Fatalf("collection = %d, want 1 left", len(left))
	}
	var gone *domain.NotActiveError
	if _, err := transfers.Confirm(t.Context(), ash, ids[:1], "r2", app.AnyRevision); !errors.As(err, &gone) {
		t.Fatalf("already transferred: err = %v", err)
	}

	undo, err := transfers.Undo(t.Context(), ash, e.ID)
	if err != nil || undo.Undoes != e.ID {
		t.Fatalf("undo: %+v, %v", undo, err)
	}
	if back, _ := f.collection.List(t.Context(), ash); len(back) != 3 {
		t.Fatalf("collection = %d after undo, want 3", len(back))
	}
	history, _ := transfers.History(t.Context(), ash)
	if len(history) != 2 || history[0].Kind != domain.TransferUndo {
		t.Fatalf("history = %+v", history)
	}
	if got := f.storage.Events()[1:]; !slices.Equal(got, []string{e.ID, undo.ID}) {
		t.Fatalf("storage recorded after %v, want after the transfer and the undo only", got)
	}
}
