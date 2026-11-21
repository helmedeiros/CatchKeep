package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestNewTransfer(t *testing.T) {
	at := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	pokemon := []domain.Pokemon{dexMon("P1", "Pidgey", "16", "120", "30"), dexMon("P2", "Pidgey", "16", "90", "29")}

	e, err := domain.NewTransfer("E1", "ash", pokemon, []domain.PokemonID{"P1", "P2"}, "req-1", at)
	if err != nil {
		t.Fatal(err)
	}
	if e.Kind != domain.TransferBulk || len(e.Pokemon) != 2 || e.Pokemon[0].Species != "Pidgey" || e.Pokemon[0].CP != 120 || !e.At.Equal(at) {
		t.Fatalf("event = %+v", e)
	}
	if one, _ := domain.NewTransfer("E2", "ash", pokemon, []domain.PokemonID{"P1"}, "req-2", at); one.Kind != domain.TransferSingle {
		t.Fatalf("kind = %q", one.Kind)
	}

	var gone *domain.NotActiveError
	if _, err := domain.NewTransfer("E3", "ash", pokemon, []domain.PokemonID{"P1", "X9", "X8"}, "req-3", at); !errors.As(err, &gone) || len(gone.IDs) != 2 {
		t.Fatalf("err = %v; the whole confirmation is refused, naming both", err)
	}
	for name, ids := range map[string][]domain.PokemonID{"none": nil, "twice": {"P1", "P1"}} {
		if _, err := domain.NewTransfer("E4", "ash", pokemon, ids, "req-4", at); !errors.Is(err, domain.ErrInvalidTransfer) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := domain.NewTransfer("E5", "ash", pokemon, []domain.PokemonID{"P1"}, " ", at); !errors.Is(err, domain.ErrInvalidTransfer) {
		t.Errorf("no request id: err = %v", err)
	}
}
