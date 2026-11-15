package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

var confirmedAt = time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)

func TestConfirmAbsent_RecordsManualNoForUnknownsOfThatSpeciesAndForm(t *testing.T) {
	pokemon := []domain.Pokemon{
		mon("P1", "species", "Pidgey", "form", "Normal"),
		mon("P2", "species", "pidgey", "form", "Normal", "shiny", "true"),
		mon("P3", "species", "Pidgey", "form", "Alola"),
		mon("P4", "species", "Rattata", "form", "Normal"),
	}

	updates, err := domain.ConfirmAbsent(pokemon, "Pidgey", "Normal", []domain.Field{domain.FieldShiny, domain.FieldCostume}, confirmedAt)
	if err != nil {
		t.Fatal(err)
	}

	if len(updates) != 2 || updates[0].PokemonID != "P1" || updates[1].PokemonID != "P2" {
		t.Fatalf("updates = %+v, want P1 and P2", updates)
	}
	if len(updates[0].Observations) != 2 || len(updates[1].Observations) != 1 {
		t.Fatalf("P1 gets %d, P2 gets %d observations; want 2 and only costume for the known-shiny P2",
			len(updates[0].Observations), len(updates[1].Observations))
	}
	o := updates[1].Observations[0]
	if o.Field != domain.FieldCostume || o.Value != "false" || o.Source != domain.SourceManual || !o.ObservedAt.Equal(confirmedAt) {
		t.Fatalf("observation = %+v", o)
	}
}

func TestConfirmAbsent_EmptyFormMeansNormal(t *testing.T) {
	updates, err := domain.ConfirmAbsent([]domain.Pokemon{mon("P1", "species", "Pidgey")}, "Pidgey", "Normal", []domain.Field{domain.FieldShiny}, confirmedAt)
	if err != nil || len(updates) != 1 {
		t.Fatalf("updates = %+v, err = %v", updates, err)
	}
}

func TestConfirmAbsent_OnlyFeaturesATrainerCanSeeInGame(t *testing.T) {
	for _, fields := range [][]domain.Field{nil, {domain.FieldCP}, {domain.FieldLucky}} {
		if _, err := domain.ConfirmAbsent(nil, "Pidgey", "Normal", fields, confirmedAt); !errors.Is(err, domain.ErrNotConfirmable) {
			t.Errorf("fields %v: err = %v, want ErrNotConfirmable", fields, err)
		}
	}
}
