package ports

import (
	"context"
	"io"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// ExportParser reads a collection export.
type ExportParser interface {
	// Parse returns domain.ErrUnrecognizedExport for files it cannot read.
	Parse(r io.Reader) (domain.ParsedExport, error)
}

// ImportBatches persists staged and committed imports.
type ImportBatches interface {
	Save(ctx context.Context, b domain.ImportBatch) error
	// Get returns domain.ErrImportNotFound unless the batch belongs to trainer.
	Get(ctx context.Context, trainer domain.TrainerID, id string) (domain.ImportBatch, error)
	// Commit atomically stores the batch (now committed) and adds the Pokémon.
	Commit(ctx context.Context, b domain.ImportBatch, add []domain.Pokemon) error
}

// CollectionReader reads a trainer's Pokémon.
type CollectionReader interface {
	// ListPokemon returns the trainer's active Pokémon with all observations.
	ListPokemon(ctx context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error)
}
