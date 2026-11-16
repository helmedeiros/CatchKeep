package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestKeep_SummaryNeedsSetupFirst(t *testing.T) {
	keep := app.NewKeep(apptest.NewSetups(), apptest.NewCollectionStore(), apptest.RareSpecies{})

	if _, err := keep.Summary(t.Context(), ash); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("err = %v, want ErrSetupNotFound", err)
	}
	if _, err := keep.Setup(t.Context(), ash); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("Setup err = %v, want ErrSetupNotFound", err)
	}
}

func TestKeep_SavesSetupAndSummarizesTheCollection(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	keep := app.NewKeep(apptest.NewSetups(), f.store, apptest.RareSpecies{"Rattata"})

	if err := keep.SaveSetup(t.Context(), ash, domain.Setup{Capacity: 300}); !errors.Is(err, domain.ErrInvalidSetup) {
		t.Fatalf("invalid setup: err = %v", err)
	}
	setup := domain.Setup{Capacity: 300, Goals: []domain.Goal{domain.GoalCollecting}}
	if err := keep.SaveSetup(t.Context(), ash, setup); err != nil {
		t.Fatalf("SaveSetup: %v", err)
	}
	if got, err := keep.Setup(t.Context(), ash); err != nil || got.Capacity != 300 {
		t.Fatalf("Setup = %+v, %v", got, err)
	}

	s, err := keep.Summary(t.Context(), ash)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if s.Total != 2 || s.NeedsConfirmation != 2 {
		t.Fatalf("summary = %+v; imported Pokémon have unknown variant features", s)
	}

	if _, err := f.collection.ConfirmAbsent(t.Context(), ash, "", "", domain.ConfirmableFields); err != nil {
		t.Fatal(err)
	}
	s, _ = keep.Summary(t.Context(), ash)
	rare := map[string]bool{}
	for _, g := range s.Groups {
		rare[g.Species] = g.Rare
	}
	if !rare["Rattata"] || rare["Pidgey"] || s.RareMinimum != 2 {
		t.Fatalf("summary = %+v; Rattata is rare in this test", s)
	}
}

func TestKeep_SpeciesRulesNeedSetupAndSurviveSetupChanges(t *testing.T) {
	keep := app.NewKeep(apptest.NewSetups(), apptest.NewCollectionStore(), apptest.RareSpecies{})
	rules := []domain.SpeciesRule{{Species: "Kyurem", Minimum: 10}}
	if err := keep.SaveSpeciesRules(t.Context(), ash, rules); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("before setup: err = %v", err)
	}
	setup := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalCollecting}}
	if err := keep.SaveSetup(t.Context(), ash, setup); err != nil {
		t.Fatal(err)
	}
	if err := keep.SaveSpeciesRules(t.Context(), ash, []domain.SpeciesRule{{Species: "Kyurem"}}); !errors.Is(err, domain.ErrInvalidSetup) {
		t.Fatalf("invalid rule: err = %v", err)
	}
	if err := keep.SaveSpeciesRules(t.Context(), ash, rules); err != nil {
		t.Fatal(err)
	}

	setup.Capacity = 5000
	if err := keep.SaveSetup(t.Context(), ash, setup); err != nil {
		t.Fatal(err)
	}
	got, _ := keep.Setup(t.Context(), ash)
	if got.Capacity != 5000 || len(got.Species) != 1 || got.Species[0] != rules[0] {
		t.Fatalf("setup = %+v; changing storage keeps the species rules", got)
	}
}
