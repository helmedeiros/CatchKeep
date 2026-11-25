package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Collection is the SQLite ports.ImportBatches and ports.CollectionReader.
type Collection struct {
	db *sql.DB
}

var (
	_ ports.ImportBatches    = (*Collection)(nil)
	_ ports.CollectionReader = (*Collection)(nil)
	_ ports.CollectionWriter = (*Collection)(nil)
)

// NewCollection returns a collection store backed by store.
func NewCollection(store *Store) *Collection {
	return &Collection{db: store.db}
}

// Save implements ports.ImportBatches.
func (c *Collection) Save(ctx context.Context, b domain.ImportBatch) error {
	parsed, err := json.Marshal(b.Parsed)
	if err != nil {
		return fmt.Errorf("encode import: %w", err)
	}
	resolutions, err := encodeResolutions(b)
	if err != nil {
		return err
	}
	table, err := nullableJSON(b.Table, b.Table != nil)
	if err != nil {
		return fmt.Errorf("encode import table: %w", err)
	}
	mapping, err := nullableJSON(b.Mapping, b.Mapping != nil)
	if err != nil {
		return fmt.Errorf("encode column mapping: %w", err)
	}
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO import_batches (id, trainer_id, status, parsed, resolutions, new_count, updated_count, possible_match_count, created_at, committed_at,
			source_table, column_mapping)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, string(b.TrainerID), string(b.Status), string(parsed), resolutions, b.NewCount, b.UpdatedCount, b.PossibleMatchCount,
		formatTime(b.CreatedAt), nullableTime(b.CommittedAt), table, mapping)
	if err != nil {
		return fmt.Errorf("insert import: %w", err)
	}
	return nil
}

// Get implements ports.ImportBatches.
func (c *Collection) Get(ctx context.Context, trainer domain.TrainerID, id string) (domain.ImportBatch, error) {
	var (
		b                 domain.ImportBatch
		trainerID, status string
		parsed, createdAt string
		resolutions       string
		committedAt       sql.NullString
		table, mapping    sql.NullString
	)
	err := c.db.QueryRowContext(ctx, `
		SELECT id, trainer_id, status, parsed, resolutions, new_count, updated_count, possible_match_count, created_at, committed_at,
			source_table, column_mapping
		FROM import_batches WHERE id = ? AND trainer_id = ?`, id, string(trainer)).
		Scan(&b.ID, &trainerID, &status, &parsed, &resolutions, &b.NewCount, &b.UpdatedCount, &b.PossibleMatchCount, &createdAt, &committedAt,
			&table, &mapping)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ImportBatch{}, domain.ErrImportNotFound
	}
	if err != nil {
		return domain.ImportBatch{}, fmt.Errorf("get import: %w", err)
	}
	b.TrainerID, b.Status = domain.TrainerID(trainerID), domain.ImportStatus(status)
	if err := json.Unmarshal([]byte(parsed), &b.Parsed); err != nil {
		return domain.ImportBatch{}, fmt.Errorf("decode import: %w", err)
	}
	if err := json.Unmarshal([]byte(resolutions), &b.Resolutions); err != nil {
		return domain.ImportBatch{}, fmt.Errorf("decode import answers: %w", err)
	}
	if table.Valid {
		if err := json.Unmarshal([]byte(table.String), &b.Table); err != nil {
			return domain.ImportBatch{}, fmt.Errorf("decode import table: %w", err)
		}
	}
	if mapping.Valid {
		if err := json.Unmarshal([]byte(mapping.String), &b.Mapping); err != nil {
			return domain.ImportBatch{}, fmt.Errorf("decode column mapping: %w", err)
		}
	}
	if b.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.ImportBatch{}, err
	}
	if committedAt.Valid {
		if b.CommittedAt, err = parseTime(committedAt.String); err != nil {
			return domain.ImportBatch{}, err
		}
	}
	return b, nil
}

// SaveResolutions implements ports.ImportBatches.
func (c *Collection) SaveResolutions(ctx context.Context, b domain.ImportBatch) error {
	resolutions, err := encodeResolutions(b)
	if err != nil {
		return err
	}
	if _, err := c.db.ExecContext(ctx, `UPDATE import_batches SET resolutions = ? WHERE id = ? AND trainer_id = ?`,
		resolutions, b.ID, string(b.TrainerID)); err != nil {
		return fmt.Errorf("update import answers: %w", err)
	}
	return nil
}

// SaveMapping implements ports.ImportBatches in a single transaction.
func (c *Collection) SaveMapping(ctx context.Context, b domain.ImportBatch) error {
	if b.Table == nil {
		return errors.New("save mapping: the import has no table")
	}
	parsed, err := json.Marshal(b.Parsed)
	if err != nil {
		return fmt.Errorf("encode import: %w", err)
	}
	resolutions, err := encodeResolutions(b)
	if err != nil {
		return err
	}
	mapping, err := json.Marshal(b.Mapping)
	if err != nil {
		return fmt.Errorf("encode column mapping: %w", err)
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE import_batches SET parsed = ?, resolutions = ?, column_mapping = ?, new_count = ?, updated_count = ?, possible_match_count = ?
		WHERE id = ? AND trainer_id = ?`,
		string(parsed), resolutions, string(mapping), b.NewCount, b.UpdatedCount, b.PossibleMatchCount, b.ID, string(b.TrainerID)); err != nil {
		return fmt.Errorf("update import mapping: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO column_mappings (trainer_id, signature, mapping) VALUES (?, ?, ?)
		ON CONFLICT (trainer_id, signature) DO UPDATE SET mapping = excluded.mapping`,
		string(b.TrainerID), b.Table.Signature(), string(mapping)); err != nil {
		return fmt.Errorf("remember mapping: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SavedMapping implements ports.ImportBatches.
func (c *Collection) SavedMapping(ctx context.Context, trainer domain.TrainerID, signature string) (domain.ColumnMapping, error) {
	var raw string
	err := c.db.QueryRowContext(ctx, `SELECT mapping FROM column_mappings WHERE trainer_id = ? AND signature = ?`,
		string(trainer), signature).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get saved mapping: %w", err)
	}
	var m domain.ColumnMapping
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("decode saved mapping: %w", err)
	}
	return m, nil
}

func nullableJSON(v any, present bool) (sql.NullString, error) {
	if !present {
		return sql.NullString{}, nil
	}
	out, err := json.Marshal(v)
	return sql.NullString{String: string(out), Valid: err == nil}, err
}

func encodeResolutions(b domain.ImportBatch) (string, error) {
	if b.Resolutions == nil {
		return "{}", nil
	}
	out, err := json.Marshal(b.Resolutions)
	if err != nil {
		return "", fmt.Errorf("encode import answers: %w", err)
	}
	return string(out), nil
}

// Commit implements ports.ImportBatches in a single transaction. Updates
// only touch the batch trainer's active Pokémon.
func (c *Collection) Commit(ctx context.Context, b domain.ImportBatch, add []domain.Pokemon, update []domain.PokemonUpdate, e domain.ImportEvent) (err error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	recordedAt := formatTime(b.CommittedAt)
	if _, err = tx.ExecContext(ctx, `
		UPDATE import_batches SET status = ?, new_count = ?, updated_count = ?, possible_match_count = ?, committed_at = ?
		WHERE id = ? AND trainer_id = ?`,
		string(b.Status), b.NewCount, b.UpdatedCount, b.PossibleMatchCount, nullableTime(b.CommittedAt), b.ID, string(b.TrainerID)); err != nil {
		return fmt.Errorf("update import: %w", err)
	}
	insertPokemon, err := tx.PrepareContext(ctx, `INSERT INTO pokemon (id, trainer_id, created_batch_id) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = insertPokemon.Close() }()
	insertObs, err := tx.PrepareContext(ctx, insertObservation)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = insertObs.Close() }()
	for _, p := range add {
		if _, err = insertPokemon.ExecContext(ctx, string(p.ID), string(p.TrainerID), b.ID); err != nil {
			return fmt.Errorf("insert pokemon: %w", err)
		}
		for _, o := range p.Observations {
			if _, err = insertObs.ExecContext(ctx, string(p.ID), string(o.Field), o.Value, string(o.Source),
				o.SourceRef, formatTime(o.ObservedAt), recordedAt); err != nil {
				return fmt.Errorf("insert observation: %w", err)
			}
		}
	}
	if err = addObservations(ctx, tx, insertObs, b.TrainerID, update, recordedAt); err != nil {
		return err
	}
	if err = insertImportEvent(ctx, tx, e); err != nil {
		return err
	}
	if err = bumpRevision(ctx, tx, b.TrainerID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SetProtected implements ports.CollectionWriter.
func (c *Collection) SetProtected(ctx context.Context, trainer domain.TrainerID, id domain.PokemonID, protected bool, change domain.Change) error {
	return inTx(ctx, c.db, trainer, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE pokemon SET manual_protected = ?
			WHERE id = ? AND trainer_id = ? AND status = 'active'`, protected, string(id), string(trainer))
		if err != nil {
			return fmt.Errorf("set protected: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("set protected: %w", err)
		} else if n == 0 {
			return domain.ErrPokemonNotFound
		}
		return insertChange(ctx, tx, change, protectionDetail{Protected: protected})
	})
}

// AddObservations implements ports.CollectionWriter in a single transaction.
func (c *Collection) AddObservations(ctx context.Context, trainer domain.TrainerID, updates []domain.PokemonUpdate) (err error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	insertObs, err := tx.PrepareContext(ctx, insertObservation)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = insertObs.Close() }()
	if err = addObservations(ctx, tx, insertObs, trainer, updates, ""); err != nil {
		return err
	}
	if err = bumpRevision(ctx, tx, trainer); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

const insertObservation = `
	INSERT INTO observations (pokemon_id, field, value, source, source_ref, observed_at, recorded_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

// addObservations adds observations to the trainer's active Pokémon, failing
// if any is not theirs. An empty recordedAt records each at its ObservedAt.
func addObservations(ctx context.Context, tx *sql.Tx, insertObs *sql.Stmt, trainer domain.TrainerID, updates []domain.PokemonUpdate, recordedAt string) error {
	for _, u := range updates {
		var owned bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pokemon WHERE id = ? AND trainer_id = ? AND status = 'active')`,
			string(u.PokemonID), string(trainer)).Scan(&owned); err != nil {
			return fmt.Errorf("check pokemon: %w", err)
		}
		if !owned {
			return fmt.Errorf("update pokemon %s: not in the trainer's collection", u.PokemonID)
		}
		for _, o := range u.Observations {
			at := recordedAt
			if at == "" {
				at = formatTime(o.ObservedAt)
			}
			if _, err := insertObs.ExecContext(ctx, string(u.PokemonID), string(o.Field), o.Value, string(o.Source),
				o.SourceRef, formatTime(o.ObservedAt), at); err != nil {
				return fmt.Errorf("insert observation: %w", err)
			}
		}
	}
	return nil
}

// ListPokemon implements ports.CollectionReader.
func (c *Collection) ListPokemon(ctx context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT p.id, p.manual_protected, o.field, o.value, o.source, o.source_ref, o.observed_at
		FROM pokemon p LEFT JOIN observations o ON o.pokemon_id = p.id
		WHERE p.trainer_id = ? AND p.status = 'active'
		ORDER BY p.id, o.id`, string(trainer))
	if err != nil {
		return nil, fmt.Errorf("list pokemon: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Pokemon
	for rows.Next() {
		var (
			id                                  string
			protected                           bool
			field, value, source, ref, observed sql.NullString
		)
		if err := rows.Scan(&id, &protected, &field, &value, &source, &ref, &observed); err != nil {
			return nil, fmt.Errorf("scan pokemon: %w", err)
		}
		if len(out) == 0 || out[len(out)-1].ID != domain.PokemonID(id) {
			out = append(out, domain.Pokemon{ID: domain.PokemonID(id), TrainerID: trainer, Protected: protected})
		}
		if !field.Valid {
			continue
		}
		at, err := parseTime(observed.String)
		if err != nil {
			return nil, err
		}
		p := &out[len(out)-1]
		p.Observations = append(p.Observations, domain.Observation{
			Field: domain.Field(field.String), Value: value.String, Source: domain.Source(source.String),
			SourceRef: ref.String, ObservedAt: at,
		})
	}
	return out, rows.Err()
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatTime(t)
}

// Revision implements ports.Revisions.
func (c *Collection) Revision(ctx context.Context, trainer domain.TrainerID) (int64, error) {
	var rev int64
	if err := c.db.QueryRowContext(ctx, `SELECT revision FROM trainers WHERE id = ?`, string(trainer)).Scan(&rev); err != nil {
		return 0, fmt.Errorf("read revision: %w", err)
	}
	return rev, nil
}

// DeleteStagedBefore removes every trainer's unfinished imports started
// before cutoff, and returns how many.
func (c *Collection) DeleteStagedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	res, err := c.db.ExecContext(ctx,
		`DELETE FROM import_batches WHERE status = 'staged' AND julianday(created_at) < julianday(?)`, formatTime(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete old imports: %w", err)
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// TrimStaged keeps only the trainer's keep newest unfinished imports.
func (c *Collection) TrimStaged(ctx context.Context, trainer domain.TrainerID, keep int) error {
	_, err := c.db.ExecContext(ctx, `
		DELETE FROM import_batches WHERE trainer_id = ? AND status = 'staged' AND id NOT IN (
			SELECT id FROM import_batches WHERE trainer_id = ? AND status = 'staged'
			ORDER BY julianday(created_at) DESC, id DESC LIMIT ?)`,
		string(trainer), string(trainer), keep)
	if err != nil {
		return fmt.Errorf("trim imports: %w", err)
	}
	return nil
}
