package sqlite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
)

func openTemp(t *testing.T) (*sqlite.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catchkeep.db")
	store, err := sqlite.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, path
}

func TestOpen_AppliesAllMigrations(t *testing.T) {
	store, _ := openTemp(t)

	version, err := store.SchemaVersion(t.Context())
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != sqlite.LatestSchemaVersion() {
		t.Fatalf("schema version = %d, want %d", version, sqlite.LatestSchemaVersion())
	}
}

func TestOpen_EnablesWALAndForeignKeys(t *testing.T) {
	store, _ := openTemp(t)
	db := store.DB()

	var mode string
	if err := db.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	var fk int
	if err := db.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" || fk != 1 {
		t.Fatalf("journal_mode=%q foreign_keys=%d, want wal and 1", mode, fk)
	}
}

func TestOpen_ReopeningAnExistingDatabaseIsIdempotent(t *testing.T) {
	store, path := openTemp(t)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := sqlite.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = again.Close() }()
	version, err := again.SchemaVersion(t.Context())
	if err != nil || version != sqlite.LatestSchemaVersion() {
		t.Fatalf("after reopen: version %d, err %v", version, err)
	}
}

func TestPing_ReportsHealthyDatabase(t *testing.T) {
	store, _ := openTemp(t)

	if err := store.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestOpen_KeepsTheDatabaseFilesToTheirOwner(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "older.db")
	if err := os.WriteFile(older, nil, 0o644); err != nil { //nolint:gosec // an install from before files were private
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "new.db"), older} {
		store, err := sqlite.Open(t.Context(), path)
		if err != nil {
			t.Fatalf("Open %s: %v", path, err)
		}
		if _, err := store.DB().ExecContext(t.Context(), `CREATE TABLE IF NOT EXISTS probe (x)`); err != nil {
			t.Fatal(err)
		}
		for _, f := range []string{path, path + "-wal", path + "-shm"} {
			info, err := os.Stat(f)
			if err != nil {
				t.Fatalf("stat %s: %v", f, err)
			}
			if mode := info.Mode().Perm(); mode != 0o600 {
				t.Errorf("%s mode = %o, want 600", filepath.Base(f), mode)
			}
		}
		_ = store.Close()
	}
}
