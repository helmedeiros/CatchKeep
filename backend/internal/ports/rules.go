package ports

import (
	"context"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// Setups persists each trainer's setup.
type Setups interface {
	// GetSetup returns domain.ErrSetupNotFound until the trainer saves one.
	GetSetup(ctx context.Context, trainer domain.TrainerID) (domain.Setup, error)
	// SaveSetup stores the setup and records c in the history.
	SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup, c domain.Change) error
}

// Species is the bundled reference data about species.
type Species interface {
	// Category names a species' category, or "" if it is in none.
	Category(species string) domain.Category
}

// Teams persists each trainer's named teams.
type Teams interface {
	// ListTeams returns the trainer's teams by name.
	ListTeams(ctx context.Context, trainer domain.TrainerID) ([]domain.Team, error)
	// SaveTeam creates or replaces a team and records c in the history.
	// It returns domain.ErrPokemonNotFound if a member is not one of the
	// trainer's active Pokémon and domain.ErrTeamNotFound if the ID belongs
	// to another trainer.
	SaveTeam(ctx context.Context, trainer domain.TrainerID, t domain.Team, c domain.Change) error
	// DeleteTeam removes a team and records c in the history. It returns
	// domain.ErrTeamNotFound unless the team is theirs.
	DeleteTeam(ctx context.Context, trainer domain.TrainerID, id domain.TeamID, c domain.Change) error
}
