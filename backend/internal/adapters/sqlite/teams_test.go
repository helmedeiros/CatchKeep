package sqlite_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestTeams_SaveListReplaceAndDelete(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status = domain.ImportCommitted
	obs := b.Parsed.Rows[0].Observations
	if err := coll.Commit(t.Context(), b, []domain.Pokemon{{ID: "P1", TrainerID: "ash", Observations: obs}, {ID: "P2", TrainerID: "ash", Observations: obs}}, nil); err != nil {
		t.Fatal(err)
	}
	teams := sqlite.NewTeams(store)

	raid := domain.Team{ID: "T1", Name: "Raid", Members: []domain.PokemonID{"P2", "P1"}}
	if err := teams.SaveTeam(t.Context(), "ash", raid); err != nil {
		t.Fatal(err)
	}
	if err := teams.SaveTeam(t.Context(), "ash", domain.Team{ID: "T2", Name: "Empty"}); err != nil {
		t.Fatal(err)
	}
	got, err := teams.ListTeams(t.Context(), "ash")
	if err != nil || len(got) != 2 || got[0].Name != "Empty" || len(got[0].Members) != 0 || !slices.Equal(got[1].Members, raid.Members) {
		t.Fatalf("teams = %+v, %v; want both by name, members in order", got, err)
	}

	if err := teams.SaveTeam(t.Context(), "misty", domain.Team{ID: "T1", Name: "Stolen"}); !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("misty overwriting ash's team: err = %v", err)
	}
	if err := teams.SaveTeam(t.Context(), "misty", domain.Team{ID: "T9", Name: "Mine", Members: []domain.PokemonID{"P1"}}); !errors.Is(err, domain.ErrPokemonNotFound) {
		t.Fatalf("misty adding ash's Pokémon: err = %v", err)
	}
	if mine, _ := teams.ListTeams(t.Context(), "misty"); len(mine) != 0 {
		t.Fatalf("a failed save left %+v behind", mine)
	}

	raid.Members = []domain.PokemonID{"P1"}
	if err := teams.SaveTeam(t.Context(), "ash", raid); err != nil {
		t.Fatal(err)
	}
	if err := teams.DeleteTeam(t.Context(), "misty", "T1"); !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("misty deleting ash's team: err = %v", err)
	}
	if err := teams.DeleteTeam(t.Context(), "ash", "T2"); err != nil {
		t.Fatal(err)
	}
	got, _ = teams.ListTeams(t.Context(), "ash")
	if len(got) != 1 || !slices.Equal(got[0].Members, []domain.PokemonID{"P1"}) {
		t.Fatalf("teams = %+v", got)
	}
}
