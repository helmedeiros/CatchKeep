package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestNewTeam(t *testing.T) {
	team, err := domain.NewTeam("T1", "  Raid Kyurem  ", []domain.PokemonID{"P1", "P2", "P1"})
	if err != nil || team.Name != "Raid Kyurem" || !slices.Equal(team.Members, []domain.PokemonID{"P1", "P2"}) {
		t.Fatalf("team = %+v, %v; want a trimmed name and each member once", team, err)
	}
	for name, n := range map[string]string{"empty": "  ", "too long": strings.Repeat("x", 41), "control": "a\tb"} {
		if _, err := domain.NewTeam("T1", n, nil); !errors.Is(err, domain.ErrInvalidTeam) {
			t.Errorf("%s: err = %v, want ErrInvalidTeam", name, err)
		}
	}
	many := make([]domain.PokemonID, 0, 101)
	for i := range 101 {
		many = append(many, domain.PokemonID(strings.Repeat("x", i+1)))
	}
	if _, err := domain.NewTeam("T1", "Big", many); !errors.Is(err, domain.ErrInvalidTeam) {
		t.Errorf("101 members: err = %v, want ErrInvalidTeam", err)
	}
}

func TestProtectTeamMembers(t *testing.T) {
	pokemon := []domain.Pokemon{{ID: "P1"}, {ID: "P2"}, {ID: "P3", Protected: true}}
	teams := []domain.Team{{ID: "T1", Name: "Raid", Members: []domain.PokemonID{"P2"}}}

	got := domain.ProtectTeamMembers(pokemon, teams)

	if got[0].Protected || !got[1].Protected || !got[2].Protected {
		t.Fatalf("protected = %v %v %v; want team members and manual protection", got[0].Protected, got[1].Protected, got[2].Protected)
	}
	if pokemon[1].Protected {
		t.Fatal("the input was changed")
	}
}
