package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Teams manages a trainer's named teams, whose members are protected.
type Teams struct {
	teams ports.Teams
	ids   ports.IDs
}

// NewTeams wires the team use cases.
func NewTeams(teams ports.Teams, ids ports.IDs) *Teams {
	return &Teams{teams: teams, ids: ids}
}

// List returns the trainer's teams by name.
func (t *Teams) List(ctx context.Context, trainer domain.TrainerID) ([]domain.Team, error) {
	return t.teams.ListTeams(ctx, trainer)
}

// Create adds a team with a name no other team of the trainer uses.
func (t *Teams) Create(ctx context.Context, trainer domain.TrainerID, name string, members []domain.PokemonID) (domain.Team, error) {
	team, err := domain.NewTeam(domain.TeamID(t.ids.NewID()), name, members)
	if err != nil {
		return domain.Team{}, err
	}
	return team, t.save(ctx, trainer, team)
}

// Update renames a team and replaces its members.
func (t *Teams) Update(ctx context.Context, trainer domain.TrainerID, id domain.TeamID, name string, members []domain.PokemonID) (domain.Team, error) {
	team, err := domain.NewTeam(id, name, members)
	if err != nil {
		return domain.Team{}, err
	}
	existing, err := t.teams.ListTeams(ctx, trainer)
	if err != nil {
		return domain.Team{}, err
	}
	if !containsTeam(existing, id) {
		return domain.Team{}, domain.ErrTeamNotFound
	}
	return team, t.save(ctx, trainer, team)
}

// Delete removes a team; its members lose that protection.
func (t *Teams) Delete(ctx context.Context, trainer domain.TrainerID, id domain.TeamID) error {
	return t.teams.DeleteTeam(ctx, trainer, id)
}

func (t *Teams) save(ctx context.Context, trainer domain.TrainerID, team domain.Team) error {
	existing, err := t.teams.ListTeams(ctx, trainer)
	if err != nil {
		return err
	}
	for _, other := range existing {
		if other.ID != team.ID && strings.EqualFold(other.Name, team.Name) {
			return domain.ErrTeamNameTaken
		}
	}
	if err := t.teams.SaveTeam(ctx, trainer, team); err != nil {
		return fmt.Errorf("save team: %w", err)
	}
	return nil
}

func containsTeam(teams []domain.Team, id domain.TeamID) bool {
	for _, t := range teams {
		if t.ID == id {
			return true
		}
	}
	return false
}
