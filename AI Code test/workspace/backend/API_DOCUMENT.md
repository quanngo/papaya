# Claim Config API Documentation

Base URL: `/api/v1` (port from `server.port` / `PORT`, default 8080). Interactive docs: `/swagger/index.html` (generated from the swag annotations in `component/handlers/rest/*.go`).

Stack: Go, Gin, sqlx, SQLite (`modernc.org/sqlite` locally, Turso in production).

## Conventions

- **Envelope**: every endpoint answers **HTTP 200** with `{code, message, error_text?, data}`. `code` is the real status, `message` is `OK` or a stable error code, `error_text` is that code translated (`vi`/`en`), and `data` is always an array; the payload is `data[0]`.
- **Auth**: an HttpOnly cookie (`session` by default) set by `POST /auth/login`. Routes marked *login* need it; *perm X* additionally needs permission X. Missing session → `401 UNAUTHENTICATED`; missing permission → `403 FORBIDDEN`.
- **Same origin**: state-changing routes reject a request whose `Origin`/`Referer` host differs from the request host with `403 CROSS_ORIGIN`.
- **Bodies** are decoded strictly: malformed JSON or an unknown field → `400 BAD_REQUEST` (the decoder error is never echoed).
- **Language**: `Accept-Language` (`vi` default), then the signed-in user's saved locale. Only texts change, never codes.
- `{id}` accepts a tenant id **or slug** everywhere.
- Unknown `/api/...` path → `404 NOT_FOUND`. Unexpected failures → `500 INTERNAL_ERROR` (details only in the server log).

## Contents

- Health: [GET /health](#get-health)
- Auth: [POST /auth/login](#post-authlogin) · [POST /auth/logout](#post-authlogout) · [POST /auth/logout-all](#post-authlogout-all) · [GET /auth/me](#get-authme) · [PUT /auth/me/locale](#put-authmelocale) · [POST /auth/change-password](#post-authchange-password)
- Configs: [GET /meta/tenant-config](#get-metatenant-config) · [GET /configs/defaults](#get-configsdefaults) · [POST /configs/validate](#post-configsvalidate)
- Tenants: [GET /tenants](#get-tenants) · [POST /tenants](#post-tenants) · [GET /tenants/diff](#get-tenantsdiff) · [GET /tenants/{id}](#get-tenantsid) · [PUT /tenants/{id}](#put-tenantsid) · [DELETE /tenants/{id}](#delete-tenantsid) · [GET /tenants/{id}/versions](#get-tenantsidversions) · [GET /tenants/{id}/versions/{version}](#get-tenantsidversionsversion) · [POST /tenants/{id}/rollback](#post-tenantsidrollback)
- Claims: [POST /claims/preview](#post-claimspreview) · [POST /claims/process](#post-claimsprocess) · [POST /claims/process-batch](#post-claimsprocess-batch) · [GET /claims](#get-claims)

## Shared shapes

**TenantConfig** (`component/models/tenant_config_model.go`)
```json
{
  "branding":      {"company_name": "SafeGuard Insurance", "logo_url": "https://…", "primary_color": "#0B3D91", "secondary_color": "#F2A900"},
  "claim_types":   {"enabled": ["OUTPATIENT", "INPATIENT"], "documents": {"OUTPATIENT": {"required": ["invoice"], "optional": ["lab_results"]}}},
  "approval":      {"auto_approval_threshold": 20000, "tiers": [{"min_amount": 20000, "max_amount": 100000, "role": "assessor"}, {"min_amount": 100000, "max_amount": null, "role": "director"}]},
  "notifications": {"webhook_url": "https://…", "events": {"claim_submitted": {"enabled": true, "channels": ["email"], "template": ""}}},
  "sla":           {"timezone": "Asia/Ho_Chi_Minh", "targets": {"OUTPATIENT": 5}, "default_days": 0, "escalation": [{"after_breach_days": 1, "notify": ["team_lead"]}]},
  "custom_fields": [{"key": "employee_id", "label": "Employee ID", "type": "text", "required": true, "pattern": "^EMP-[0-9]{5}$"}]
}
```
Enumerations: claim types `OUTPATIENT|INPATIENT|DENTAL|MATERNITY|OPTICAL`; events `claim_submitted|approved|rejected|payment_sent`; channels `email|sms|webhook`; field types `text|number|select`. Tiers are `[min, max)`; the first starts at the threshold, they must be contiguous, and only the last is unbounded.

**Issue**: `{path, code, params?, text}`, e.g. `{"path": "approval.tiers[1].min_amount", "code": "TIER_GAP", "params": {"expected": 100000}, "text": "…"}`. Codes are listed under `issue.*` in `i18n/locales/en.json`.

**Tenant**: `{id, slug, name, current_version, archived_at, created_by, created_at, updated_at}`. **TenantDetail** = Tenant + `config`.

**ProcessResult**: `{claim_id?, accepted, errors: Issue[], tenant: {id, slug, name, config_version}, claim_type, amount, documents: {required, optional, missing}, approval: {routing: auto_approved|manual_review|rejected, auto_approved, approver_role?, tier?, tier_index?, threshold}, notifications: [{event, channels, template, default_template}], sla: {business_days, timezone, submitted_at, deadline, escalation} | null, custom_fields: [{key, label, type, required, value, valid, code?, text?}], trace: [{module, code, params, text}]}`. A claim the tenant does not accept is still `code 200` with `accepted=false`, `routing=rejected`, `sla=null`, no notifications.

## Health

### GET /health
- **Description**: liveness and database ping.
- **Auth**: public
- **Request**: none
- **Response**: 200, `data[0] = {status: "ok"|"degraded", db: "ok"|"unreachable", version}`
- **Errors**: none

## Auth

### POST /auth/login
- **Description**: password login; sets the session cookie (persistent for `auth.remember-me-days` when `remember_me`, otherwise a browser-session cookie lasting `auth.session-ttl-hours` server-side).
- **Auth**: public, same origin
- **Request**: `{email, password, remember_me}` (email is case-insensitive)
- **Response**: 200, `data[0] = SessionUser {id, email, display_name, auth_method, role: {code, name}, permissions[], permission_names{}, locale}`
- **Errors**: 400 `BAD_REQUEST`; 401 `INVALID_CREDENTIALS` (also unknown, disabled or password-less accounts, constant time); 423 `ACCOUNT_LOCKED` (after `auth.max-failed-logins` failures, for `auth.lockout-minutes`); 403 `CROSS_ORIGIN`

### POST /auth/logout
- **Description**: revokes the current session and clears the cookie; always succeeds.
- **Auth**: public, same origin
- **Request**: none
- **Response**: 200, `data = []`
- **Errors**: 403 `CROSS_ORIGIN`

### POST /auth/logout-all
- **Description**: revokes every session of the user and clears this cookie.
- **Auth**: login, same origin
- **Request**: none
- **Response**: 200, `data[0] = {revoked_sessions}`
- **Errors**: 401, 403 `CROSS_ORIGIN`

### GET /auth/me
- **Description**: the signed-in user, names in the user's language.
- **Auth**: login
- **Response**: 200, `data[0] = SessionUser`
- **Errors**: 401

### PUT /auth/me/locale
- **Description**: saves the UI language on the account.
- **Auth**: login, same origin
- **Request**: `{locale: "vi"|"en"}`
- **Response**: 200, `data[0] = SessionUser` (in the new language)
- **Errors**: 400 `BAD_REQUEST`, `LOCALE_UNSUPPORTED`; 401; 403 `CROSS_ORIGIN`

### POST /auth/change-password
- **Description**: changes the own password and revokes the user's other sessions.
- **Auth**: login, same origin
- **Request**: `{current_password, new_password}`
- **Response**: 200, `data = []`
- **Errors**: 400 `BAD_REQUEST`, `INVALID_CURRENT_PASSWORD`, `PASSWORD_TOO_SHORT` (< `auth.min-password-length`), `PASSWORD_TOO_LONG` (> 72 bytes); 401; 403 `CROSS_ORIGIN`

## Configs

### GET /meta/tenant-config
- **Description**: enumerations for the UI.
- **Auth**: perm `tenants.read`
- **Response**: 200, `data[0] = {claim_types[], notification_events[], channels[], custom_field_types[], modules[]}`
- **Errors**: 401, 403

### GET /configs/defaults
- **Description**: a minimal valid TenantConfig to prefill the new-tenant form.
- **Auth**: perm `tenants.manage`
- **Response**: 200, `data[0] = TenantConfig`
- **Errors**: 401, 403

### POST /configs/validate
- **Description**: validates without saving.
- **Auth**: perm `tenants.manage`, same origin
- **Request**: `{config: TenantConfig}`
- **Response**: 200 even when invalid, `data[0] = {valid, issues: Issue[]}`
- **Errors**: 400 `BAD_REQUEST`, `CONFIG_REQUIRED`; 401; 403

## Tenants

### GET /tenants
- **Description**: active tenants ordered by name.
- **Auth**: perm `tenants.read`
- **Request**: query `include_archived=1|true` (optional)
- **Response**: 200, `data[0] = {tenants: Tenant[]}`
- **Errors**: 401, 403

### POST /tenants
- **Description**: onboards a tenant; stores the config as version 1 (`source=create`).
- **Auth**: perm `tenants.manage`, same origin
- **Request**: `{slug, config: TenantConfig, note}`; slug `^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])$`
- **Response**: 200, `data[0] = TenantDetail`
- **Errors**: 400 `BAD_REQUEST`, `CONFIG_REQUIRED`, `TENANT_SLUG_INVALID`, `CONFIG_INVALID` (`data[0] = {valid: false, issues}`); 409 `TENANT_SLUG_TAKEN`; 401; 403

### GET /tenants/diff
- **Description**: compares two configs. `a`/`b` are `<id|slug>` (current version) or `<id|slug>@<version>`. Scalar lists compare as sets; tiers match by `role`, custom fields by `key`; absent/null/""/empty count as equal.
- **Auth**: perm `tenants.read`
- **Request**: query `a`, `b` (required)
- **Response**: 200, `data[0] = {a: {tenant_id, slug, name, version}, b: {…}, changes: [{path, module, kind: added|removed|changed, left, right}]}`
- **Errors**: 400 `BAD_REQUEST` (missing ref or bad version); 404 `TENANT_NOT_FOUND`, `VERSION_NOT_FOUND`; 401; 403

### GET /tenants/{id}
- **Description**: tenant with its current config (archived tenants included).
- **Auth**: perm `tenants.read`
- **Response**: 200, `data[0] = TenantDetail`
- **Errors**: 404 `TENANT_NOT_FOUND`; 401; 403

### PUT /tenants/{id}
- **Description**: saves a new version (`source=update`). `base_version` must equal the current version.
- **Auth**: perm `tenants.manage`, same origin
- **Request**: `{config: TenantConfig, note, base_version}`
- **Response**: 200, `data[0] = TenantDetail` (new `current_version`)
- **Errors**: 400 `BAD_REQUEST`, `CONFIG_REQUIRED`, `VERSION_REQUIRED`, `CONFIG_INVALID`; 404 `TENANT_NOT_FOUND`; 409 `VERSION_CONFLICT` (stale base or concurrent save), `TENANT_ARCHIVED`; 401; 403

### DELETE /tenants/{id}
- **Description**: archives (soft-deletes) the tenant; history stays readable, claims are refused.
- **Auth**: perm `tenants.manage`, same origin
- **Response**: 200, `data = []`
- **Errors**: 404 `TENANT_NOT_FOUND`; 409 `TENANT_ARCHIVED`; 401; 403

### GET /tenants/{id}/versions
- **Description**: history, newest first, without configs.
- **Auth**: perm `tenants.read`
- **Response**: 200, `data[0] = {versions: [{tenant_id, version, note, source: create|update|rollback, rolled_back_from, created_by, created_by_email, created_at, is_current}]}`
- **Errors**: 404 `TENANT_NOT_FOUND`; 401; 403

### GET /tenants/{id}/versions/{version}
- **Description**: one version including its `config`.
- **Auth**: perm `tenants.read`
- **Response**: 200, `data[0] = TenantVersion` with `config`
- **Errors**: 400 `BAD_REQUEST` (non-positive version); 404 `TENANT_NOT_FOUND`, `VERSION_NOT_FOUND`; 401; 403

### POST /tenants/{id}/rollback
- **Description**: creates a new version (`source=rollback`, `rolled_back_from`) whose config equals the chosen version; the old config is re-validated first.
- **Auth**: perm `tenants.manage`, same origin
- **Request**: `{version, note}` (note defaults to "Rollback to vN")
- **Response**: 200, `data[0] = TenantDetail`
- **Errors**: 400 `BAD_REQUEST`, `CONFIG_INVALID`; 404 `TENANT_NOT_FOUND`, `VERSION_NOT_FOUND`; 409 `VERSION_CONFLICT`, `TENANT_ARCHIVED`; 401; 403

## Claims

### POST /claims/preview
- **Description**: runs the same engine as `/claims/process` without storing. With `config`, previews an unsaved config (`tenant.config_version = 0`); `tenant_id` then only names it.
- **Auth**: perm `tenants.read`, same origin
- **Request**: `{tenant_id?, config?: TenantConfig, claim: {claim_type, amount, submitted_at?, custom_fields?, documents?}}` (`submitted_at` RFC 3339, default now)
- **Response**: 200, `data[0] = ProcessResult`
- **Errors**: 400 `BAD_REQUEST`, `TENANT_REQUIRED`, `CONFIG_INVALID`, `CLAIM_TYPE_REQUIRED`, `CLAIM_TYPE_UNKNOWN`, `CLAIM_AMOUNT_INVALID`; 404 `TENANT_NOT_FOUND`; 409 `TENANT_ARCHIVED`; 401; 403

### POST /claims/process
- **Description**: `processClaim(tenantId, claimData)`: documents, approval routing, notifications, SLA deadline (business days in the tenant's time zone), custom-field validation. Stores the claim in `claim` with the config version used.
- **Auth**: perm `claims.process`, same origin
- **Request**: `{tenant_id, claim}`
- **Response**: 200, `data[0] = ProcessResult` with `claim_id`
- **Errors**: as `/claims/preview` (no `CONFIG_INVALID`)

### POST /claims/process-batch
- **Description**: the same claim against several tenants (empty `tenant_ids` = all active), each stored. All tenants are resolved before anything is stored.
- **Auth**: perm `claims.process`, same origin
- **Request**: `{tenant_ids: string[], claim}`
- **Response**: 200, `data[0] = {results: ProcessResult[]}`
- **Errors**: as `/claims/process`

### GET /claims
- **Description**: processed claims, newest first, with their stored results.
- **Auth**: perm `claims.read`
- **Request**: query `tenant_id` (id or slug, optional), `limit` (default 50, max 200)
- **Response**: 200, `data[0] = {claims: [{id, tenant_id, tenant_slug, config_version, claim_type, amount, submitted_at, accepted, routing, approver_role, sla_deadline, created_by, created_at, result}]}`
- **Errors**: 404 `TENANT_NOT_FOUND`; 401; 403
