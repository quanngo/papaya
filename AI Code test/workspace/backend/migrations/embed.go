// Package migrations embeds the numbered SQLite migration files so the binary
// can apply them without the source tree (see server/migrate.go).
package migrations

import "embed"

// FS holds every NNN_*.sql file in this directory.
//
//go:embed *.sql
var FS embed.FS
