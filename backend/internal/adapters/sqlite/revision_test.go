package sqlite_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestRevision_EveryMutationThatCanChangeRecommendationsBumpsIt(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	trainers := sqlite.NewTrainers(store)
	coll := sqlite.NewCollection(store)
	setups := sqlite.NewSetups(store)
	teams := sqlite.NewTeams(store)
	ctx := t.Context()
	b := stagedBatch()
	obs := b.Parsed.Rows[0].Observations

	revision := func(id domain.TrainerID) int64 {
		t.Helper()
		tr, err := trainers.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return tr.Revision
	}
	if r := revision("ash"); r != 0 {
		t.Fatalf("new trainer revision = %d, want 0", r)
	}
	if err := coll.Save(ctx, b); err != nil {
		t.Fatal(err)
	}
	if r := revision("ash"); r != 0 {
		t.Fatalf("staging an import bumped the revision to %d", r)
	}

	b.Status = domain.ImportCommitted
	steps := []struct {
		name   string
		mutate func() error
	}{
		{"commit an import", func() error {
			return coll.Commit(ctx, b, []domain.Pokemon{{ID: "P1", TrainerID: "ash", Observations: obs}}, nil, imported(b))
		}},
		{"add observations", func() error {
			return coll.AddObservations(ctx, "ash", []domain.PokemonUpdate{{PokemonID: "P1", Observations: obs}})
		}},
		{"protect", func() error { return coll.SetProtected(ctx, "ash", "P1", true, protecting("ash", "C1")) }},
		{"save setup", func() error {
			return setups.SaveSetup(ctx, "ash", domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}})
		}},
		{"save team", func() error {
			return teams.SaveTeam(ctx, "ash", domain.Team{ID: "T1", Name: "Raid", Members: []domain.PokemonID{"P1"}}, teamChange("ash"))
		}},
		{"delete team", func() error { return teams.DeleteTeam(ctx, "ash", "T1", teamChange("ash")) }},
	}
	for i, s := range steps {
		if err := s.mutate(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if r := revision("ash"); r != int64(i+1) {
			t.Fatalf("after %s revision = %d, want %d", s.name, r, i+1)
		}
	}

	if err := coll.SetProtected(ctx, "misty", "P1", false, protecting("misty", "C2")); err == nil {
		t.Fatal("misty protected ash's Pokémon")
	}
	if a, m := revision("ash"), revision("misty"); a != int64(len(steps)) || m != 0 {
		t.Fatalf("a failed mutation changed revisions: ash %d, misty %d", a, m)
	}
}
