package app

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
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
	UpdatedCount       int
	PossibleMatchCount int
	// Matches are the possible duplicates and their answers; staged only.
	Matches        []domain.PossibleMatch
	Problems       []domain.ImportProblem
	IgnoredColumns []string
	UnknownFields  []domain.Field
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
	batch := domain.ImportBatch{
		ID: u.ids.NewID(), TrainerID: trainer, Status: domain.ImportStaged, Parsed: parsed, CreatedAt: u.clock.Now(),
	}
	review, err := u.review(ctx, batch)
	if err != nil {
		return ImportPreview{}, err
	}
	if err := u.batches.Save(ctx, withCounts(batch, review)); err != nil {
		return ImportPreview{}, fmt.Errorf("save import: %w", err)
	}
	return stagedPreview(batch, review), nil
}

// Get returns the trainer's import. A staged import is reviewed against the
// collection as it is now.
func (u *Imports) Get(ctx context.Context, trainer domain.TrainerID, id string) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	if batch.Status == domain.ImportCommitted {
		return previewOf(batch), nil
	}
	review, err := u.review(ctx, batch)
	if err != nil {
		return ImportPreview{}, err
	}
	return stagedPreview(batch, review), nil
}

// Resolve records the trainer's answers for possible duplicates: the same
// Pokémon as a candidate, or a different one. Later answers for a line
// replace earlier ones.
func (u *Imports) Resolve(ctx context.Context, trainer domain.TrainerID, id string, answers []domain.Resolution) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	if batch.Status == domain.ImportCommitted {
		return ImportPreview{}, domain.ErrImportAlreadyCommitted
	}
	review, err := u.review(ctx, batch)
	if err != nil {
		return ImportPreview{}, err
	}
	resolutions := maps.Clone(batch.Resolutions)
	if resolutions == nil {
		resolutions = make(map[int]domain.Resolution, len(answers))
	}
	for _, a := range answers {
		if err := review.Validate(a); err != nil {
			return ImportPreview{}, err
		}
		resolutions[a.Line] = a
	}
	batch.Resolutions = resolutions
	if err := u.batches.SaveResolutions(ctx, batch); err != nil {
		return ImportPreview{}, fmt.Errorf("save answers: %w", err)
	}
	if review, err = u.review(ctx, batch); err != nil {
		return ImportPreview{}, err
	}
	return stagedPreview(batch, review), nil
}

// Commit adds the import's new Pokémon and updates the ones the trainer
// answered as the same. Rows are reviewed again against the collection as it
// is now, so imports committed in between are respected; unanswered possible
// duplicates are held back.
func (u *Imports) Commit(ctx context.Context, trainer domain.TrainerID, id string) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	if batch.Status == domain.ImportCommitted {
		return ImportPreview{}, domain.ErrImportAlreadyCommitted
	}
	review, err := u.review(ctx, batch)
	if err != nil {
		return ImportPreview{}, err
	}
	add := make([]domain.Pokemon, 0, len(review.Add))
	for _, row := range review.Add {
		add = append(add, domain.Pokemon{ID: domain.PokemonID(u.ids.NewID()), TrainerID: trainer, Observations: row.Observations})
	}
	batch = withCounts(batch, review)
	batch.Status, batch.CommittedAt = domain.ImportCommitted, u.clock.Now()
	if err := u.batches.Commit(ctx, batch, add, review.Update); err != nil {
		return ImportPreview{}, fmt.Errorf("commit import: %w", err)
	}
	return previewOf(batch), nil
}

func (u *Imports) review(ctx context.Context, b domain.ImportBatch) (domain.ImportReview, error) {
	existing, err := u.collection.ListPokemon(ctx, b.TrainerID)
	if err != nil {
		return domain.ImportReview{}, err
	}
	return domain.ReviewImport(existing, b.Parsed.Rows, b.Resolutions), nil
}

func withCounts(b domain.ImportBatch, r domain.ImportReview) domain.ImportBatch {
	b.NewCount, b.UpdatedCount, b.PossibleMatchCount = len(r.Add), len(r.Update), r.Pending()
	return b
}

func stagedPreview(b domain.ImportBatch, r domain.ImportReview) ImportPreview {
	p := previewOf(withCounts(b, r))
	p.Matches = r.Matches
	return p
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
		TotalRows: len(b.Parsed.Rows) + skipped, NewCount: b.NewCount, UpdatedCount: b.UpdatedCount,
		PossibleMatchCount: b.PossibleMatchCount,
		Problems:           b.Parsed.Problems, IgnoredColumns: b.Parsed.IgnoredColumns, UnknownFields: b.Parsed.UnknownFields,
	}
}

// Collection reads and confirms a trainer's Pokémon.
type Collection struct {
	reader ports.CollectionReader
	writer ports.CollectionWriter
	clock  ports.Clock
}

// NewCollection wires the collection use cases.
func NewCollection(reader ports.CollectionReader, writer ports.CollectionWriter, clock ports.Clock) *Collection {
	return &Collection{reader: reader, writer: writer, clock: clock}
}

// ConfirmAbsent records that none of the trainer's Pokémon of one species
// and form has the given features, and returns how many Pokémon changed.
func (c *Collection) ConfirmAbsent(ctx context.Context, trainer domain.TrainerID, species, form string, fields []domain.Field) (int, error) {
	list, err := c.reader.ListPokemon(ctx, trainer)
	if err != nil {
		return 0, err
	}
	updates, err := domain.ConfirmAbsent(list, species, form, fields, c.clock.Now())
	if err != nil || len(updates) == 0 {
		return 0, err
	}
	if err := c.writer.AddObservations(ctx, trainer, updates); err != nil {
		return 0, fmt.Errorf("confirm features: %w", err)
	}
	return len(updates), nil
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
