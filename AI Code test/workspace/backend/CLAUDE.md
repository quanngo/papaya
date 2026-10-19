# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Backend of AI Challenge 15 (multi-tenant claim configuration). Go module `claimconfig`: Gin + sqlx + SQLite (`modernc.org/sqlite` locally, Turso via `libsql` in production) + swaggo. The API contract is in `API_DOCUMENT.md`; Swagger UI is served at `/swagger/index.html`.

## Commands

```bash
go run .                         # serve on :8080 (auto-migrate + seed); Swagger at /swagger/index.html
go run . migrate --dry-run       # list applied/pending migrations; `go run . migrate` applies
echo 'long-password' | go run . create-user --email a@papaya.com --name "A" --role operator --password-stdin
go test ./...
go test ./component/tenantconfig/ -run TestProcessClaim_SameClaimThreeTenants -v
scripts/dev.sh swag              # regenerate docs/ after changing any swag annotation or DTO
scripts/dev.sh check             # swag + stale-docs check + vet + test
scripts/demo.sh [base-url]       # end-to-end demo against a running server
bash ../../.claude/skills/backend-check/check.sh   # pre-commit gate (from repo root: bash .claude/skills/backend-check/check.sh)
```

There is no `make`; `scripts/dev.sh` replaces it. `go.sum` is gitignored, so run `go mod tidy` on a fresh clone. Config comes from `config/application.yml` (see `.example`; optional), `config/.env`, and the env vars `DATABASE_URL`/`PORT`, which always win.

## Architecture

- `main.go` holds the Swagger general info and calls `server.Main` (subcommands `serve`, `migrate`, `create-user`).
- `server/base.go` `Initialize` opens the DB, runs `Migrate` (embedded `migrations/*.sql`, tracked in `schema_migrations`), seeds `seed/*.json` through `TenantService.Create` when `tenant` is empty, wires repos → services → `rest.APIRoutes`, and serves the built frontend from `server.static-dir` with an SPA fallback (same origin is required by the cookie session).
- `component/tenantconfig` is the **pure domain** (no gin/sqlx). Each config section is a `Module` (`Validate`, `Process`, `DiffKeys`) listed in `registry.go` in processing order (claim type and custom fields decide acceptance before approval, SLA and notifications). `ProcessClaim` is the single engine used by both preview and runtime. `Diff` walks the JSON form; `DiffKeys` makes tiers match by `role` and custom fields by `key`.
  - Adding a config dimension (e.g. currency): a field on `models.TenantConfig`, a `mod_<name>.go`, one line in `Modules`, and its `issue.*`/`trace.*` keys in both locale files.
- `component/services` hold business rules and stable error codes (`AppError` in `auth_errors.go`/`tenant_errors.go`; `ConfigInvalidError` carries issues). They depend on small private store interfaces.
- `component/repositories`: raw SQL with `?` placeholders. Saving a config is `AppendVersion`: insert version `base+1`, then `UPDATE … WHERE current_version = base`. A stale base, the DB's sequence trigger, or UNIQUE all map to `ErrConflict` → `VERSION_CONFLICT`. The local DB uses one connection (SQLite single writer); `foreign_keys(1)` must be in the DSN (`Open` refuses otherwise).
- `component/handlers/rest`: `rest.go` mounts every route with `RequireAuth`, `RequirePermission(models.Perm…)` and `sameOrigin`. Handlers only call `api.Ok` / `api.ErrorWithMessage` / `respondError` and fill translated text via `localize.go` (codes never change with the language).

## Rules specific to this codebase

- Validation issues and engine errors carry a **code + params**, never text. Every code needs `issue.<CODE>` (or `trace.<CODE>`, `error.<CODE>`) in **both** `i18n/locales/vi.json` and `en.json`; `i18n_test.go` and `services/errors_i18n_test.go` fail otherwise.
- `bindJSON` rejects unknown fields; keep DTO JSON tags snake_case and give new DTO fields `example:` tags for Swagger.
- `tenant_config_version` is append-only and `tenant` cannot be deleted (DB triggers). Rollback = a new version; archive = `archived_at`.
- Timestamps in SQLite are `utils.FormatTime` (UTC, ms, `Z`), so string comparison is time order.
- Integration tests (`component/handlers/rest/api_test.go`) start the real app on a temp SQLite file and log in with the seeded accounts (`admin@papaya.com`/`123456`, `operator@papaya.com`/`Operator@2026`, `viewer@papaya.com`/`Viewer@2026!`).
