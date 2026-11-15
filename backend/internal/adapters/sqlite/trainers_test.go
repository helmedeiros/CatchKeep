package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestTrainers_CreateThenGetAndList(t *testing.T) {
	store, _ := openTemp(t)
	trainers := sqlite.NewTrainers(store)
	ash := domain.Trainer{ID: "01JD2Z8Q4M8V7Y6X5W4T3S2R1P", DisplayName: "Ash", PINHash: "h1", CreatedAt: time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)}

	if err := trainers.Create(t.Context(), ash); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := trainers.Get(t.Context(), ash.ID)
	if err != nil || got != ash {
		t.Fatalf("Get = %+v, %v; want %+v", got, err, ash)
	}
	list, err := trainers.List(t.Context())
	if err != nil || len(list) != 1 || list[0] != ash {
		t.Fatalf("List = %+v, %v", list, err)
	}
}

func TestTrainers_RejectsDuplicateDisplayNameIgnoringCase(t *testing.T) {
	store, _ := openTemp(t)
	trainers := sqlite.NewTrainers(store)
	if err := trainers.Create(t.Context(), domain.Trainer{ID: "A", DisplayName: "Ash", PINHash: "h"}); err != nil {
		t.Fatal(err)
	}

	err := trainers.Create(t.Context(), domain.Trainer{ID: "B", DisplayName: "ASH", PINHash: "h"})

	if !errors.Is(err, domain.ErrDisplayNameTaken) {
		t.Fatalf("err = %v, want ErrDisplayNameTaken", err)
	}
}

func TestTrainers_GetUnknownIsNotFound(t *testing.T) {
	store, _ := openTemp(t)

	_, err := sqlite.NewTrainers(store).Get(t.Context(), "missing")

	if !errors.Is(err, domain.ErrTrainerNotFound) {
		t.Fatalf("err = %v, want ErrTrainerNotFound", err)
	}
}
