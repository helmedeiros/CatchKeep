package sqlite

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// MigrateTo applies migrations up to version, to test upgrading data.
func MigrateTo(ctx context.Context, db *sql.DB, version int64) error {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, sub)
	if err != nil {
		return err
	}
	_, err = provider.UpTo(ctx, version)
	return err
}
