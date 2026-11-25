package sqlite_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/ids"
	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSetups_SaveReplaceAndGetPerTrainer(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	setups := sqlite.NewSetups(store)

	if _, err := setups.GetSetup(t.Context(), "ash"); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("before saving: err = %v", err)
	}
	first := domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}
	second := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalBattling, domain.GoalTrading}, InGameCount: 2500, Limit: 75}
	if err := setups.SaveSetup(t.Context(), "ash", first, setupChange("ash")); err != nil {
		t.Fatal(err)
	}
	if err := setups.SaveSetup(t.Context(), "ash", second, setupChange("ash")); err != nil {
		t.Fatal(err)
	}

	got, err := setups.GetSetup(t.Context(), "ash")
	if err != nil || got.Capacity != 2700 || !slices.Equal(got.Goals, second.Goals) || got.InGameCount != 2500 || got.Limit != 75 {
		t.Fatalf("GetSetup = %+v, %v", got, err)
	}
	if _, err := setups.GetSetup(t.Context(), "misty"); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("misty: err = %v", err)
	}
}

func TestSetups_KeepSpeciesRules(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	setups := sqlite.NewSetups(store)
	rules := []domain.SpeciesRule{{Species: "Kyurem", Minimum: 10}, {Species: "Eevee", Favorite: true}}
	if err := setups.SaveSetup(t.Context(), "ash", domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}, Species: rules}, setupChange("ash")); err != nil {
		t.Fatal(err)
	}

	got, err := setups.GetSetup(t.Context(), "ash")
	if err != nil || !slices.Equal(got.Species, rules) {
		t.Fatalf("Species = %+v, %v", got.Species, err)
	}
}

func TestSetups_KeepCategoryRules(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	setups := sqlite.NewSetups(store)
	rules := []domain.CategoryRule{{Category: domain.CategoryMythical, Minimum: 4}, {Category: domain.CategoryUltraBeast, Favorite: true}}
	if err := setups.SaveSetup(t.Context(), "ash", domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}, Categories: rules}, setupChange("ash")); err != nil {
		t.Fatal(err)
	}

	got, err := setups.GetSetup(t.Context(), "ash")
	if err != nil || !slices.Equal(got.Categories, rules) {
		t.Fatalf("Categories = %+v, %v", got.Categories, err)
	}
}

func TestSetups_RecordEachSaveInTheHistory(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	setups := sqlite.NewSetups(store)
	setup := domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}, InGameCount: 300}
	if err := setups.SaveSetup(t.Context(), "ash", setup, setupChange("ash")); err != nil {
		t.Fatal(err)
	}
	setup.Species = []domain.SpeciesRule{{Species: "Eevee", Favorite: true}}
	rules := setupChange("ash")
	rules.Kind = domain.ChangeRules
	if err := setups.SaveSetup(t.Context(), "ash", setup, rules); err != nil {
		t.Fatal(err)
	}

	var kinds, details string
	if err := store.DB().QueryRowContext(t.Context(), `
		SELECT group_concat(kind), group_concat(detail, ' ') FROM (SELECT kind, detail FROM history_events WHERE trainer_id = 'ash' ORDER BY rowid)`).Scan(&kinds, &details); err != nil {
		t.Fatal(err)
	}
	want := `{"capacity":350,"goals":["collecting"],"inGameCount":300} {"species":[{"species":"Eevee","favorite":true}],"categories":[]}`
	if kinds != "setup,rules" || details != want {
		t.Fatalf("history = %s %s, want setup,rules %s", kinds, details, want)
	}
}

// setupChange is a setup change by trainer with an id of its own.
func setupChange(trainer domain.TrainerID) domain.Change {
	return domain.Change{ID: ids.NewULID().NewID(), TrainerID: trainer, Kind: domain.ChangeSetup, Pokemon: []domain.Transferred{}, At: at}
}
