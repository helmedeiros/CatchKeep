package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSessions_CreateGetTouchDelete(t *testing.T) {
	store, _ := openTemp(t)
	if err := sqlite.NewTrainers(store).Create(t.Context(), domain.Trainer{ID: "T1", DisplayName: "Ash", PINHash: "h"}); err != nil {
		t.Fatal(err)
	}
	sessions := sqlite.NewSessions(store)
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	s := domain.Session{TokenHash: "abc", TrainerID: "T1", CreatedAt: at, LastSeenAt: at}

	if err := sessions.Create(t.Context(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, err := sessions.Get(t.Context(), "abc"); err != nil || got != s {
		t.Fatalf("Get = %+v, %v", got, err)
	}
	later := at.Add(time.Hour)
	if err := sessions.Touch(t.Context(), "abc", later); err != nil {
		t.Fatal(err)
	}
	if got, _ := sessions.Get(t.Context(), "abc"); !got.LastSeenAt.Equal(later) {
		t.Fatalf("LastSeenAt = %v, want %v", got.LastSeenAt, later)
	}
	if err := sessions.Delete(t.Context(), "abc"); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Get(t.Context(), "abc"); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("after delete: err = %v, want ErrNoSession", err)
	}
}

func TestSessions_RequireAnExistingTrainer(t *testing.T) {
	store, _ := openTemp(t)

	err := sqlite.NewSessions(store).Create(t.Context(), domain.Session{TokenHash: "x", TrainerID: "ghost"})

	if err == nil {
		t.Fatal("session for unknown trainer accepted")
	}
}

func TestSessions_DeleteForTrainerAndSetPINHash(t *testing.T) {
	store, _ := openTemp(t)
	trainers := sqlite.NewTrainers(store)
	for _, id := range []domain.TrainerID{"T1", "T2"} {
		if err := trainers.Create(t.Context(), domain.Trainer{ID: id, DisplayName: string(id), PINHash: "h"}); err != nil {
			t.Fatal(err)
		}
	}
	sessions := sqlite.NewSessions(store)
	for hash, id := range map[string]domain.TrainerID{"a": "T1", "b": "T1", "c": "T2"} {
		if err := sessions.Create(t.Context(), domain.Session{TokenHash: hash, TrainerID: id}); err != nil {
			t.Fatal(err)
		}
	}

	if err := sessions.DeleteForTrainer(t.Context(), "T1"); err != nil {
		t.Fatal(err)
	}
	for hash, gone := range map[string]bool{"a": true, "b": true, "c": false} {
		if _, err := sessions.Get(t.Context(), hash); errors.Is(err, domain.ErrNoSession) != gone {
			t.Errorf("session %s: err = %v, gone should be %v", hash, err, gone)
		}
	}

	if err := trainers.SetPINHash(t.Context(), "T1", "new"); err != nil {
		t.Fatal(err)
	}
	if got, _ := trainers.Get(t.Context(), "T1"); got.PINHash != "new" {
		t.Fatalf("PINHash = %q", got.PINHash)
	}
	if err := trainers.SetPINHash(t.Context(), "ghost", "x"); !errors.Is(err, domain.ErrTrainerNotFound) {
		t.Fatalf("unknown trainer: err = %v", err)
	}
}

func TestSessions_DeleteIdleSince(t *testing.T) {
	store, _ := openTemp(t)
	if err := sqlite.NewTrainers(store).Create(t.Context(), domain.Trainer{ID: "T1", DisplayName: "Ash", PINHash: "h"}); err != nil {
		t.Fatal(err)
	}
	sessions := sqlite.NewSessions(store)
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	for hash, seen := range map[string]time.Time{"idle": at, "used": at.Add(time.Hour + time.Millisecond)} {
		if err := sessions.Create(t.Context(), domain.Session{TokenHash: hash, TrainerID: "T1", CreatedAt: at, LastSeenAt: seen}); err != nil {
			t.Fatal(err)
		}
	}

	n, err := sessions.DeleteIdleSince(t.Context(), at.Add(time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("DeleteIdleSince = %d, %v; want 1", n, err)
	}
	if _, err := sessions.Get(t.Context(), "idle"); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("idle session: err = %v, want removed", err)
	}
	if _, err := sessions.Get(t.Context(), "used"); err != nil {
		t.Fatalf("used session: %v, want kept", err)
	}
}
