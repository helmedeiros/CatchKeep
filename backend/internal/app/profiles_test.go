package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func newProfiles() (*app.Profiles, *apptest.Trainers) {
	trainers := apptest.NewTrainers()
	return app.NewProfiles(trainers, apptest.PlainHasher{}, apptest.NewSequentialIDs(), apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))), trainers
}

func TestCreateProfile_StoresTrainerWithHashedPIN(t *testing.T) {
	profiles, trainers := newProfiles()

	got, err := profiles.Create(t.Context(), " Ash ", "2468")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got.DisplayName != "Ash" || got.ID == "" {
		t.Fatalf("Create = %+v, want trimmed name and an id", got)
	}
	stored, err := trainers.Get(t.Context(), got.ID)
	if err != nil {
		t.Fatalf("stored trainer: %v", err)
	}
	if stored.PINHash == "2468" || stored.PINHash == "" {
		t.Fatalf("PIN stored as %q, want a hash", stored.PINHash)
	}
}

func TestCreateProfile_RejectsInvalidInput(t *testing.T) {
	profiles, _ := newProfiles()

	if _, err := profiles.Create(t.Context(), "", "2468"); !errors.Is(err, domain.ErrInvalidDisplayName) {
		t.Fatalf("empty name: err = %v", err)
	}
	if _, err := profiles.Create(t.Context(), "Ash", "12"); !errors.Is(err, domain.ErrInvalidPIN) {
		t.Fatalf("short PIN: err = %v", err)
	}
}

func TestCreateProfile_RejectsDuplicateNameIgnoringCase(t *testing.T) {
	profiles, _ := newProfiles()
	if _, err := profiles.Create(t.Context(), "Ash", "2468"); err != nil {
		t.Fatal(err)
	}

	if _, err := profiles.Create(t.Context(), "ash", "1357"); !errors.Is(err, domain.ErrDisplayNameTaken) {
		t.Fatalf("duplicate: err = %v, want ErrDisplayNameTaken", err)
	}
}

func TestListProfiles_ReturnsAllProfilesSortedByName(t *testing.T) {
	profiles, _ := newProfiles()
	for _, name := range []string{"misty", "Brock", "Ash"} {
		if _, err := profiles.Create(t.Context(), name, "2468"); err != nil {
			t.Fatal(err)
		}
	}

	list, err := profiles.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, p := range list {
		names = append(names, p.DisplayName)
	}
	if want := []string{"Ash", "Brock", "misty"}; len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Fatalf("names = %v, want %v", names, want)
	}
}
