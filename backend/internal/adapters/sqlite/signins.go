package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// SignInLog is the SQLite ports.SignInLog, in a table of its own.
type SignInLog struct {
	db *sql.DB
}

var _ ports.SignInLog = (*SignInLog)(nil)

// NewSignInLog returns a sign-in log backed by store.
func NewSignInLog(store *Store) *SignInLog {
	return &SignInLog{db: store.db}
}

// Record implements ports.SignInLog.
func (l *SignInLog) Record(ctx context.Context, e domain.SignInEntry) error {
	if _, err := l.db.ExecContext(ctx, `INSERT INTO sign_in_log (kind, trainer_id, device, created_at) VALUES (?, NULLIF(?, ''), ?, ?)`,
		string(e.Kind), string(e.TrainerID), e.Device, formatTime(e.At)); err != nil {
		return fmt.Errorf("record sign-in: %w", err)
	}
	return nil
}

// List implements ports.SignInLog.
func (l *SignInLog) List(ctx context.Context, limit int) ([]domain.SignInEntry, error) {
	rows, err := l.db.QueryContext(ctx, `
		SELECT kind, COALESCE(trainer_id, ''), device, created_at FROM sign_in_log
		ORDER BY julianday(created_at) DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list sign-ins: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.SignInEntry
	for rows.Next() {
		var (
			e                      domain.SignInEntry
			kind, trainer, created string
		)
		if err := rows.Scan(&kind, &trainer, &e.Device, &created); err != nil {
			return nil, fmt.Errorf("scan sign-in: %w", err)
		}
		if e.At, err = parseTime(created); err != nil {
			return nil, err
		}
		e.Kind, e.TrainerID = domain.SignInKind(kind), domain.TrainerID(trainer)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteBefore implements ports.SignInLog.
func (l *SignInLog) DeleteBefore(ctx context.Context, cutoff time.Time) (int, error) {
	res, err := l.db.ExecContext(ctx, `DELETE FROM sign_in_log WHERE julianday(created_at) < julianday(?)`, formatTime(cutoff))
	if err != nil {
		return 0, fmt.Errorf("sweep sign-in log: %w", err)
	}
	n, err := res.RowsAffected()
	return int(n), err
}
