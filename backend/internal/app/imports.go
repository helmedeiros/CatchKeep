package app

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Import states, re-exported for callers of this package.
const (
	ImportStaged    = domain.ImportStaged
	ImportCommitted = domain.ImportCommitted
)

// ImportPreview is what the trainer reviews before and after committing.
type ImportPreview struct {
	ID                 string
	Status             domain.ImportStatus
	Source             string
	TotalRows          int
	NewCount           int
	PossibleMatchCount int
	Problems           []domain.ImportProblem
	IgnoredColumns     []string
	UnknownFields      []domain.Field
}

// Imports stages exports for review and commits them to the collection.
type Imports struct {
	parser     ports.ExportParser
	batches    ports.ImportBatches
	collection ports.CollectionReader
	ids        ports.IDs
	clock      ports.Clock
}

// NewImports wires the import use cases.
func NewImports(parser ports.ExportParser, batches ports.ImportBatches, collection ports.CollectionReader, ids ports.IDs, clock ports.Clock) *Imports {
	return &Imports{parser: parser, batches: batches, collection: collection, ids: ids, clock: clock}
}

// Stage reads an export and stores it for review. The collection is unchanged.
func (u *Imports) Stage(ctx context.Context, trainer domain.TrainerID, r io.Reader) (ImportPreview, error) {
	parsed, err := u.parser.Parse(r)
	if err != nil {
		return ImportPreview{}, err
	}
	existing, err := u.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return ImportPreview{}, err
	}
	fresh, possible := domain.ClassifyImport(existing, parsed.Rows)
	batch := domain.ImportBatch{
		ID: u.ids.NewID(), TrainerID: trainer, Status: domain.ImportStaged, Parsed: parsed,
		NewCount: len(fresh), PossibleMatchCount: possible, CreatedAt: u.clock.Now(),
	}
	if err := u.batches.Save(ctx, batch); err != nil {
		return ImportPreview{}, fmt.Errorf("save import: %w", err)
	}
	return previewOf(batch), nil
}

// Get returns the trainer's import.
func (u *Imports) Get(ctx context.Context, trainer domain.TrainerID, id string) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	return previewOf(batch), nil
}

// Commit adds the import's new Pokémon. Rows are classified again against
// the collection as it is now, so imports committed in between are respected.
func (u *Imports) Commit(ctx context.Context, trainer domain.TrainerID, id string) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	if batch.Status == domain.ImportCommitted {
		return ImportPreview{}, domain.ErrImportAlreadyCommitted
	}
	existing, err := u.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return ImportPreview{}, err
	}
	fresh, possible := domain.ClassifyImport(existing, batch.Parsed.Rows)
	add := make([]domain.Pokemon, 0, len(fresh))
	for _, row := range fresh {
		add = append(add, domain.Pokemon{ID: domain.PokemonID(u.ids.NewID()), TrainerID: trainer, Observations: row.Observations})
	}
	batch.Status, batch.NewCount, batch.PossibleMatchCount, batch.CommittedAt = domain.ImportCommitted, len(fresh), possible, u.clock.Now()
	if err := u.batches.Commit(ctx, batch, add); err != nil {
		return ImportPreview{}, fmt.Errorf("commit import: %w", err)
	}
	return previewOf(batch), nil
}

func previewOf(b domain.ImportBatch) ImportPreview {
	skipped := 0
	for _, p := range b.Parsed.Problems {
		if p.Skipped {
			skipped++
		}
	}
	return ImportPreview{
		ID: b.ID, Status: b.Status, Source: b.Parsed.Source,
		TotalRows: len(b.Parsed.Rows) + skipped, NewCount: b.NewCount, PossibleMatchCount: b.PossibleMatchCount,
		Problems: b.Parsed.Problems, IgnoredColumns: b.Parsed.IgnoredColumns, UnknownFields: b.Parsed.UnknownFields,
	}
}

// Collection reads a trainer's Pokémon.
type Collection struct {
	reader ports.CollectionReader
}

// NewCollection wires the collection use cases.
func NewCollection(reader ports.CollectionReader) *Collection {
	return &Collection{reader: reader}
}

// List returns the trainer's Pokémon, highest CP first, unknown CP last.
func (c *Collection) List(ctx context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error) {
	list, err := c.reader.ListPokemon(ctx, trainer)
	if err != nil {
		return nil, err
	}
	cp := func(p domain.Pokemon) int {
		if v, ok := p.Current().Int(domain.FieldCP); ok {
			return v
		}
		return -1
	}
	slices.SortStableFunc(list, func(a, b domain.Pokemon) int { return cmp.Compare(cp(b), cp(a)) })
	return list, nil
}
