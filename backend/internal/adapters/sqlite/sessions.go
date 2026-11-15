package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Sessions is the SQLite ports.Sessions.
type Sessions struct {
	db *sql.DB
}

var _ ports.Sessions = (*Sessions)(nil)

// NewSessions returns a session store backed by store.
func NewSessions(store *Store) *Sessions {
	return &Sessions{db: store.db}
}

// Create implements ports.Sessions.
func (r *Sessions) Create(ctx context.Context, s domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, trainer_id, created_at, last_seen_at) VALUES (?, ?, ?, ?)`,
		s.TokenHash, string(s.TrainerID), formatTime(s.CreatedAt), formatTime(s.LastSeenAt))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// Get implements ports.Sessions.
func (r *Sessions) Get(ctx context.Context, tokenHash string) (domain.Session, error) {
	var (
		s                 domain.Session
		trainerID         string
		created, lastSeen string
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT token_hash, trainer_id, created_at, last_seen_at FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&s.TokenHash, &trainerID, &created, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrNoSession
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	s.TrainerID = domain.TrainerID(trainerID)
	if s.CreatedAt, err = parseTime(created); err != nil {
		return domain.Session{}, err
	}
	if s.LastSeenAt, err = parseTime(lastSeen); err != nil {
		return domain.Session{}, err
	}
	return s, nil
}

// Touch implements ports.Sessions.
func (r *Sessions) Touch(ctx context.Context, tokenHash string, at time.Time) error {
	res, err := r.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, formatTime(at), tokenHash)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return domain.ErrNoSession
	}
	return nil
}

// Delete implements ports.Sessions.
func (r *Sessions) Delete(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
