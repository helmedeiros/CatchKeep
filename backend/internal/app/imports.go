package app

import (
	"bytes"
	"cmp"
	"context"
	"errors"
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
	// NeedsMapping is set for a file CatchKeep does not recognize until
	// the trainer maps its columns. Columns and Mapping (the current one,
	// or a suggestion) are set for any such file.
	NeedsMapping bool
	Columns      []ColumnPreview
	Mapping      domain.ColumnMapping
}

// ColumnPreview is one column of an unrecognized file with a few values.
type ColumnPreview struct {
	Name    string
	Samples []string
}

const columnSamples = 3

// Imports stages exports for review and commits them to the collection.
type Imports struct {
	parser     ports.ExportParser
	tables     ports.TableParser
	batches    ports.ImportBatches
	collection ports.CollectionReader
	ids        ports.IDs
	clock      ports.Clock
}

// NewImports wires the import use cases.
func NewImports(parser ports.ExportParser, tables ports.TableParser, batches ports.ImportBatches, collection ports.CollectionReader, ids ports.IDs, clock ports.Clock) *Imports {
	return &Imports{parser: parser, tables: tables, batches: batches, collection: collection, ids: ids, clock: clock}
}

// Stage reads an export and stores it for review. The collection is
// unchanged. A file that is not a recognized export is staged as a table:
// with the trainer's saved mapping for its columns if there is one,
// otherwise waiting for one (spec V1-IMP-02).
func (u *Imports) Stage(ctx context.Context, trainer domain.TrainerID, r io.Reader) (ImportPreview, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return ImportPreview{}, fmt.Errorf("read export: %w", err)
	}
	batch := domain.ImportBatch{ID: u.ids.NewID(), TrainerID: trainer, Status: domain.ImportStaged, CreatedAt: u.clock.Now()}
	batch.Parsed, err = u.parser.Parse(bytes.NewReader(data))
	if errors.Is(err, domain.ErrUnrecognizedExport) {
		err = u.stageTable(ctx, &batch, data)
	}
	if err != nil {
		return ImportPreview{}, err
	}
	if batch.NeedsMapping() {
		if err := u.batches.Save(ctx, batch); err != nil {
			return ImportPreview{}, fmt.Errorf("save import: %w", err)
		}
		return previewOf(batch), nil
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

func (u *Imports) stageTable(ctx context.Context, batch *domain.ImportBatch, data []byte) error {
	table, err := u.tables.ReadTable(bytes.NewReader(data))
	if err != nil {
		return err
	}
	batch.Table, batch.Parsed = &table, domain.ParsedExport{Source: domain.MappedSource}
	saved, err := u.batches.SavedMapping(ctx, batch.TrainerID, table.Signature())
	if err != nil {
		return err
	}
	if m := saved.Rebind(table); len(saved) > 0 && m.Validate(table) == nil {
		batch.Mapping = m
		batch.Parsed = u.tables.ParseTable(table, m, batch.CreatedAt)
	}
	return nil
}

// Map reads a staged table with the trainer's column mapping, replacing any
// earlier mapping and answers, and remembers it for files with the same
// columns.
func (u *Imports) Map(ctx context.Context, trainer domain.TrainerID, id string, m domain.ColumnMapping) (ImportPreview, error) {
	batch, err := u.batches.Get(ctx, trainer, id)
	if err != nil {
		return ImportPreview{}, err
	}
	switch {
	case batch.Status == domain.ImportCommitted:
		return ImportPreview{}, domain.ErrImportAlreadyCommitted
	case batch.Table == nil:
		return ImportPreview{}, fmt.Errorf("%w: this export is read without a mapping", domain.ErrInvalidMapping)
	}
	if err := m.Validate(*batch.Table); err != nil {
		return ImportPreview{}, err
	}
	batch.Mapping, batch.Resolutions = m, nil
	batch.Parsed = u.tables.ParseTable(*batch.Table, m, batch.CreatedAt)
	review, err := u.review(ctx, batch)
	if err != nil {
		return ImportPreview{}, err
	}
	batch = withCounts(batch, review)
	if err := u.batches.SaveMapping(ctx, batch); err != nil {
		return ImportPreview{}, fmt.Errorf("save mapping: %w", err)
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
	if batch.Status == domain.ImportCommitted || batch.NeedsMapping() {
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
	if err := ready(batch); err != nil {
		return ImportPreview{}, err
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
	if err := ready(batch); err != nil {
		return ImportPreview{}, err
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

// ready rejects imports that are committed or not mapped yet.
func ready(b domain.ImportBatch) error {
	switch {
	case b.Status == domain.ImportCommitted:
		return domain.ErrImportAlreadyCommitted
	case b.NeedsMapping():
		return domain.ErrImportNeedsMapping
	}
	return nil
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
	p := ImportPreview{
		ID: b.ID, Status: b.Status, Source: b.Parsed.Source,
		TotalRows: len(b.Parsed.Rows) + skipped, NewCount: b.NewCount, UpdatedCount: b.UpdatedCount,
		PossibleMatchCount: b.PossibleMatchCount,
		Problems:           b.Parsed.Problems, IgnoredColumns: b.Parsed.IgnoredColumns, UnknownFields: b.Parsed.UnknownFields,
		NeedsMapping: b.NeedsMapping(), Mapping: b.Mapping,
	}
	if b.Table != nil {
		for _, h := range b.Table.Header {
			p.Columns = append(p.Columns, ColumnPreview{Name: h, Samples: b.Table.Samples(h, columnSamples)})
		}
		if p.Mapping == nil {
			p.Mapping = domain.SuggestMapping(b.Table.Header)
		}
	}
	return p
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

// Mark records the trainer's own yes or no for features of one of their
// Pokémon and returns it as it is now.
func (c *Collection) Mark(ctx context.Context, trainer domain.TrainerID, id domain.PokemonID, marks map[domain.Field]bool) (domain.Pokemon, error) {
	list, err := c.reader.ListPokemon(ctx, trainer)
	if err != nil {
		return domain.Pokemon{}, err
	}
	i := slices.IndexFunc(list, func(p domain.Pokemon) bool { return p.ID == id })
	if i < 0 {
		return domain.Pokemon{}, domain.ErrPokemonNotFound
	}
	u, err := domain.MarkFeatures(list[i], marks, c.clock.Now())
	if err != nil {
		return domain.Pokemon{}, err
	}
	if err := c.writer.AddObservations(ctx, trainer, []domain.PokemonUpdate{u}); err != nil {
		return domain.Pokemon{}, fmt.Errorf("mark features: %w", err)
	}
	p := list[i]
	p.Observations = append(p.Observations, u.Observations...)
	return p, nil
}

// Protect sets or clears the trainer's manual protection of one Pokémon.
func (c *Collection) Protect(ctx context.Context, trainer domain.TrainerID, id domain.PokemonID, protected bool) error {
	return c.writer.SetProtected(ctx, trainer, id, protected)
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
