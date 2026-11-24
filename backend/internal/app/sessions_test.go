package app_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

type sessionsFixture struct {
	sessions *app.Sessions
	clock    *apptest.FixedClock
	ash      app.ProfileSummary
}

func newSessionsFixture(t *testing.T) sessionsFixture {
	t.Helper()
	trainers := apptest.NewTrainers()
	clock := apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	profiles := app.NewProfiles(trainers, apptest.PlainHasher{}, apptest.NewSequentialIDs(), clock)
	ash, err := profiles.Create(t.Context(), "Ash", "2468")
	if err != nil {
		t.Fatal(err)
	}
	sessions := app.NewSessions(trainers, apptest.NewSessionStore(), apptest.PlainHasher{}, apptest.NewSequentialTokens(), clock)
	return sessionsFixture{sessions: sessions, clock: clock, ash: ash}
}

func TestSignIn_WithTheRightPINStartsASession(t *testing.T) {
	f := newSessionsFixture(t)

	token, me, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	if token == "" || me != f.ash {
		t.Fatalf("SignIn = %q, %+v; want a token and %+v", token, me, f.ash)
	}
	resolved, err := f.sessions.Resolve(t.Context(), token)
	if err != nil || resolved != f.ash {
		t.Fatalf("Resolve = %+v, %v", resolved, err)
	}
}

func TestSignIn_WrongPINOrUnknownProfileLooksTheSame(t *testing.T) {
	f := newSessionsFixture(t)

	if _, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111"); !errors.Is(err, domain.ErrWrongPIN) {
		t.Fatalf("wrong PIN: err = %v, want ErrWrongPIN", err)
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "phone", "nobody", "2468"); !errors.Is(err, domain.ErrWrongPIN) {
		t.Fatalf("unknown profile: err = %v, want ErrWrongPIN", err)
	}
}

func TestSignIn_LocksTheProfileAfterFiveWrongPINsAndBacksOff(t *testing.T) {
	f := newSessionsFixture(t)
	for range 5 {
		_, _, _ = f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111")
	}

	_, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	var locked *domain.LockedOutError
	if !errors.As(err, &locked) || !locked.Until.Equal(f.clock.Now().Add(time.Minute)) {
		t.Fatalf("right PIN while locked: err = %v, want lockout for 1 minute", err)
	}

	f.clock.Advance(time.Minute)
	_, _, _ = f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111")
	_, _, err = f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	if !errors.As(err, &locked) || !locked.Until.Equal(f.clock.Now().Add(2*time.Minute)) {
		t.Fatalf("after another failure: err = %v, want lockout doubled to 2 minutes", err)
	}
}

func TestSignIn_SuccessResetsTheFailureCount(t *testing.T) {
	f := newSessionsFixture(t)
	for range 4 {
		_, _, _ = f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111")
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468"); err != nil {
		t.Fatal(err)
	}

	_, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111")

	if !errors.Is(err, domain.ErrWrongPIN) {
		t.Fatalf("err = %v, want ErrWrongPIN (not locked)", err)
	}
}

func TestResolve_ExpiresSessionsIdleFor30Days(t *testing.T) {
	f := newSessionsFixture(t)
	token, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	if err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(29 * 24 * time.Hour)
	if _, err := f.sessions.Resolve(t.Context(), token); err != nil {
		t.Fatalf("after 29 days: %v", err)
	}
	f.clock.Advance(29 * 24 * time.Hour)
	if _, err := f.sessions.Resolve(t.Context(), token); err != nil {
		t.Fatalf("29 days after last use: %v", err)
	}
	f.clock.Advance(31 * 24 * time.Hour)
	if _, err := f.sessions.Resolve(t.Context(), token); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("after 31 idle days: err = %v, want ErrNoSession", err)
	}
}

func TestSignOut_EndsTheSession(t *testing.T) {
	f := newSessionsFixture(t)
	token, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.sessions.SignOut(t.Context(), token); err != nil {
		t.Fatal(err)
	}

	if _, err := f.sessions.Resolve(t.Context(), token); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("err = %v, want ErrNoSession", err)
	}
}

func TestResetPIN_SetsTheNewPINSignsOutEverywhereAndLiftsALockout(t *testing.T) {
	f := newSessionsFixture(t)
	token, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468")
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		_, _, _ = f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1111")
	}

	if err := f.sessions.ResetPIN(t.Context(), f.ash.ID, "12"); !errors.Is(err, domain.ErrInvalidPIN) {
		t.Fatalf("short PIN: err = %v", err)
	}
	if err := f.sessions.ResetPIN(t.Context(), "ghost", "1357"); !errors.Is(err, domain.ErrTrainerNotFound) {
		t.Fatalf("unknown profile: err = %v", err)
	}
	if err := f.sessions.ResetPIN(t.Context(), f.ash.ID, "1357"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.sessions.Resolve(t.Context(), token); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("old session: err = %v, want signed out", err)
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468"); !errors.Is(err, domain.ErrWrongPIN) {
		t.Fatalf("old PIN: err = %v", err)
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "1357"); err != nil {
		t.Fatalf("new PIN: err = %v", err)
	}
}

func TestSignIn_PausesADeviceThatGuessesAcrossProfiles(t *testing.T) {
	f := newSessionsFixture(t)
	for i := range 10 {
		_, _, _ = f.sessions.SignIn(t.Context(), "laptop", domain.TrainerID(fmt.Sprintf("guess-%d", i)), "1111")
	}

	_, _, err := f.sessions.SignIn(t.Context(), "laptop", f.ash.ID, "2468")
	var locked *domain.LockedOutError
	if !errors.As(err, &locked) || !locked.Until.Equal(f.clock.Now().Add(15*time.Minute)) {
		t.Fatalf("right PIN from the guessing device: err = %v, want a 15-minute pause", err)
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "phone", f.ash.ID, "2468"); err != nil {
		t.Fatalf("another device: err = %v, want signed in", err)
	}

	f.clock.Advance(15 * time.Minute)
	if _, _, err := f.sessions.SignIn(t.Context(), "laptop", f.ash.ID, "2468"); err != nil {
		t.Fatalf("after the pause: err = %v, want signed in", err)
	}
}

func TestSignIn_SigningInBetweenGuessesDoesNotClearADevicesCount(t *testing.T) {
	f := newSessionsFixture(t)
	for i := range 9 {
		_, _, _ = f.sessions.SignIn(t.Context(), "laptop", domain.TrainerID(fmt.Sprintf("guess-%d", i)), "1111")
	}
	if _, _, err := f.sessions.SignIn(t.Context(), "laptop", f.ash.ID, "2468"); err != nil {
		t.Fatal(err)
	}
	_, _, _ = f.sessions.SignIn(t.Context(), "laptop", "guess-9", "1111")

	var locked *domain.LockedOutError
	if _, _, err := f.sessions.SignIn(t.Context(), "laptop", f.ash.ID, "2468"); !errors.As(err, &locked) {
		t.Fatalf("err = %v, want the device paused", err)
	}
}

func TestSignIn_ADevicesOldMistakesExpire(t *testing.T) {
	f := newSessionsFixture(t)
	for i := range 9 {
		_, _, _ = f.sessions.SignIn(t.Context(), "laptop", domain.TrainerID(fmt.Sprintf("guess-%d", i)), "1111")
	}
	f.clock.Advance(15 * time.Minute)
	_, _, _ = f.sessions.SignIn(t.Context(), "laptop", "guess-9", "1111")

	if _, _, err := f.sessions.SignIn(t.Context(), "laptop", f.ash.ID, "2468"); err != nil {
		t.Fatalf("err = %v, want mistakes older than 15 minutes forgotten", err)
	}
}
