package app_test

import (
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestStorage_AttachesHowFullStorageIsToTheEvent(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nPidgey,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	keep := newKeep(apptest.NewSetups(), f.store, apptest.SpeciesCategories{}, apptest.NewTeams(f.store))
	storage := app.NewStorage(keep, f.store)

	storage.RecordStorage(t.Context(), ash, "before-setup")
	if err := keep.SaveSetup(t.Context(), ash, domain.Setup{Capacity: 300, Goals: []domain.Goal{domain.GoalCollecting}, InGameCount: 250}); err != nil {
		t.Fatal(err)
	}
	storage.RecordStorage(t.Context(), ash, "after-setup")

	if got := f.store.Storage("before-setup"); got != (domain.StorageSnapshot{Active: 2}) {
		t.Fatalf("before a setup = %+v, want only the active count", got)
	}
	got := f.store.Storage("after-setup")
	if !got.HasSetup || got.Capacity != 300 || got.InGameCount != 250 || got.Active != 2 || got.Kept+got.PossibleTransfers+got.NeedsConfirmation != 2 {
		t.Fatalf("with a setup = %+v", got)
	}
}
