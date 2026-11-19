package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// bumpRevision increments the trainer's revision inside the mutation's own
// transaction (spec §5.1).
func bumpRevision(ctx context.Context, tx *sql.Tx, trainer domain.TrainerID) error {
	if _, err := tx.ExecContext(ctx, `UPDATE trainers SET revision = revision + 1 WHERE id = ?`, string(trainer)); err != nil {
		return fmt.Errorf("bump revision: %w", err)
	}
	return nil
}

// inTx runs change in a transaction that also bumps the trainer's revision,
// committing only if both succeed.
func inTx(ctx context.Context, db *sql.DB, trainer domain.TrainerID, change func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = change(tx); err != nil {
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
