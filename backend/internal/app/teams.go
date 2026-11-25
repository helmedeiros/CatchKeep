package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Teams manages a trainer's named teams, whose members are protected.
type Teams struct {
	teams      ports.Teams
	collection ports.CollectionReader
	ids        ports.IDs
	clock      ports.Clock
}

// NewTeams wires the team use cases.
func NewTeams(teams ports.Teams, collection ports.CollectionReader, ids ports.IDs, clock ports.Clock) *Teams {
	return &Teams{teams: teams, collection: collection, ids: ids, clock: clock}
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
	existing, err := t.teams.ListTeams(ctx, trainer)
	if err != nil {
		return domain.Team{}, err
	}
	return team, t.save(ctx, trainer, existing, team)
}

// Update renames a team and replaces its members; saving it unchanged
// changes nothing.
func (t *Teams) Update(ctx context.Context, trainer domain.TrainerID, id domain.TeamID, name string, members []domain.PokemonID) (domain.Team, error) {
	team, err := domain.NewTeam(id, name, members)
	if err != nil {
		return domain.Team{}, err
	}
	existing, err := t.teams.ListTeams(ctx, trainer)
	if err != nil {
		return domain.Team{}, err
	}
	i := slices.IndexFunc(existing, func(x domain.Team) bool { return x.ID == id })
	if i < 0 {
		return domain.Team{}, domain.ErrTeamNotFound
	}
	if existing[i].Name == team.Name && slices.Equal(existing[i].Members, team.Members) {
		return team, nil
	}
	return team, t.save(ctx, trainer, existing, team)
}

// Delete removes a team; its members lose that protection.
func (t *Teams) Delete(ctx context.Context, trainer domain.TrainerID, id domain.TeamID) error {
	existing, err := t.teams.ListTeams(ctx, trainer)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(existing, func(x domain.Team) bool { return x.ID == id })
	if i < 0 {
		return domain.ErrTeamNotFound
	}
	change, err := t.change(ctx, trainer, existing[i])
	if err != nil {
		return err
	}
	return t.teams.DeleteTeam(ctx, trainer, id, change)
}

func (t *Teams) save(ctx context.Context, trainer domain.TrainerID, existing []domain.Team, team domain.Team) error {
	for _, other := range existing {
		if other.ID != team.ID && strings.EqualFold(other.Name, team.Name) {
			return domain.ErrTeamNameTaken
		}
	}
	change, err := t.change(ctx, trainer, team)
	if err != nil {
		return err
	}
	if err := t.teams.SaveTeam(ctx, trainer, team, change); err != nil {
		return fmt.Errorf("save team: %w", err)
	}
	return nil
}

// change records a change to team, naming its members.
func (t *Teams) change(ctx context.Context, trainer domain.TrainerID, team domain.Team) (domain.Change, error) {
	owned, err := t.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return domain.Change{}, err
	}
	var members []domain.Pokemon
	for _, id := range team.Members {
		if i := slices.IndexFunc(owned, func(p domain.Pokemon) bool { return p.ID == id }); i >= 0 {
			members = append(members, owned[i])
		}
	}
	return domain.NewChange(t.ids.NewID(), trainer, domain.ChangeTeam, members, t.clock.Now()), nil
}
