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

func TestConfirmAbsent_NoSpeciesMeansTheWholeCollection(t *testing.T) {
	pokemon := []domain.Pokemon{
		mon("P1", "species", "Pidgey"),
		mon("P2", "species", "Rattata", "form", "Alola"),
		mon("P3", "species", "Mewtwo", "shiny", "true"),
	}

	updates, err := domain.ConfirmAbsent(pokemon, "", "", []domain.Field{domain.FieldShiny}, confirmedAt)

	if err != nil || len(updates) != 2 || updates[0].PokemonID != "P1" || updates[1].PokemonID != "P2" {
		t.Fatalf("updates = %+v, err = %v; want P1 and P2 only", updates, err)
	}
}

func TestMarkFeatures_RecordsTheTrainersAnswerForOnePokemon(t *testing.T) {
	at := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	export := mon("P1", append(knownVariant("Dialga"), "shadow", "true")...)

	u, err := domain.MarkFeatures(export, map[domain.Field]bool{domain.FieldPurified: true, domain.FieldShiny: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	export.Observations = append(export.Observations, u.Observations...)
	cur := export.Current()
	if cur.Tri(domain.FieldPurified) != domain.Yes || cur.Tri(domain.FieldShadow) != domain.No || cur.Tri(domain.FieldShiny) != domain.Yes {
		t.Fatalf("current = %v; purified, so not shadow, and shiny", cur)
	}
	for _, o := range u.Observations {
		if o.Source != domain.SourceManual || !o.ObservedAt.Equal(at) {
			t.Fatalf("observation %+v should be the trainer's own, now", o)
		}
	}

	later := mon("P1", append(knownVariant("Dialga"), "shadow", "true")...)
	later.Observations = append(u.Observations, later.Observations...)
	for i := range later.Observations[len(u.Observations):] {
		later.Observations[len(u.Observations)+i].ObservedAt = at.Add(time.Hour)
	}
	if later.Current().Tri(domain.FieldShadow) != domain.No {
		t.Fatal("a newer export must not override the trainer's answer")
	}

	if _, err := domain.MarkFeatures(export, map[domain.Field]bool{domain.FieldShadow: true, domain.FieldPurified: true}, at); !errors.Is(err, domain.ErrNotConfirmable) {
		t.Fatalf("shadow and purified together: err = %v", err)
	}
	if _, err := domain.MarkFeatures(export, map[domain.Field]bool{domain.FieldCP: true}, at); !errors.Is(err, domain.ErrNotConfirmable) {
		t.Fatalf("CP is not a yes/no feature: err = %v", err)
	}
	if _, err := domain.MarkFeatures(export, nil, at); !errors.Is(err, domain.ErrNotConfirmable) {
		t.Fatalf("nothing to mark: err = %v", err)
	}
}
