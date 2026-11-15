package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Trainers is the SQLite ports.Trainers.
type Trainers struct {
	db *sql.DB
}

var _ ports.Trainers = (*Trainers)(nil)

// NewTrainers returns a trainer store backed by store.
func NewTrainers(store *Store) *Trainers {
	return &Trainers{db: store.db}
}

// Create implements ports.Trainers.
func (r *Trainers) Create(ctx context.Context, t domain.Trainer) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trainers (id, display_name, pin_hash, created_at) VALUES (?, ?, ?, ?)`,
		string(t.ID), t.DisplayName, t.PINHash, formatTime(t.CreatedAt))
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: trainers.display_name") {
		return domain.ErrDisplayNameTaken
	}
	if err != nil {
		return fmt.Errorf("insert trainer: %w", err)
	}
	return nil
}

// Get implements ports.Trainers.
func (r *Trainers) Get(ctx context.Context, id domain.TrainerID) (domain.Trainer, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, display_name, pin_hash, created_at FROM trainers WHERE id = ?`, string(id))
	t, err := scanTrainer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Trainer{}, domain.ErrTrainerNotFound
	}
	return t, err
}

// List implements ports.Trainers.
func (r *Trainers) List(ctx context.Context) ([]domain.Trainer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, display_name, pin_hash, created_at FROM trainers`)
	if err != nil {
		return nil, fmt.Errorf("list trainers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Trainer
	for rows.Next() {
		t, err := scanTrainer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTrainer(s scanner) (domain.Trainer, error) {
	var (
		t       domain.Trainer
		id      string
		created string
	)
	if err := s.Scan(&id, &t.DisplayName, &t.PINHash, &created); err != nil {
		return domain.Trainer{}, err
	}
	at, err := parseTime(created)
	if err != nil {
		return domain.Trainer{}, err
	}
	t.ID, t.CreatedAt = domain.TrainerID(id), at
	return t, nil
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}
