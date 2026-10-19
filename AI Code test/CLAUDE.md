# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

A fullstack **workspace template** (orchestration scripts, Claude Code skills, a project registry) holding the solution to **AI Challenge 15: Multi-Tenant Configuration Platform** under `workspace/`:

- `workspace/backend/`: a Go module `claimconfig` (Gin, sqlx, SQLite, swaggo). It covers password auth, RBAC, tenants with an append-only config history (rollback, diff), `processClaim`, and Swagger. Read `workspace/backend/CLAUDE.md` and `API_DOCUMENT.md` before changing it.
- `workspace/frontend/`: a React 18 + Vite admin UI in plain JS. It covers the tenant form with live backend validation, preview, diff, history, and processed claims. Read `workspace/frontend/CLAUDE.md`.

`PLAN-challenge15.md` (requirements analysis, assumptions, demo script) and `PLAN-backend.md` are the design notes. The code was written from scratch following the skills' conventions; there is no external base codebase.

The project lives in a path with a space (`F:\Papaya\AI Code test`), so quote paths in shell commands. The git repository root is `F:\Papaya` (remote `github.com/quanngo/papaya`, branch `main`), so this folder is a subdirectory of the repo; `.claude/skills/` is gitignored there and not published. Live deployment: https://papaya-2ze0.onrender.com (Render, built from this folder's `Dockerfile`; SQLite resets on restart).

## Running the app

```bash
cd workspace/frontend && npm run build                    # once, or after UI changes
cd workspace/backend && STATIC_DIR=../frontend/dist go run .   # UI + API + Swagger on http://localhost:8080
```

For UI development, run `go run .` in `workspace/backend` and `npm run dev` in `workspace/frontend` (http://localhost:5173 proxies `/api` and `/swagger`).

Docker: `docker compose up --build` in this directory builds one image from the root `Dockerfile` (Vite build → static Go binary → Alpine). It serves everything on :8080 and keeps SQLite in the `claimconfig-data` volume. `HOST_PORT` changes the host port, and `DATABASE_URL` can point at Turso. `workspace/backend/Dockerfile` builds the API alone.

- The app must be **same-origin**: session cookie plus a `sameOrigin` check on writes. Use the Vite proxy or the Go static server; never add CORS.
- Seeded logins: `admin@papaya.com` / `123456`, `operator@papaya.com` / `Operator@2026` (tenant management and claim processing), `viewer@papaya.com` / `Viewer@2026!` (read-only).
- Config: `DATABASE_URL`, `STATIC_DIR` and `PORT` env vars override the optional `config/application.yml` (see `.example`).

## Checks and tests

```bash
bash .claude/skills/backend-check/check.sh            # go mod tidy, gofmt (changed files), build, vet, test, secret scan
cd workspace/backend && go test ./component/tenantconfig/ -run TestProcessClaim_SameClaimThreeTenants -v
cd workspace/backend && scripts/dev.sh swag           # regenerate docs/ after changing swag annotations or DTOs
cd workspace/frontend && npm test && npm run build
cd workspace/frontend && npx vitest run src/pages/TenantEditPage.test.jsx -t "onboards"
cd workspace/frontend && E2E_BASE_URL=http://localhost:8080 npm run e2e   # Playwright vs a running full stack
```

- The gofmt step of `check.sh` only inspects git-changed files; run `gofmt -l` for a full check.
- `go.sum` is gitignored, so run `go mod tidy` on a fresh clone. There is no `make`: `workspace/backend/scripts/dev.sh` replaces it.
- On Windows, Node is at `C:\Program Files\nodejs`. Prepend it to `PATH` in Git Bash if `node` isn't found.

## Database

SQLite through a Go-embedded migration runner. Locally `modernc.org/sqlite` (no CGO), DB `workspace/backend/data/app.db` (gitignored); Turso via `libsql://…` in production. The DSN must include `_pragma=foreign_keys(1)`.

- `go run . migrate --dry-run` lists applied and pending files; the server also auto-migrates on start. Applied files are recorded in `schema_migrations`.
- The `db-migrate` skill and its `migrate.sh` target the Postgres project the skills were written for. **Do not use them here.**
- Migrations `001`–`006`:
  - auth/RBAC tables;
  - roles and permissions (`users.*`, `tenants.*`, `claims.*`);
  - `tenant` plus the append-only `tenant_config_version` (sequence and immutability triggers, deferred FK for `current_version`);
  - `claim`;
  - seed users;
  - rename of the old `@mq.com` seed users.

  Never edit an applied file; add `NNN_*.sql`. Seed tenants (`seed/*.json`) are inserted through `TenantService.Create` only when `tenant` is empty.

## Workspace scripts (template, not used for this task)

`setup.sh` (clone repos and regenerate CLAUDE.md files), `start.sh` (PM2) and `deliver.sh` (Jira ticket → plan pipeline) read `projects.env`, which still lists placeholder `org/...` repos and expects code at `workspace/<type>/<key>/`. The code lives directly in `workspace/backend` and `workspace/frontend`, where the skills expect it.

- Don't run `setup.sh`: step 3 **overwrites this CLAUDE.md** with headless `claude` and splits project CLAUDE.md files using `--dangerously-skip-permissions`.
- In `deliver.sh`, only the `ticket` and `plan` phases exist.

## Cross-cutting conventions (from `.claude/skills/write-code-*`)

Read the relevant skill before implementing. The skills also describe an e-commerce codebase (marketplaces, workers, Google OIDC, Postgres) that does not exist here; follow only the conventions.

- **Envelope**: every endpoint answers HTTP 200 `{code, message, error_text?, data: [...]}`, with the real status in `code` and a stable code in `message`. The FE's `apiRequest` returns `data[0]` and throws `ApiError`. Never echo `err.Error()` for bind errors.
- **Backend layers**: `component/models` (snake_case JSON) → `repositories` (raw SQL, missing row = `(nil, nil)`) → `services` (interface + `*Impl` over small private store interfaces; `AppError` codes) → `handlers/rest` (routes in `rest.go`, `RequireAuth` + `RequirePermission(models.Perm…)` + `sameOrigin` on writes). Wiring is in `server/base.go` `Initialize`.
- **Config dimensions are modules**: a new one (e.g. currency) = a field on `models.TenantConfig`, a `component/tenantconfig/mod_<name>.go`, one line in `Modules`, i18n keys, and a FE section registered in `pages/tenant-sections/index.js`. `ProcessClaim` is the only engine, used by both preview and runtime.
- **i18n vi + en everywhere**:
  - Backend: `error.*`, `issue.*`, `trace.*`, `role.*`, `permission.*` in `i18n/locales`.
  - Frontend: `src/locales`.
  - Parity tests on both sides fail on a missing key. Codes never change with the language.
- **Frontend**: plain JS/JSX (JSDoc for shapes), no state, component or CSS libraries, CSS modules with `var(--…)` tokens; only the UI language code may be stored in `localStorage`.
- **Tests**: Go integration tests start the real app on a temp SQLite file and call it over HTTP (`component/handlers/rest/api_test.go`). FE tests use an in-memory MSW backend with the real envelope (`src/test/server.js`) and `renderApp('/route', { as: email })`. E2E specs skip without `E2E_BASE_URL`.
