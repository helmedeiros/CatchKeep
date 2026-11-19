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

// speciesRule is the stored form of a domain.SpeciesRule.
type speciesRule struct {
	Species  string `json:"species"`
	Minimum  int    `json:"minimum,omitempty"`
	Favorite bool   `json:"favorite,omitempty"`
}

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
		s              domain.Setup
		goals, species string
	)
	err := r.db.QueryRowContext(ctx, `SELECT capacity, goals, species FROM trainer_setups WHERE trainer_id = ?`, string(trainer)).
		Scan(&s.Capacity, &goals, &species)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Setup{}, domain.ErrSetupNotFound
	}
	if err != nil {
		return domain.Setup{}, fmt.Errorf("get setup: %w", err)
	}
	if err := json.Unmarshal([]byte(goals), &s.Goals); err != nil {
		return domain.Setup{}, fmt.Errorf("decode goals: %w", err)
	}
	var rules []speciesRule
	if err := json.Unmarshal([]byte(species), &rules); err != nil {
		return domain.Setup{}, fmt.Errorf("decode species rules: %w", err)
	}
	for _, sr := range rules {
		s.Species = append(s.Species, domain.SpeciesRule(sr))
	}
	return s, nil
}

// SaveSetup implements ports.Setups.
func (r *Setups) SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error {
	goals, err := json.Marshal(s.Goals)
	if err != nil {
		return fmt.Errorf("encode goals: %w", err)
	}
	rules := make([]speciesRule, 0, len(s.Species))
	for _, sr := range s.Species {
		rules = append(rules, speciesRule(sr))
	}
	species, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("encode species rules: %w", err)
	}
	return inTx(ctx, r.db, trainer, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO trainer_setups (trainer_id, capacity, goals, species) VALUES (?, ?, ?, ?)
			ON CONFLICT (trainer_id) DO UPDATE SET capacity = excluded.capacity, goals = excluded.goals, species = excluded.species`,
			string(trainer), s.Capacity, string(goals), string(species)); err != nil {
			return fmt.Errorf("save setup: %w", err)
		}
		return nil
	})
}
