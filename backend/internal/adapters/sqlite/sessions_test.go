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
