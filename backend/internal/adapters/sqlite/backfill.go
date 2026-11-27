package sqlite

import (
	"context"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

var _ ports.PastImports = (*Collection)(nil)

// UnrecordedImports implements ports.PastImports. An import's observations
// were all recorded at its commit time, so they tell what it added and
// changed; everything recorded earlier is how the Pokémon were before it.
func (c *Collection) UnrecordedImports(ctx context.Context) ([]domain.PastImport, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT b.id, b.trainer_id, b.committed_at FROM import_batches b
		WHERE b.status = 'committed' AND NOT EXISTS (
			SELECT 1 FROM history_events h
			WHERE h.trainer_id = b.trainer_id AND h.kind = 'import' AND json_extract(h.detail, '$.batchId') = b.id)
		ORDER BY julianday(b.committed_at), b.id`)
	if err != nil {
		return nil, fmt.Errorf("find past imports: %w", err)
	}
	var past []domain.PastImport
	for rows.Next() {
		var (
			p                 domain.PastImport
			trainer, recorded string
		)
		if err := rows.Scan(&p.BatchID, &trainer, &recorded); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan past import: %w", err)
		}
		if p.CommittedAt, err = parseTime(recorded); err != nil {
			_ = rows.Close()
			return nil, err
		}
		p.TrainerID = domain.TrainerID(trainer)
		past = append(past, p)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range past {
		if err := c.rebuild(ctx, &past[i]); err != nil {
			return nil, err
		}
	}
	return past, nil
}

// rebuild fills in what p added and changed, and the Pokémon before it.
func (c *Collection) rebuild(ctx context.Context, p *domain.PastImport) error {
	rows, err := c.db.QueryContext(ctx, `
		SELECT pk.id, COALESCE(pk.created_batch_id, '') = ?, o.recorded_at = ?,
			o.field, o.value, o.source, o.source_ref, o.observed_at
		FROM pokemon pk JOIN observations o ON o.pokemon_id = pk.id
		WHERE pk.trainer_id = ? AND julianday(o.recorded_at) <= julianday(?)
		ORDER BY pk.id, o.id`, p.BatchID, formatTime(p.CommittedAt), string(p.TrainerID), formatTime(p.CommittedAt))
	if err != nil {
		return fmt.Errorf("rebuild past import: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			id                                  string
			created, now                        bool
			field, value, source, ref, observed string
		)
		if err := rows.Scan(&id, &created, &now, &field, &value, &source, &ref, &observed); err != nil {
			return fmt.Errorf("scan past import: %w", err)
		}
		at, err := parseTime(observed)
		if err != nil {
			return err
		}
		pid := domain.PokemonID(id)
		o := domain.Observation{Field: domain.Field(field), Value: value, Source: domain.Source(source), SourceRef: ref, ObservedAt: at}
		switch {
		case created:
			p.Added = withObservation(p.Added, p.TrainerID, pid, o)
		case now:
			if n := len(p.Updates); n == 0 || p.Updates[n-1].PokemonID != pid {
				p.Updates = append(p.Updates, domain.PokemonUpdate{PokemonID: pid})
			}
			u := &p.Updates[len(p.Updates)-1]
			u.Observations = append(u.Observations, o)
		default:
			p.Existing = withObservation(p.Existing, p.TrainerID, pid, o)
		}
	}
	return rows.Err()
}

// withObservation adds o to Pokémon id, the last in list or a new one; rows come
// grouped by Pokémon.
func withObservation(list []domain.Pokemon, trainer domain.TrainerID, id domain.PokemonID, o domain.Observation) []domain.Pokemon {
	if n := len(list); n == 0 || list[n-1].ID != id {
		list = append(list, domain.Pokemon{ID: id, TrainerID: trainer})
	}
	last := &list[len(list)-1]
	last.Observations = append(last.Observations, o)
	return list
}

// RecordImport implements ports.PastImports. It changes no Pokémon, so the
// revision stays.
func (c *Collection) RecordImport(ctx context.Context, e domain.ImportEvent) (err error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = insertImportEvent(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit()
}
