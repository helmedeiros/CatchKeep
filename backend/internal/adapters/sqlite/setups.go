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

// categoryRule is the stored form of a domain.CategoryRule.
type categoryRule struct {
	Category domain.Category `json:"category"`
	Minimum  int             `json:"minimum,omitempty"`
	Favorite bool            `json:"favorite,omitempty"`
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
		s                          domain.Setup
		goals, species, categories string
	)
	err := r.db.QueryRowContext(ctx, `SELECT capacity, goals, species, categories, in_game_count, storage_limit FROM trainer_setups WHERE trainer_id = ?`, string(trainer)).
		Scan(&s.Capacity, &goals, &species, &categories, &s.InGameCount, &s.Limit)
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
	var groups []categoryRule
	if err := json.Unmarshal([]byte(categories), &groups); err != nil {
		return domain.Setup{}, fmt.Errorf("decode category rules: %w", err)
	}
	for _, cr := range groups {
		s.Categories = append(s.Categories, domain.CategoryRule(cr))
	}
	return s, nil
}

// SaveSetup implements ports.Setups.
func (r *Setups) SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup, c domain.Change) error {
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
	groups := make([]categoryRule, 0, len(s.Categories))
	for _, cr := range s.Categories {
		groups = append(groups, categoryRule(cr))
	}
	categories, err := json.Marshal(groups)
	if err != nil {
		return fmt.Errorf("encode category rules: %w", err)
	}
	return inTx(ctx, r.db, trainer, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO trainer_setups (trainer_id, capacity, goals, species, categories, in_game_count, storage_limit) VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (trainer_id) DO UPDATE SET capacity = excluded.capacity, goals = excluded.goals,
				species = excluded.species, categories = excluded.categories,
				in_game_count = excluded.in_game_count, storage_limit = excluded.storage_limit`,
			string(trainer), s.Capacity, string(goals), string(species), string(categories), s.InGameCount, s.Limit); err != nil {
			return fmt.Errorf("save setup: %w", err)
		}
		if c.Kind == domain.ChangeRules {
			return insertChange(ctx, tx, c, rulesDetail{Species: rules, Categories: groups})
		}
		return insertChange(ctx, tx, c, setupDetail{Capacity: s.Capacity, Goals: s.Goals, InGameCount: s.InGameCount, Limit: s.Limit})
	})
}

// setupDetail is the storage and goals a setup change saved.
type setupDetail struct {
	Capacity    int           `json:"capacity"`
	Goals       []domain.Goal `json:"goals"`
	InGameCount int           `json:"inGameCount,omitempty"`
	Limit       int           `json:"limit,omitempty"`
}

// rulesDetail is the rules a rule change saved.
type rulesDetail struct {
	Species    []speciesRule  `json:"species"`
	Categories []categoryRule `json:"categories"`
}
