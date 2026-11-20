package ports

import (
	"context"
	"io"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// ExportParser reads a collection export.
type ExportParser interface {
	// Parse returns domain.ErrUnrecognizedExport for files it cannot read.
	Parse(r io.Reader) (domain.ParsedExport, error)
}

// TableParser reads files that are not a recognized export, so the trainer
// can map their columns.
type TableParser interface {
	// ReadTable returns domain.ErrUnrecognizedExport for files with no
	// columns or no rows.
	ReadTable(r io.Reader) (domain.Table, error)
	ParseTable(t domain.Table, m domain.ColumnMapping, observedAt time.Time) domain.ParsedExport
}

// ImportBatches persists staged and committed imports.
type ImportBatches interface {
	Save(ctx context.Context, b domain.ImportBatch) error
	// Get returns domain.ErrImportNotFound unless the batch belongs to trainer.
	Get(ctx context.Context, trainer domain.TrainerID, id string) (domain.ImportBatch, error)
	// SaveResolutions stores the batch's duplicate review answers.
	SaveResolutions(ctx context.Context, b domain.ImportBatch) error
	// SaveMapping atomically stores the batch's mapping, rows, answers and
	// counts, and remembers the mapping for the trainer's files with the
	// same columns.
	SaveMapping(ctx context.Context, b domain.ImportBatch) error
	// SavedMapping returns the trainer's mapping for files with this
	// signature, or nil.
	SavedMapping(ctx context.Context, trainer domain.TrainerID, signature string) (domain.ColumnMapping, error)
	// Commit atomically stores the batch (now committed), adds the new
	// Pokémon and adds observations to the trainer's existing ones.
	Commit(ctx context.Context, b domain.ImportBatch, add []domain.Pokemon, update []domain.PokemonUpdate) error
}

// CollectionWriter changes a trainer's Pokémon.
type CollectionWriter interface {
	// AddObservations atomically adds observations to the trainer's active
	// Pokémon; it fails without changes if any Pokémon is not theirs.
	AddObservations(ctx context.Context, trainer domain.TrainerID, updates []domain.PokemonUpdate) error
	// SetProtected sets the trainer's manual protection of one of their
	// active Pokémon, or returns domain.ErrPokemonNotFound.
	SetProtected(ctx context.Context, trainer domain.TrainerID, id domain.PokemonID, protected bool) error
}

// CollectionReader reads a trainer's Pokémon.
type CollectionReader interface {
	// ListPokemon returns the trainer's active Pokémon with all observations.
	ListPokemon(ctx context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error)
}
