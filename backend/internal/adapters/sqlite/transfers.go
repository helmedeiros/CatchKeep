package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Transfers is the SQLite ports.Transfers.
type Transfers struct {
	db *sql.DB
}

var _ ports.Transfers = (*Transfers)(nil)

// NewTransfers returns a transfer store backed by store.
func NewTransfers(store *Store) *Transfers {
	return &Transfers{db: store.db}
}

// Record implements ports.Transfers.
func (r *Transfers) Record(ctx context.Context, e domain.TransferEvent, expectedRevision int64) (domain.TransferEvent, error) {
	if earlier, ok, err := r.FindByRequest(ctx, e.TrainerID, e.RequestID); err != nil || ok {
		return earlier, err
	}
	err := inTx(ctx, r.db, e.TrainerID, func(tx *sql.Tx) error {
		if expectedRevision >= 0 {
			var current int64
			if err := tx.QueryRowContext(ctx, `SELECT revision FROM trainers WHERE id = ?`, string(e.TrainerID)).Scan(&current); err != nil {
				return fmt.Errorf("read revision: %w", err)
			}
			if current != expectedRevision {
				return domain.ErrStaleRevision
			}
		}
		var gone []domain.PokemonID
		for _, p := range e.Pokemon {
			res, err := tx.ExecContext(ctx, `UPDATE pokemon SET status = 'transferred' WHERE id = ? AND trainer_id = ? AND status = 'active'`,
				string(p.ID), string(e.TrainerID))
			if err != nil {
				return fmt.Errorf("mark transferred: %w", err)
			}
			if n, err := res.RowsAffected(); err != nil {
				return fmt.Errorf("mark transferred: %w", err)
			} else if n == 0 {
				gone = append(gone, p.ID)
			}
		}
		if len(gone) > 0 {
			return &domain.NotActiveError{IDs: gone}
		}
		if err := countInGame(ctx, tx, e.TrainerID, -len(e.Pokemon)); err != nil {
			return err
		}
		return insertEvent(ctx, tx, e)
	})
	if err != nil {
		return domain.TransferEvent{}, err
	}
	return e, nil
}

// Undo implements ports.Transfers.
func (r *Transfers) Undo(ctx context.Context, trainer domain.TrainerID, eventID string, undo domain.TransferEvent) (domain.TransferEvent, error) {
	err := inTx(ctx, r.db, trainer, func(tx *sql.Tx) error {
		original, err := scanEvent(tx.QueryRowContext(ctx, selectEvent+` WHERE id = ? AND trainer_id = ?`, eventID, string(trainer)))
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return domain.ErrTransferNotFound
		case err != nil:
			return err
		case original.Kind == domain.TransferUndo:
			return fmt.Errorf("%w: an undo cannot be undone; confirm the transfer again instead", domain.ErrInvalidTransfer)
		case original.UndoneBy != "":
			return domain.ErrAlreadyUndone
		}
		for _, p := range original.Pokemon {
			if _, err := tx.ExecContext(ctx, `UPDATE pokemon SET status = 'active' WHERE id = ? AND trainer_id = ? AND status = 'transferred'`,
				string(p.ID), string(trainer)); err != nil {
				return fmt.Errorf("restore: %w", err)
			}
		}
		if err := countInGame(ctx, tx, trainer, len(original.Pokemon)); err != nil {
			return err
		}
		undo.Pokemon, undo.Undoes = original.Pokemon, original.ID
		if err := insertEvent(ctx, tx, undo); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE history_events SET undone_by = ? WHERE id = ?`, undo.ID, original.ID); err != nil {
			return fmt.Errorf("mark undone: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.TransferEvent{}, err
	}
	return undo, nil
}

// ListTransfers implements ports.Transfers.
func (r *Transfers) ListTransfers(ctx context.Context, trainer domain.TrainerID, limit int) ([]domain.TransferEvent, error) {
	rows, err := r.db.QueryContext(ctx, selectEvent+` WHERE trainer_id = ? ORDER BY created_at DESC, rowid DESC LIMIT ?`, string(trainer), limit)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.TransferEvent
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FindByRequest implements ports.Transfers.
func (r *Transfers) FindByRequest(ctx context.Context, trainer domain.TrainerID, requestID string) (domain.TransferEvent, bool, error) {
	e, err := scanEvent(r.db.QueryRowContext(ctx, selectEvent+` WHERE trainer_id = ? AND request_id = ?`, string(trainer), requestID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TransferEvent{}, false, nil
	}
	return e, err == nil, err
}

// countInGame keeps the trainer's in-game count in step with what they
// transferred, so the next plan starts from the storage left. An unknown
// count (0) stays unknown.
func countInGame(ctx context.Context, tx *sql.Tx, trainer domain.TrainerID, delta int) error {
	if _, err := tx.ExecContext(ctx, `UPDATE trainer_setups SET in_game_count = MAX(0, in_game_count + ?) WHERE trainer_id = ? AND in_game_count > 0`,
		delta, string(trainer)); err != nil {
		return fmt.Errorf("update in-game count: %w", err)
	}
	return nil
}

const selectEvent = `SELECT id, trainer_id, kind, pokemon, COALESCE(request_id, ''), COALESCE(undoes, ''), COALESCE(undone_by, ''), created_at FROM history_events`

type transferred struct {
	ID      string `json:"id"`
	Species string `json:"species"`
	CP      int    `json:"cp,omitempty"`
	HasCP   bool   `json:"hasCp,omitempty"`
}

func insertEvent(ctx context.Context, tx *sql.Tx, e domain.TransferEvent) error {
	list := make([]transferred, len(e.Pokemon))
	for i, p := range e.Pokemon {
		list[i] = transferred{ID: string(p.ID), Species: p.Species, CP: p.CP, HasCP: p.HasCP}
	}
	pokemon, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("encode transfer: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO history_events (id, trainer_id, kind, pokemon, request_id, undoes, created_at) VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)`,
		e.ID, string(e.TrainerID), string(e.Kind), string(pokemon), e.RequestID, e.Undoes, formatTime(e.At)); err != nil {
		return fmt.Errorf("insert transfer: %w", err)
	}
	return nil
}

func scanEvent(s scanner) (domain.TransferEvent, error) {
	var (
		e                      domain.TransferEvent
		trainer, kind, pokemon string
		created                string
	)
	if err := s.Scan(&e.ID, &trainer, &kind, &pokemon, &e.RequestID, &e.Undoes, &e.UndoneBy, &created); err != nil {
		return domain.TransferEvent{}, err
	}
	var list []transferred
	if err := json.Unmarshal([]byte(pokemon), &list); err != nil {
		return domain.TransferEvent{}, fmt.Errorf("decode transfer: %w", err)
	}
	for _, p := range list {
		e.Pokemon = append(e.Pokemon, domain.Transferred{ID: domain.PokemonID(p.ID), Species: p.Species, CP: p.CP, HasCP: p.HasCP})
	}
	at, err := parseTime(created)
	if err != nil {
		return domain.TransferEvent{}, err
	}
	e.TrainerID, e.Kind, e.At = domain.TrainerID(trainer), domain.TransferKind(kind), at
	return e, nil
}
