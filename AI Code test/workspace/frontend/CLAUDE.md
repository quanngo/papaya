# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Admin UI of AI Challenge 15 (multi-tenant claim configuration). React 18 + Vite, **plain JavaScript/JSX**: no TypeScript, no state/component/CSS libraries. It talks to the Go backend in `../backend` (contract: `../backend/API_DOCUMENT.md`).

## Commands

```bash
npm run dev                  # http://localhost:5173, proxies /api and /swagger to VITE_BACKEND_URL (default http://localhost:8080)
npm test                     # Vitest + Testing Library + MSW (jsdom)
npx vitest run src/pages/TenantEditPage.test.jsx -t "onboards"
npm run build                # dist/, served by the backend: STATIC_DIR=../frontend/dist go run . (in ../backend)
E2E_BASE_URL=http://localhost:8080 npm run e2e   # Playwright against a real backend serving dist/ (skips without the env var)
```

On Windows, Node is at `C:\Program Files\nodejs`. Playwright needs `npx playwright install chromium` once.

## Architecture

- **Same origin is required.** The session is an HttpOnly cookie and the backend rejects cross-origin writes, so dev uses the Vite proxy (`changeOrigin: false`), and production has the Go server serve `dist/`. Never add CORS or call another origin.
- `src/api/client.js` `apiRequest` is the only place that calls `fetch`. The backend always answers HTTP 200 `{code, message, error_text, data[]}`; `apiRequest` returns `data[0]` or throws `ApiError {status, code, errorText, data}` (`data` holds `{issues}` for `CONFIG_INVALID`). Per-area functions live in `src/api/<area>.js`, with JSDoc typedefs of every shape.
- `src/App.jsx` holds the routes without a router (`main.jsx` adds `BrowserRouter`, tests `MemoryRouter`). Signed-in pages sit inside `ProtectedRoute` → `AppLayout`; each page route gets `<ProtectedRoute permission="…">`. Permission codes come from `/auth/me` (`useAuth().can`).
- **Tenant form** (`pages/TenantEditPage.jsx`): one config object in state, edited through `update(path, value)` / `replace(fn)` (`tenant/configState.js`). Each backend config module has a section in `pages/tenant-sections/` registered in `SECTIONS` (a new config dimension = one section file + one line). Validation is **not** reimplemented: the form posts to `/configs/validate` (debounced, re-run on language change) and shows issues by backend path (`approval.tiers[1].min_amount`) via `issuesAt`. Tier minimums are derived (`chainTiers`) so tiers stay contiguous. The route component keys the page by id, so state never leaks from `/tenants/new` to the created tenant.
- Preview and runtime use the backend engine: `/claims/preview` (nothing stored, can send an unsaved config) and `/claims/process-batch` (stored). `tenant/ResultCard.jsx` renders a ProcessResult; `tenant/ChangesTable.jsx` renders diffs (also used by history).

## Conventions

- Every UI string goes in **both** `src/locales/vi.json` and `en.json` (`locales.test.js` checks parity); dates and numbers via `src/i18n/format.js`. Errors are kept as codes in state and translated at render with `actionErrorMessage(t, err, context)` (`errors.<context>.<CODE>` → `errors.common.<CODE>` → backend `error_text`). Validation issue, result and trace texts come translated from the backend in the user's locale.
- Styling: CSS modules with `var(--…)` tokens from `src/styles/global.css` (light and dark); shared classes in `components/ui.module.css`. Layouts must work at phone width.
- Only the UI language code (`claimconfig-admin-lang`) may be stored in `localStorage`.
- Tests: `src/test/server.js` is an in-memory MSW backend with the real envelope (toggles on `backend`, reset before each test). Render through `renderApp('/route', { as: 'operator@papaya.com' })` and assert redirects with `getByTestId('location')`. Tests run in Vietnamese.
