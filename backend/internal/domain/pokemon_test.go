package domain_test

import (
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

var t0 = time.Date(2026, 9, 5, 22, 30, 0, 0, time.UTC)

func obs(f domain.Field, v string, src domain.Source, at time.Time) domain.Observation {
	return domain.Observation{Field: f, Value: v, Source: src, ObservedAt: at}
}

func TestCurrent_LatestObservationWins(t *testing.T) {
	p := domain.Pokemon{Observations: []domain.Observation{
		obs(domain.FieldCP, "1200", domain.SourceCSV, t0),
		obs(domain.FieldCP, "1500", domain.SourceCSV, t0.Add(time.Hour)),
		obs(domain.FieldCP, "1300", domain.SourceCSV, t0.Add(-time.Hour)),
	}}

	if cp, ok := p.Current().Int(domain.FieldCP); !ok || cp != 1500 {
		t.Fatalf("CP = %d, %v; want 1500", cp, ok)
	}
}

func TestCurrent_ManualCorrectionBeatsLaterImports(t *testing.T) {
	p := domain.Pokemon{Observations: []domain.Observation{
		obs(domain.FieldSpecies, "Pidgey", domain.SourceManual, t0),
		obs(domain.FieldSpecies, "Pidgeot", domain.SourceCSV, t0.Add(24*time.Hour)),
	}}

	if got := p.Current().String(domain.FieldSpecies); got != "Pidgey" {
		t.Fatalf("species = %q, want manual value Pidgey", got)
	}
}

func TestCurrent_UnobservedFeaturesAreUnknownNotAbsent(t *testing.T) {
	p := domain.Pokemon{Observations: []domain.Observation{
		obs(domain.FieldLucky, "true", domain.SourceCSV, t0),
		obs(domain.FieldShadow, "false", domain.SourceManual, t0),
	}}
	cur := p.Current()

	if got := cur.Tri(domain.FieldLucky); got != domain.Yes {
		t.Fatalf("lucky = %v, want Yes", got)
	}
	if got := cur.Tri(domain.FieldShadow); got != domain.No {
		t.Fatalf("shadow = %v, want No", got)
	}
	if got := cur.Tri(domain.FieldShiny); got != domain.Unknown {
		t.Fatalf("shiny = %v, want Unknown", got)
	}
	if _, ok := cur.Int(domain.FieldCP); ok {
		t.Fatal("CP reported as known with no observation")
	}
}

func TestIVs_AreKnownOnlyWhenAllThreeAre(t *testing.T) {
	partial := domain.Pokemon{Observations: []domain.Observation{
		obs(domain.FieldIVAttack, "15", domain.SourceCSV, t0),
		obs(domain.FieldIVDefense, "14", domain.SourceCSV, t0),
	}}
	if _, ok := partial.Current().IVs(); ok {
		t.Fatal("IVs known with stamina missing")
	}

	full := domain.Pokemon{Observations: append(partial.Observations, obs(domain.FieldIVStamina, "13", domain.SourceCSV, t0))}
	iv, ok := full.Current().IVs()
	if !ok || iv != (domain.IVs{Attack: 15, Defense: 14, Stamina: 13}) {
		t.Fatalf("IVs = %+v, %v", iv, ok)
	}
}
