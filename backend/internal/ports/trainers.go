package ports

import (
	"context"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// Trainers persists trainer profiles.
type Trainers interface {
	// Create stores a new trainer. It returns domain.ErrDisplayNameTaken if
	// another trainer has the same display name, ignoring case.
	Create(ctx context.Context, t domain.Trainer) error
	// Get returns domain.ErrTrainerNotFound if no trainer has the id.
	Get(ctx context.Context, id domain.TrainerID) (domain.Trainer, error)
	// List returns every trainer.
	List(ctx context.Context) ([]domain.Trainer, error)
	// SetPINHash replaces a trainer's PIN hash, or returns
	// domain.ErrTrainerNotFound.
	SetPINHash(ctx context.Context, id domain.TrainerID, hash string) error
}

// PINHasher hashes and verifies PINs.
type PINHasher interface {
	Hash(pin string) (string, error)
	Verify(hash, pin string) (bool, error)
}

// IDs generates new unique identifiers.
type IDs interface {
	NewID() string
}

// Clock tells the current time.
type Clock interface {
	Now() time.Time
}

// Sessions persists sign-in sessions keyed by the hash of their token.
type Sessions interface {
	Create(ctx context.Context, s domain.Session) error
	// Get returns domain.ErrNoSession if no session has the token hash.
	Get(ctx context.Context, tokenHash string) (domain.Session, error)
	Touch(ctx context.Context, tokenHash string, at time.Time) error
	Delete(ctx context.Context, tokenHash string) error
	// DeleteForTrainer ends every session of the trainer.
	DeleteForTrainer(ctx context.Context, trainer domain.TrainerID) error
}

// Tokens generates unguessable session tokens.
type Tokens interface {
	NewToken() string
}
