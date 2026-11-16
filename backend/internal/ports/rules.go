package ports

import (
	"context"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// Setups persists each trainer's setup.
type Setups interface {
	// GetSetup returns domain.ErrSetupNotFound until the trainer saves one.
	GetSetup(ctx context.Context, trainer domain.TrainerID) (domain.Setup, error)
	SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error
}

// Species is the bundled reference data about species.
type Species interface {
	// Rare reports legendary, mythical and Ultra Beast species.
	Rare(species string) bool
}
