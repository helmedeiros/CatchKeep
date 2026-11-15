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
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO import_batches (id, trainer_id, status, parsed, resolutions, new_count, updated_count, possible_match_count, created_at, committed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, string(b.TrainerID), string(b.Status), string(parsed), resolutions, b.NewCount, b.UpdatedCount, b.PossibleMatchCount,
		formatTime(b.CreatedAt), nullableTime(b.CommittedAt))
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
	)
	err := c.db.QueryRowContext(ctx, `
		SELECT id, trainer_id, status, parsed, resolutions, new_count, updated_count, possible_match_count, created_at, committed_at
		FROM import_batches WHERE id = ? AND trainer_id = ?`, id, string(trainer)).
		Scan(&b.ID, &trainerID, &status, &parsed, &resolutions, &b.NewCount, &b.UpdatedCount, &b.PossibleMatchCount, &createdAt, &committedAt)
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
func (c *Collection) Commit(ctx context.Context, b domain.ImportBatch, add []domain.Pokemon, update []domain.PokemonUpdate) (err error) {
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
	insertObs, err := tx.PrepareContext(ctx, `
		INSERT INTO observations (pokemon_id, field, value, source, source_ref, observed_at, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
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
	for _, u := range update {
		var owned bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pokemon WHERE id = ? AND trainer_id = ? AND status = 'active')`,
			string(u.PokemonID), string(b.TrainerID)).Scan(&owned); err != nil {
			return fmt.Errorf("check pokemon: %w", err)
		}
		if !owned {
			err = fmt.Errorf("update pokemon %s: not in the trainer's collection", u.PokemonID)
			return err
		}
		for _, o := range u.Observations {
			if _, err = insertObs.ExecContext(ctx, string(u.PokemonID), string(o.Field), o.Value, string(o.Source),
				o.SourceRef, formatTime(o.ObservedAt), recordedAt); err != nil {
				return fmt.Errorf("insert observation: %w", err)
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListPokemon implements ports.CollectionReader.
func (c *Collection) ListPokemon(ctx context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT p.id, o.field, o.value, o.source, o.source_ref, o.observed_at
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
			field, value, source, ref, observed sql.NullString
		)
		if err := rows.Scan(&id, &field, &value, &source, &ref, &observed); err != nil {
			return nil, fmt.Errorf("scan pokemon: %w", err)
		}
		if len(out) == 0 || out[len(out)-1].ID != domain.PokemonID(id) {
			out = append(out, domain.Pokemon{ID: domain.PokemonID(id), TrainerID: trainer})
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
