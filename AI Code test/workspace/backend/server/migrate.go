package server

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"sort"

	"claimconfig/migrations"
	"claimconfig/utils"

	"github.com/jmoiron/sqlx"
)

// Migrate applies every embedded migrations/*.sql not yet recorded in
// schema_migrations, in filename order, each in its own transaction together
// with its schema_migrations row. With dryRun it only reports.
func Migrate(ctx context.Context, db *sqlx.DB, dryRun bool, out io.Writer) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  filename   TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied := map[string]bool{}
	var done []string
	if err := db.SelectContext(ctx, &done, `SELECT filename FROM schema_migrations`); err != nil {
		return err
	}
	for _, f := range done {
		applied[f] = true
	}

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	pending := 0
	for _, name := range files {
		if applied[name] {
			fmt.Fprintf(out, "applied  %s\n", name)
			continue
		}
		pending++
		if dryRun {
			fmt.Fprintf(out, "pending  %s\n", name)
			continue
		}
		src, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(src)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (filename, applied_at) VALUES (?, ?)`,
			name, utils.FormatTime(nowFunc())); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Fprintf(out, "migrated %s\n", name)
	}
	if pending == 0 {
		fmt.Fprintln(out, "database is up to date")
	}
	return nil
}
