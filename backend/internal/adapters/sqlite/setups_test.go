package sqlite_test

import (
	"errors"
	"slices"
	"testing"

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
	second := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalBattling, domain.GoalTrading}}
	if err := setups.SaveSetup(t.Context(), "ash", first); err != nil {
		t.Fatal(err)
	}
	if err := setups.SaveSetup(t.Context(), "ash", second); err != nil {
		t.Fatal(err)
	}

	got, err := setups.GetSetup(t.Context(), "ash")
	if err != nil || got.Capacity != 2700 || !slices.Equal(got.Goals, second.Goals) {
		t.Fatalf("GetSetup = %+v, %v", got, err)
	}
	if _, err := setups.GetSetup(t.Context(), "misty"); !errors.Is(err, domain.ErrSetupNotFound) {
		t.Fatalf("misty: err = %v", err)
	}
}
