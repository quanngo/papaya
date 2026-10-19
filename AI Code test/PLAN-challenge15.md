# Plan — AI Challenge 15: Multi-Tenant Configuration Platform

> Đề: https://github.com/papaya-insurtech/pumpkin/blob/main/Hiring/AI_Engineering_Challenges/AI_Challenge_15.md
> Plan này bám theo convention của các skill trong `.claude/skills/` (`write-code-full/be/fe`, `backend-check`, `db-migrate`, `admin-auth`).
> Ước lượng gửi reviewer: **11–13 giờ** (đề gợi ý 5–8h; phần vượt là do dựng nền có login, i18n vi/en và chuyển nền code sang SQLite — nên giải thích trong README).

---

## 0. Quyết định chính

| Hạng mục | Lựa chọn | Nguồn |
|---|---|---|
| Vị trí code | BE **trực tiếp** trong `workspace/backend/`, FE trực tiếp trong `workspace/frontend/` (không có thư mục `<key>`) | write-code-full bước 2 |
| Nền code | Copy codebase có sẵn (Go Gin + sqlx, auth Google OIDC + email/password, RBAC, i18n, React 18 JS) rồi **lược bỏ phần ecom** | Bạn chọn |
| Backend | Go + **Gin + sqlx**, layering `models → repositories → services (interface + Impl) → handlers/rest`, wiring trong `server/base.go` | write-code-be/full bước 5 |
| Response | Luôn **HTTP 200** `{code, message, error_text?, data: [...]}`; `message` là mã lỗi ổn định; `data` luôn là mảng | write-code-full |
| JSON | **snake_case** theo `json` tag của Go | write-code-fe bước 7 |
| DB | **SQLite** (giữ theo lựa chọn của bạn). Local: `modernc.org/sqlite` (pure Go, không CGO). Production: **Turso** (`libsql-client-go`). sqlx mở theo scheme của URL | Bạn chọn |
| Migrations | `workspace/backend/migrations/NNN_*.sql`, chạy mỗi file đúng 1 lần, ghi vào bảng `schema_migrations` — cùng ngữ nghĩa với skill db-migrate, nhưng runner viết cho SQLite | db-migrate (điều chỉnh) |
| i18n | Đủ **vi + en** ở cả BE (`i18n/locales/{vi,en}.json`, `error_text`) và FE (`src/locales/{vi,en}.json`, `t(...)`) | EC-7 trong skill |
| Auth | Dùng lại auth của nền code; thêm permission `tenants.read`, `tenants.manage`, `claims.process` | admin-auth |
| Frontend | React 18 + Vite, **JavaScript/JSX thuần**, CSS module + token `var(--…)`, `useState/useEffect`, gọi API qua `apiRequest`. **Không** thêm state library, component library, CSS framework (không Tailwind, không TanStack Query, không React Hook Form) | write-code-fe bước 6 |
| Test | BE: `go test` (service test với fake store, handler test bằng gin + httptest qua `APIRoutes`, repo test trên SQLite). FE: Vitest + Testing Library + MSW qua `renderApp('/route')`. E2E: Playwright | write-code-full bước 8, 10 |
| Kiểm tra | `bash .claude/skills/backend-check/check.sh`; FE `npm test && npm run build` | backend-check |
| Deploy | **1 service**: Go phục vụ luôn bản build FE (cùng origin → cookie session và `sameOrigin` hoạt động), Docker trên Render free; DB trên Turso | Ràng buộc same-origin của auth |

**Nguyên tắc vàng (ghi vào root `CLAUDE.md` và `workspace/backend/CLAUDE.md`):**
1. Không bao giờ có `if tenant == "..."` — mọi khác biệt nằm trong config.
2. Engine là **pure function** `ProcessClaim(cfg, claim, now)`, không I/O. Preview và runtime gọi **cùng** hàm.
3. Backend là nguồn sự thật cho validation; FE chỉ hiển thị issue BE trả về.
4. Config bất biến theo version: mọi lần lưu/rollback = **insert version mới**.
5. Mọi chữ người dùng đọc đều có ở cả `vi.json` và `en.json`.

---

## 1. Kiến trúc

```
Trình duyệt ── same origin ──► Go (Gin)
                                ├── /api/v1/...   (REST, envelope HTTP 200)
                                └── /*            (static: workspace/frontend/dist, SPA fallback)

workspace/backend/
  component/
    models/            tenant_model.go, tenant_config_model.go, claim_model.go
    tenantconfig/      ← domain thuần, không I/O
      module.go, registry.go, engine.go, diff.go, businessdays.go, issues.go
      mod_branding.go, mod_claimtypes.go, mod_approval.go,
      mod_notifications.go, mod_sla.go, mod_customfields.go
    repositories/      tenant_repo.go, tenant_version_repo.go
    services/          tenant_service.go, claim_service.go, tenant_errors.go
    handlers/rest/     tenant_handler.go, claim_handler.go, config_handler.go, diff_handler.go (+ mount trong rest.go)
  migrations/          001.. (auth đã port) + 0NN_create_tenant_config.sql + 0NN_tenant_permissions.sql
  seed/                safeguard.json, healthfirst.json, govhealth.json (go:embed)
  i18n/locales/        vi.json, en.json
  server/base.go       wiring
  API_DOCUMENT.md

workspace/frontend/src/
  api/                 tenants.js, claims.js, configs.js, diff.js   (gọi apiRequest)
  pages/               TenantsPage.jsx, TenantEditPage.jsx, PreviewPage.jsx,
                       DiffPage.jsx, TenantHistoryPage.jsx (+ .module.css)
  pages/tenant-sections/  BrandingSection.jsx, ClaimTypesSection.jsx, ApprovalSection.jsx,
                          NotificationsSection.jsx, SlaSection.jsx, CustomFieldsSection.jsx, index.js
  tenant/              issues.js (map issue.path → field), tenantErrorMessage.js
  locales/             vi.json, en.json
  test/server.js       MSW handlers với envelope thật
  e2e/                 tenant-onboarding.spec.js
```

### Module contract (chìa khoá cho tiêu chí "modular")
```go
type Module interface {
    Key() string                                                      // "approval"
    Validate(cfg *models.TenantConfig) []Issue                        // gồm rule liên module
    Process(cfg *models.TenantConfig, claim models.Claim, ctx *Ctx, res *models.ProcessResult)
    DiffKeys() map[string]string                                      // "approval.tiers" → "role"
}

var Modules = []Module{
    BrandingModule{}, ClaimTypesModule{}, ApprovalModule{},
    NotificationsModule{}, SLAModule{}, CustomFieldsModule{},
}
```
`Issue` mang **mã** chứ không mang câu chữ, để dịch được:
```go
type Issue struct {
    Path   string         `json:"path"`             // "approval.tiers[1].min_amount"
    Code   string         `json:"code"`             // "TIER_GAP"
    Params map[string]any `json:"params,omitempty"` // {"expected": 100000}
    Text   string         `json:"text"`             // BE dịch theo locale request (fallback cho FE)
}
```
Thêm "currency" = field `Currency` trong `TenantConfig` + `mod_currency.go` + 1 dòng registry + `CurrencySection.jsx` + key i18n. (Phase 9, commit riêng để chứng minh.)

---

## 2. Config schema (JSON snake_case)

```go
type TenantConfig struct {
    Branding      BrandingConfig      `json:"branding"`
    ClaimTypes    ClaimTypesConfig    `json:"claim_types"`
    Approval      ApprovalConfig      `json:"approval"`
    Notifications NotificationsConfig `json:"notifications"`
    SLA           SLAConfig           `json:"sla"`
    CustomFields  []CustomField       `json:"custom_fields"`
}
type BrandingConfig struct {
    CompanyName    string `json:"company_name"`
    LogoURL        string `json:"logo_url"`
    PrimaryColor   string `json:"primary_color"`
    SecondaryColor string `json:"secondary_color"`
}
type ClaimTypesConfig struct {
    Enabled   []string                 `json:"enabled"`   // OUTPATIENT|INPATIENT|DENTAL|MATERNITY|OPTICAL
    Documents map[string]DocumentRules `json:"documents"`
}
type DocumentRules struct {
    Required []string `json:"required"`
    Optional []string `json:"optional"`
}
type ApprovalConfig struct {
    AutoApprovalThreshold float64 `json:"auto_approval_threshold"`
    Tiers                 []Tier  `json:"tiers"`
}
type Tier struct {
    MinAmount float64  `json:"min_amount"`
    MaxAmount *float64 `json:"max_amount"` // nil = không giới hạn; khoảng [min, max)
    Role      string   `json:"role"`
}
type NotificationsConfig struct {
    Events     map[string]EventRule `json:"events"`        // claim_submitted|approved|rejected|payment_sent
    WebhookURL string               `json:"webhook_url,omitempty"`
}
type EventRule struct {
    Enabled  bool     `json:"enabled"`
    Channels []string `json:"channels"`                    // email|sms|webhook
    Template string   `json:"template,omitempty"`
}
type SLAConfig struct {
    Timezone    string           `json:"timezone"`
    Targets     map[string]int   `json:"targets"`          // ngày làm việc > 0
    DefaultDays int              `json:"default_days,omitempty"`
    Escalation  []EscalationRule `json:"escalation"`
}
type EscalationRule struct {
    AfterBreachDays int      `json:"after_breach_days"`
    Notify          []string `json:"notify"`
}
type CustomField struct {
    Key      string   `json:"key"`
    Label    string   `json:"label"`
    Type     string   `json:"type"`                        // text|number|select
    Required bool     `json:"required"`
    Pattern  string   `json:"pattern,omitempty"`
    Options  []string `json:"options,omitempty"`
}
```
Decode bằng `json.Decoder.DisallowUnknownFields()`. **Không** echo `err.Error()` của lỗi bind ra client (rule của skill) — trả `BAD_REQUEST`.

### Validation (mã issue → cần key `issue.<CODE>` trong i18n BE + `errors.config.<CODE>` trong FE)
- [ ] `THRESHOLD_NEGATIVE` — `auto_approval_threshold >= 0`
- [ ] `NO_CLAIM_TYPE` — bật ít nhất 1 loại; `CLAIM_TYPE_UNKNOWN`, `CLAIM_TYPE_DUPLICATE`
- [ ] `SLA_NOT_POSITIVE`; `SLA_MISSING` — loại đang bật phải có SLA (hoặc `default_days`)
- [ ] `DOCUMENTS_MISSING`; `DOCUMENT_BOTH_REQUIRED_OPTIONAL`
- [ ] `NO_TIER`; `TIER_START_MISMATCH` (tier đầu ≠ threshold); `TIER_GAP`/`TIER_OVERLAP`; `TIER_UNBOUNDED_NOT_LAST`; `TIER_ROLE_EMPTY`
- [ ] `WEBHOOK_URL_REQUIRED` / `WEBHOOK_URL_INVALID`; `CHANNEL_UNKNOWN`; `EVENT_NO_CHANNEL`
- [ ] `COLOR_INVALID`, `URL_INVALID`, `COMPANY_NAME_REQUIRED`
- [ ] `TIMEZONE_INVALID`
- [ ] `FIELD_KEY_DUPLICATE`, `FIELD_KEY_INVALID`, `FIELD_OPTIONS_REQUIRED`, `FIELD_PATTERN_INVALID`

---

## 3. Engine — `ProcessClaim(cfg, claim, now)`

```go
type Claim struct {
    ClaimType    string         `json:"claim_type"`
    Amount       float64        `json:"amount"`
    SubmittedAt  *time.Time     `json:"submitted_at"`   // nil = now
    CustomFields map[string]any `json:"custom_fields"`
    Documents    []string       `json:"documents"`
}
type ProcessResult struct {
    Accepted      bool                 `json:"accepted"`
    Errors        []Issue              `json:"errors"`         // CLAIM_TYPE_NOT_ENABLED, FIELD_REQUIRED, FIELD_PATTERN_MISMATCH...
    Tenant        TenantRef            `json:"tenant"`         // id, name, config_version
    Documents     DocumentsResult      `json:"documents"`      // required, optional, missing
    Approval      ApprovalResult       `json:"approval"`       // auto_approved, role, tier, reason_code
    Notifications []NotificationResult `json:"notifications"`  // event, channels, template
    SLA           SLAResult            `json:"sla"`            // business_days, submitted_at, deadline, escalation
    CustomFields  []CustomFieldResult  `json:"custom_fields"`
    Trace         []TraceStep          `json:"trace"`          // {code, params} — FE dịch, dùng cho Preview
}
```
Quy tắc:
- `amount < threshold` ⇒ auto-approve; ngược lại chọn tier có `min <= amount < max`.
- Notifications: `claim_submitted` (nếu bật); auto-approve thì thêm `approved`.
- SLA: cộng N ngày làm việc (bỏ T7/CN) theo `sla.timezone`; `import _ "time/tzdata"` trong `main.go`.
- Claim type không bật ⇒ `accepted=false` + error trong `data[0]`, envelope vẫn `code=200` (đây là kết quả nghiệp vụ, không phải lỗi request).

---

## 4. Data model & migrations (SQLite)

`0NN_create_tenant_config.sql`:
```sql
-- Multi-tenant configuration: tenants and their immutable config versions.
CREATE TABLE IF NOT EXISTS tenant (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  current_version INTEGER NOT NULL,
  archived_at TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tenant_config_version (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id TEXT NOT NULL REFERENCES tenant(id),
  version INTEGER NOT NULL,
  config TEXT NOT NULL,                 -- JSON
  note TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL,                 -- create|update|rollback
  rolled_back_from INTEGER,
  created_by TEXT,                      -- app_user.id
  created_at TEXT NOT NULL,
  UNIQUE (tenant_id, version)
);
```
`0NN_tenant_permissions.sql`: thêm `tenants.read`, `tenants.manage`, `claims.process` vào `permission` và gán cho role (admin: cả 3; operator: read + process; viewer: read).

- Cột Go không phải con trỏ ⇒ `NOT NULL` (rule trong skill).
- Repo: struct trên `*sqlx.DB`, SQL thuần; không tìm thấy ⇒ trả `(nil, nil)`.
- Service phụ thuộc **interface store nhỏ, private** để test bằng fake.
- Save = transaction: validate → insert version `current+1` → update `current_version`; `base_version` lệch ⇒ `VERSION_CONFLICT`.
- Seed 3 tenant khi boot nếu bảng `tenant` trống.
- Runner migration: thư mục `migrations/` nhúng bằng `go:embed`, bảng `schema_migrations(filename, applied_at)`, mỗi file 1 transaction. Chạy khi boot **và** qua lệnh `go run . migrate [--dry-run]` để giữ thói quen dry-run → xác nhận của skill db-migrate.

---

## 5. API contract (`/api/v1`, envelope HTTP 200)

| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/meta/tenant-config` | `tenants.read` | Enum: claim types, events, channels, field types |
| GET | `/configs/defaults` | `tenants.manage` | Config mẫu hợp lệ để tạo tenant mới |
| POST | `/configs/validate` | `tenants.manage` + sameOrigin | `{config}` → `data[0] = {valid, issues}` |
| GET | `/tenants` | `tenants.read` | `data[0] = {tenants}` |
| POST | `/tenants` | `tenants.manage` + sameOrigin | `{slug, config, note}` → v1 |
| GET | `/tenants/:id` | `tenants.read` | Config hiện tại + version |
| PUT | `/tenants/:id` | `tenants.manage` + sameOrigin | `{config, note, base_version}` → version mới |
| DELETE | `/tenants/:id` | `tenants.manage` + sameOrigin | Archive |
| GET | `/tenants/:id/versions` | `tenants.read` | Lịch sử |
| GET | `/tenants/:id/versions/:v` | `tenants.read` | Config của 1 version |
| POST | `/tenants/:id/rollback` | `tenants.manage` + sameOrigin | `{version, note}` → version mới |
| POST | `/claims/process` | `claims.process` + sameOrigin | `{tenant_id, claim}` → `ProcessResult` |
| POST | `/claims/preview` | `tenants.read` + sameOrigin | `{tenant_id}` hoặc `{config}` + `claim` |
| GET | `/tenants/diff?a=<id>[@v]&b=<id>[@v]` | `tenants.read` | `data[0] = {changes: [{path, module, kind, left, right}]}` |

Mã lỗi trong `message` (mỗi mã cần `error.<CODE>` ở `i18n/locales/vi.json` **và** `en.json`):
`BAD_REQUEST`, `CONFIG_INVALID` (kèm `data[0].issues`), `TENANT_NOT_FOUND`, `TENANT_SLUG_TAKEN`, `TENANT_SLUG_INVALID`, `TENANT_ARCHIVED`, `VERSION_NOT_FOUND`, `VERSION_CONFLICT`, cùng các mã auth có sẵn (`UNAUTHENTICATED`, `FORBIDDEN`...).

Diff: marshal 2 config → `map[string]any`, đệ quy; mảng `enabled`/`channels` so như tập hợp; `tiers` theo `role`; `custom_fields` theo `key`.

Mọi handler mới có **swag annotation** như `auth_handler.go`.

---

## 6. Frontend

| Route (trong `ProtectedRoute`) | Quyền | Nội dung |
|---|---|---|
| `/tenants` | `tenants.read` | Bảng tenant + nút New (chỉ khi `can('tenants.manage')`) |
| `/tenants/new`, `/tenants/:id` | `tenants.manage` | Form chia section từ `tenant-sections/index.js`; validate debounce qua `/configs/validate`; issue hiện dưới đúng field (`role="alert"`); nút "Xem thử" chạy preview với config chưa lưu |
| `/tenants/:id/history` | `tenants.read` | Timeline version → xem, diff với hiện tại, Rollback (dùng `ConfirmDialog`) |
| `/preview` | `tenants.read` | Chọn tenant + claim mẫu (custom field render theo config) → card kết quả + trace; "So sánh tất cả tenant" |
| `/diff` | `tenants.read` | Chọn A/B (tenant hoặc tenant@version), 2 cột theo module, highlight, lọc "chỉ khác biệt" |

- Thêm menu "Tenants", "Preview", "Diff" vào `AppLayout.jsx`, ẩn theo `can(...)`.
- State: `useState/useEffect`; form lồng nhau dùng 1 state object + hàm `updateAt(path, value)` tự viết; mảng (tiers, documents, custom fields, escalation) dùng component `ArrayEditor` dùng chung.
- Text: mọi chuỗi qua `t(...)` với key ở **cả** `vi.json` và `en.json`; tiền và ngày qua `src/i18n/format.js`.
- Lỗi: giữ `ApiError`/mã trong state, dịch lúc render bằng `tenantErrorMessage` (cùng kiểu `actionErrorMessage`), fallback `error_text`. Issue dịch qua `errors.config.<CODE>` với `params`, fallback `issue.text`.
- Loading dùng `Spinner` (`role="status"`). Không lưu gì vào localStorage ngoài mã ngôn ngữ.
- Dữ liệu dùng JSDoc `@typedef` (TenantConfig, ProcessResult, Issue) trong `src/api/tenants.js`.
- Giao diện hoạt động ở độ rộng điện thoại; dùng token `var(--…)` để light/dark đều ổn. Màu branding của tenant chỉ dùng cho ô xem trước, không đè token của app.

---

## 7. Các phase triển khai

### Phase 0 — Nền code (1h)
- [ ] Copy codebase vào **`workspace/backend/`** và **`workspace/frontend/`** (trực tiếp, không thư mục con)
- [ ] Lược bỏ phần ecom: marketplace, orders, connector, workers ecom, migrations ecom, menu Orders; giữ auth, users/roles, user_config, i18n, middleware, `api` response, FE auth + layout + test setup
- [ ] Đổi branding app, default admin; `config/application.yml.example` dùng `"${VAR}"`
- [ ] Sửa `projects.env` (1 FE, 1 BE, START: `go run .` và `npm run dev`); viết root `CLAUDE.md` và CLAUDE.md từng phía với nguyên tắc vàng
- [ ] `bash .claude/skills/backend-check/check.sh` và FE `npm test && npm run build` **xanh** trước khi đi tiếp
- [ ] Tạo 2 repo GitHub (hoặc 1 monorepo) và commit nền

### Phase 1 — Chuyển nền sang SQLite (1–1.5h) ⚠️ rủi ro lớn nhất
- [ ] `utils/config.go`: thay block Postgres bằng `database.url` (`file:./data/app.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)` / `libsql://…`)
- [ ] Mở sqlx theo scheme: `sqlite` (modernc) hoặc `libsql`
- [ ] Port các migration auth: bỏ `public.` và schema, `SERIAL/BIGSERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`, `TIMESTAMPTZ`/`NOW()` → `TEXT` + giờ UTC do Go ghi, `JSONB` → `TEXT`, `BOOLEAN` → `INTEGER`, bỏ `CREATE EXTENSION`, `ILIKE` → `LIKE`
- [ ] Rà SQL trong repositories: `$1..$n` → `?` (sqlx `Rebind`), `RETURNING` (SQLite ≥ 3.35 có hỗ trợ), `ON CONFLICT` giữ nguyên, `NOW()`/`INTERVAL` → tham số thời gian từ Go
- [ ] Migration runner nhúng (mục 4) + lệnh `migrate --dry-run`
- [ ] Repo integration test chạy trên SQLite file tạm (không cần `ECOM_TEST_DATABASE_URL`)
- [ ] Đăng nhập bằng email/password chạy được end to end; check xanh
- [ ] Cập nhật skill `db-migrate` (hoặc ghi chú trong CLAUDE.md) rằng DB giờ là SQLite

### Phase 2 — Domain tenantconfig (1.5h) · `/write-code-be`
- [ ] Models, 6 module, registry, issues
- [ ] `businessdays.go` + test (cuối tuần, timezone, nộp thứ Bảy)
- [ ] `engine.go` + table test **3 tenant × cùng claim** (mục 9) + test biên (đúng threshold, đúng ranh giới tier, threshold 0, claim type không bật, thiếu/sai custom field)
- [ ] `diff.go` + test (tập hợp, theo key, lồng nhau, thêm/xóa)
- [ ] 3 file seed pass `ValidateConfig` (có test)

### Phase 3 — Persistence + API (1.5h) · `/write-code-be`
- [ ] Migration bảng tenant + permissions
- [ ] Repositories, `TenantService`/`ClaimService` (interface + Impl, fake store trong test), `tenant_errors.go`
- [ ] Handlers + mount trong `rest.go` với `RequirePermission` và `sameOrigin`; wiring `server/base.go`; seed khi boot
- [ ] i18n BE: `error.<CODE>`, `issue.<CODE>`, `permission.tenants.*`, `permission.claims.process` ở **vi + en**
- [ ] Handler test (gin + httptest qua `APIRoutes`) kiểm `code`/`message`; repo test SQLite
- [ ] Swag annotations; `/backend-api-docs` → `API_DOCUMENT.md`
- [ ] `backend-check` xanh

### Phase 4 — FE danh sách + form tenant (2h) · `/write-code-fe`
- [ ] `src/api/*.js` + JSDoc typedef; menu; route trong `App.jsx`
- [ ] `TenantsPage`, `TenantEditPage` + 6 section + `ArrayEditor`
- [ ] Hiện issue theo path; chặn Save khi invalid; xử lý `VERSION_CONFLICT`
- [ ] Locale vi + en; MSW handlers trong `src/test/server.js`; test qua `renderApp`

### Phase 5 — Preview (45') · `/write-code-fe`
- [ ] Form claim động + result card + trace dịch được + "so sánh tất cả tenant"

### Phase 6 — Diff (45') · `/write-code-fe`

### Phase 7 — History & Rollback (30') · `/write-code-fe`

### Phase 8 — E2E + deploy (1h)
- [ ] `e2e/tenant-onboarding.spec.js`: login → tạo tenant 4 → preview claim → rollback; tự skip khi thiếu env; header ghi cách seed
- [ ] Go phục vụ `workspace/frontend/dist` (SPA fallback cho route không phải `/api`)
- [ ] Dockerfile multi-stage: build FE (node) → build Go (`CGO_ENABLED=0`) → image nhỏ
- [ ] Turso DB; Render web service, env: `DATABASE_URL`, secrets auth, `APP_BASE_URL`; health check
- [ ] Tạo tài khoản demo bằng CLI `create-user` (quyền operator) để ghi vào README cho reviewer; Google OIDC tắt hoặc cấu hình redirect production
- [ ] Smoke test production

### Phase 9 — Chứng minh + tài liệu (45')
- [ ] Tạo tenant thứ 4 **qua UI** trên production, xử lý claim, quay GIF/video
- [ ] (Bonus) module `currency` trong 1 commit riêng → README trỏ tới commit để chứng minh "minimal changes"
- [ ] `scripts/demo.sh` (curl có cookie login) + README: kiến trúc, giả định, cách chạy, link live, tài khoản demo, demo script, thời gian thực tế so với ước lượng, cách đã dùng Claude Code và các skill

---

## 8. Demo script (đưa vào README)

1. Đăng nhập bằng tài khoản demo → menu Tenants thấy 3 tenant seed. Đổi ngôn ngữ vi/en.
2. `/preview` → claim `OUTPATIENT, 15,000, nộp thứ Sáu 2026-10-16` → "So sánh tất cả" → 3 kết quả khác nhau.
3. Thử `DENTAL` → GovHealth báo không hỗ trợ.
4. `/tenants/new` → thử lưu config sai (threshold âm, không bật loại claim, SLA 0, tier hở) → bị chặn, lỗi hiện đúng ô.
5. Tạo **"CareLink Insurance"** (tenant 4): OUTPATIENT + OPTICAL, threshold 10,000, 2 tier, email + webhook, SLA 3 ngày, custom field `policy_number` có pattern → Save.
6. Preview claim với tenant 4 → kết quả theo config mới, **không đụng code**.
7. Sửa threshold SafeGuard → v2 → History → rollback về v1 (tạo v3).
8. `/diff` SafeGuard vs GovHealth, rồi SafeGuard@v1 vs @v2.

---

## 9. Bảng kết quả mong đợi (dùng làm test)

Claim: `OUTPATIENT, amount 15,000, submitted_at 2026-10-16 (Thứ Sáu)`

| | SafeGuard | HealthFirst | GovHealth |
|---|---|---|---|
| Approval | auto-approved | assessor | committee |
| Notifications | submitted + approved: email | submitted: email, sms | submitted: email, webhook |
| SLA | 5 ngày LV → 2026-10-23 | 7 → 2026-10-27 | 15 → 2026-11-06 |
| Custom fields | employee_id (bắt buộc) | không có | department, budget_code (bắt buộc) |

Claim thứ 2 (`INPATIENT, 150,000`): SafeGuard → team_lead, HealthFirst → manager, GovHealth → committee.

---

## 10. Giả định (ghi vào ASSUMPTIONS.md)

- Ngưỡng tier `[min, max)`; tier đầu bắt đầu tại `auto_approval_threshold` (threshold là nguồn duy nhất).
- Mức tier tự đặt: SafeGuard 20k–100k assessor / 100k–500k team_lead / ≥500k director; HealthFirst 5k–50k assessor / ≥50k manager; GovHealth ≥0 committee.
- SafeGuard bật DENTAL nhưng đề không cho SLA → đặt 5 ngày (ghi rõ là giả định).
- Ngày làm việc = T2–T6, không tính ngày lễ; timezone theo tenant.
- `ProcessClaim` trả notification cho sự kiện tại thời điểm submit.
- Danh sách documents tự đặt hợp lý theo từng loại claim.
- Rollback tạo version mới; xóa tenant = archive.
- Tên tenant, nhãn custom field, tên document là dữ liệu do người vận hành nhập → **không** dịch; chỉ UI và mã lỗi được dịch.
- Số tiền dùng `float64` cho đơn giản (production nên dùng số nguyên theo đơn vị nhỏ nhất).
- Không lưu claim; `processClaim` chỉ tính toán. SLA escalation thật (cron qua skill `add-worker`) là hướng mở rộng.

---

## 11. Rủi ro & cách tránh

| Rủi ro | Phòng tránh |
|---|---|
| Nền code viết cho Postgres, chuyển sang SQLite vỡ query auth | Phase 1 riêng, chạy test repo + login trước khi làm tính năng. Nếu tốn quá 2h → cân nhắc quay lại Postgres (Neon) |
| Skill `db-migrate`/`migrate.sh` gọi `psql` | Dùng runner nhúng trong Go; ghi chú trong CLAUDE.md để Claude không gọi `migrate.sh` |
| Cookie session cần same-origin | Go phục vụ luôn FE build (1 service) |
| Render free: sleep ~30s, disk tạm | Turso cho DB; README ghi chú lần đầu tải chậm |
| SQLite ghi đồng thời | `busy_timeout`, `SetMaxOpenConns(1)` cho file local |
| Form lồng nhau không có thư viện form | `updateAt(path)` + `ArrayEditor` dùng chung; validate để BE lo |
| Quên key i18n một bên | Test sẵn có kiểm vi/en khớp; BE thêm test tương tự cho `error.*`/`issue.*` |
| Diff mảng sai | `DiffKeys()` từng module + test |
| Preview lệch runtime | Preview gọi đúng `ProcessClaim` (test chung) |
| Lố thời gian | Cắt theo thứ tự: currency bonus → Playwright e2e → preview config chưa lưu → lọc diff |
