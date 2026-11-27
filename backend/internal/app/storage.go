package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// StorageRecorder notes how full storage is after a change.
type StorageRecorder interface {
	RecordStorage(ctx context.Context, trainer domain.TrainerID, eventID string)
}

// Storage keeps storage snapshots in the history log (spec §13).
type Storage struct {
	keep    *Keep
	history ports.History
}

// NewStorage wires the storage snapshots.
func NewStorage(keep *Keep, history ports.History) *Storage {
	return &Storage{keep: keep, history: history}
}

// RecordStorage attaches how full the trainer's storage is now to their
// event. The change itself is already saved, so a failure is logged, not
// returned.
func (s *Storage) RecordStorage(ctx context.Context, trainer domain.TrainerID, eventID string) {
	snapshot, err := s.keep.Storage(ctx, trainer)
	if err == nil {
		err = s.history.AttachStorage(ctx, trainer, eventID, snapshot)
	}
	if err != nil {
		slog.WarnContext(ctx, "storage snapshot not saved", "event", eventID, "error", err)
	}
}

// Storage is how full the trainer's storage is now.
func (k *Keep) Storage(ctx context.Context, trainer domain.TrainerID) (domain.StorageSnapshot, error) {
	setup, pokemon, err := k.inputs(ctx, trainer)
	if errors.Is(err, domain.ErrSetupNotFound) {
		pokemon, err = k.collection.ListPokemon(ctx, trainer)
		return domain.StorageSnapshot{Active: len(pokemon)}, err
	}
	if err != nil {
		return domain.StorageSnapshot{}, err
	}
	return domain.SnapshotStorage(setup, domain.SummarizeKeeps(pokemon, setup, k.species.Category)), nil
}
