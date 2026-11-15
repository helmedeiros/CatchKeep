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

// Setups is the SQLite ports.Setups.
type Setups struct {
	db *sql.DB
}

var _ ports.Setups = (*Setups)(nil)

// NewSetups returns a setup store backed by store.
func NewSetups(store *Store) *Setups {
	return &Setups{db: store.db}
}

// GetSetup implements ports.Setups.
func (r *Setups) GetSetup(ctx context.Context, trainer domain.TrainerID) (domain.Setup, error) {
	var (
		s     domain.Setup
		goals string
	)
	err := r.db.QueryRowContext(ctx, `SELECT capacity, goals FROM trainer_setups WHERE trainer_id = ?`, string(trainer)).
		Scan(&s.Capacity, &goals)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Setup{}, domain.ErrSetupNotFound
	}
	if err != nil {
		return domain.Setup{}, fmt.Errorf("get setup: %w", err)
	}
	if err := json.Unmarshal([]byte(goals), &s.Goals); err != nil {
		return domain.Setup{}, fmt.Errorf("decode goals: %w", err)
	}
	return s, nil
}

// SaveSetup implements ports.Setups.
func (r *Setups) SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error {
	goals, err := json.Marshal(s.Goals)
	if err != nil {
		return fmt.Errorf("encode goals: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO trainer_setups (trainer_id, capacity, goals) VALUES (?, ?, ?)
		ON CONFLICT (trainer_id) DO UPDATE SET capacity = excluded.capacity, goals = excluded.goals`,
		string(trainer), s.Capacity, string(goals)); err != nil {
		return fmt.Errorf("save setup: %w", err)
	}
	return nil
}
