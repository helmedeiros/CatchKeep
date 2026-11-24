package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestNewImportEvent_ListsWhatTheImportAddedAndChanged(t *testing.T) {
	at := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	existing := []domain.Pokemon{
		mon("P1", "species", "Pidgey", "cp", "120", "hp", "30"),
		mon("P2", "species", "Rattata", "cp", "50"),
	}
	add := []domain.Pokemon{mon("P3", "species", "Zubat", "cp", "80")}
	update := []domain.PokemonUpdate{
		{PokemonID: "P1", Observations: row(2, "species", "Pidgey", "cp", "300", "hp", "45").Observations},
		{PokemonID: "P2", Observations: row(3, "species", "Rattata", "cp", "50").Observations},
	}

	e := domain.NewImportEvent("E1", "ash", "B1", existing, add, update, at)

	if e.ID != "E1" || e.TrainerID != "ash" || e.BatchID != "B1" || !e.At.Equal(at) {
		t.Fatalf("event = %+v", e)
	}
	if len(e.Added) != 1 || e.Added[0].Species != "Zubat" || e.Added[0].CP != 80 {
		t.Fatalf("added = %+v", e.Added)
	}
	if len(e.Changed) != 1 || e.Changed[0].ID != "P1" || e.Changed[0].CP != 300 ||
		!slices.Equal(e.Changed[0].Fields, []domain.Field{domain.FieldCP, domain.FieldHP}) {
		t.Fatalf("changed = %+v; only Pidgey changed, in CP and HP", e.Changed)
	}
}
