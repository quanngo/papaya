package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso (production)
	_ "modernc.org/sqlite"                               // local file, no CGO
)

// ErrConflict means a concurrent write won: a stale base version, or a
// duplicate version number rejected by the trigger / UNIQUE constraint.
var ErrConflict = errors.New("conflict")

// ErrDuplicate means a UNIQUE constraint failed (e.g. tenant slug taken).
var ErrDuplicate = errors.New("duplicate")

// Open connects to DATABASE_URL: "file:…" uses the local SQLite driver,
// "libsql://…" or "https://…" uses Turso.
func Open(url string) (*sqlx.DB, error) {
	if strings.HasPrefix(url, "libsql://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		db, err := sqlx.Open("libsql", url)
		if err != nil {
			return nil, err
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
		return db, db.Ping()
	}
	db, err := sqlx.Open("sqlite", url)
	if err != nil {
		return nil, err
	}
	// SQLite allows one writer; a single connection avoids "database is
	// locked" and makes the per-connection foreign_keys pragma apply everywhere.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	var fk int
	if err := db.Get(&fk, "PRAGMA foreign_keys"); err != nil || fk != 1 {
		db.Close()
		return nil, errors.New("foreign keys are off: add _pragma=foreign_keys(1) to DATABASE_URL")
	}
	return db, nil
}

// WithTx runs fn in a transaction, rolling back on error.
func WithTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// isUnique reports a UNIQUE / PRIMARY KEY violation (both drivers put the
// SQLite message in the error text).
func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// isVersionRace reports the tenant_config_version sequence trigger firing.
func isVersionRace(err error) bool {
	return err != nil && strings.Contains(err.Error(), "version must be previous max + 1")
}
