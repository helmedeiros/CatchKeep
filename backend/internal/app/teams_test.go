package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestTeams_CreateRenameAndDelete(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := f.collection.List(t.Context(), ash)
	teams := app.NewTeams(apptest.NewTeams(f.store), f.store, f.ids, f.clock)

	raid, err := teams.Create(t.Context(), ash, "Raid", []domain.PokemonID{list[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teams.Create(t.Context(), ash, "raid", nil); !errors.Is(err, domain.ErrTeamNameTaken) {
		t.Fatalf("same name: err = %v", err)
	}
	if _, err := teams.Create(t.Context(), ash, "GBL", []domain.PokemonID{"missing"}); !errors.Is(err, domain.ErrPokemonNotFound) {
		t.Fatalf("missing member: err = %v", err)
	}
	if _, err := teams.Update(t.Context(), ash, "nope", "X", nil); !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("unknown team: err = %v", err)
	}
	if _, err := teams.Update(t.Context(), ash, raid.ID, "Raid", []domain.PokemonID{list[0].ID, list[1].ID}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := teams.List(t.Context(), ash)
	if len(got) != 1 || len(got[0].Members) != 2 {
		t.Fatalf("teams = %+v", got)
	}
	if err := teams.Delete(t.Context(), ash, raid.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := teams.List(t.Context(), ash); len(got) != 0 {
		t.Fatalf("teams after delete = %+v", got)
	}
}

func TestKeep_TeamMembersAreProtected(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nPidgey,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.collection.ConfirmAbsent(t.Context(), ash, "", "", domain.ConfirmableFields); err != nil {
		t.Fatal(err)
	}
	store := apptest.NewTeams(f.store)
	keep := app.NewKeep(apptest.NewSetups(), f.store, apptest.SpeciesCategories{}, store, f.ids, f.clock)
	if err := keep.SaveSetup(t.Context(), ash, domain.Setup{Capacity: 300, Goals: []domain.Goal{domain.GoalCollecting}}); err != nil {
		t.Fatal(err)
	}
	list, _ := f.collection.List(t.Context(), ash)
	if _, err := app.NewTeams(store, f.store, f.ids, f.clock).Create(t.Context(), ash, "Pets", []domain.PokemonID{list[1].ID}); err != nil {
		t.Fatal(err)
	}

	s, _ := keep.Summary(t.Context(), ash)
	if s.Protected != 1 || s.Groups[0].KeptIDs[0] != list[1].ID {
		t.Fatalf("summary = %+v; the team member is kept", s)
	}
}

func TestTeams_RecordEachChangeInTheHistory(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := f.collection.List(t.Context(), ash)
	teams := app.NewTeams(apptest.NewTeams(f.store), f.store, f.ids, f.clock)

	raid, err := teams.Create(t.Context(), ash, "Raid", []domain.PokemonID{list[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	both := []domain.PokemonID{list[0].ID, list[1].ID}
	for range 2 {
		if _, err := teams.Update(t.Context(), ash, raid.ID, "Raid", both); err != nil {
			t.Fatal(err)
		}
	}
	if err := teams.Delete(t.Context(), ash, raid.ID); err != nil {
		t.Fatal(err)
	}

	changes := f.store.Changes(ash)
	members := func(c domain.Change) int { return len(c.Pokemon) }
	if len(changes) != 3 || changes[0].Kind != domain.ChangeTeam || members(changes[0]) != 1 || members(changes[1]) != 2 || members(changes[2]) != 2 ||
		changes[2].At.IsZero() || changes[2].Pokemon[1].Species != list[1].Current().String(domain.FieldSpecies) {
		t.Fatalf("history = %+v; want created, changed once (saving it unchanged is nothing) and deleted", changes)
	}
}
