package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Team errors.
var (
	ErrInvalidTeam   = errors.New("invalid team")
	ErrTeamNotFound  = errors.New("team not found")
	ErrTeamNameTaken = errors.New("team name is already used")
)

const (
	maxTeamNameRunes = 40
	maxTeamMembers   = 100
)

// TeamID identifies a team.
type TeamID string

// Team is a named group of the trainer's Pokémon, all protected (spec
// V1-SET-03, §7.2).
type Team struct {
	ID      TeamID
	Name    string
	Members []PokemonID
}

// NewTeam trims the name, keeps each member once and checks the limits.
func NewTeam(id TeamID, name string, members []PokemonID) (Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxTeamNameRunes || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Team{}, fmt.Errorf("%w: name must be 1 to %d characters", ErrInvalidTeam, maxTeamNameRunes)
	}
	unique := make([]PokemonID, 0, len(members))
	for _, m := range members {
		if !slices.Contains(unique, m) {
			unique = append(unique, m)
		}
	}
	if len(unique) > maxTeamMembers {
		return Team{}, fmt.Errorf("%w: at most %d Pokémon", ErrInvalidTeam, maxTeamMembers)
	}
	return Team{ID: id, Name: name, Members: unique}, nil
}

// ProtectTeamMembers returns a copy of pokemon with every team member
// protected.
func ProtectTeamMembers(pokemon []Pokemon, teams []Team) []Pokemon {
	onTeam := map[PokemonID]bool{}
	for _, t := range teams {
		for _, m := range t.Members {
			onTeam[m] = true
		}
	}
	out := slices.Clone(pokemon)
	for i := range out {
		out[i].Protected = out[i].Protected || onTeam[out[i].ID]
	}
	return out
}
