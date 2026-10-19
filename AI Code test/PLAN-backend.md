# Plan — Backend Go (Gin + sqlx + SQLite + Swagger)

> Phạm vi: `workspace/backend/` cho AI Challenge 15. Bổ sung cho `PLAN-challenge15.md` (đọc mục 2–5 của file đó cho config schema, engine và API contract).
> Convention lấy từ các skill `write-code-be`, `write-code-full`, `admin-auth`, `backend-check`, `backend-api-docs`.
> Schema DB **đã có và đã chạy**: `migrations/001–005`, `data/app.db`, `seed/*.json`.

---

## 1. Stack & thư viện

| Mục đích | Thư viện |
|---|---|
| HTTP | `github.com/gin-gonic/gin` |
| DB | `github.com/jmoiron/sqlx` + `modernc.org/sqlite` (local, không CGO) + `github.com/tursodatabase/libsql-client-go/libsql` (production) |
| Swagger | `github.com/swaggo/swag` (CLI `swag`, đã cài ở `~/go/bin`), `github.com/swaggo/gin-swagger`, `github.com/swaggo/files` |
| Mật khẩu | `golang.org/x/crypto/bcrypt` (cost 12) |
| Config | `gopkg.in/yaml.v3` + `github.com/joho/godotenv` (`config/.env`, mở rộng `"${VAR}"` trong yml) |
| ID | `github.com/google/uuid` |
| Test | `testing` + `net/http/httptest` của stdlib (không thêm testify — skill yêu cầu dùng tool có sẵn) |

---

## 2. Cấu trúc thư mục

```
workspace/backend/
  main.go                       # @title… swagger general info; subcommand: serve (mặc định) | migrate [--dry-run] | create-user
  go.mod
  Makefile                      # run, test, swag, check, migrate
  config/
    application.yml.example     # mọi secret là "${VAR}"
    application.yml             # gitignored
    .env                        # gitignored
  docs/                         # do `swag init` sinh ra: docs.go, swagger.json, swagger.yaml (commit vào repo)
  migrations/                   # 001–005 (đã có) — go:embed
  seed/                         # safeguard/healthfirst/govhealth.json (đã có) — go:embed
  i18n/
    i18n.go                     # T(locale, key, params), Supported = [vi, en]
    locales/vi.json, en.json    # error.<CODE>, issue.<CODE>, role.<code>, permission.<code>
  utils/
    config.go                   # AppConfig, DatabaseConfig, AuthConfig + WithDefaults()
    logs.go                     # ShowInfoLogs / ShowErrorLogs (không bao giờ log secret)
  middleware/
    locale.go                   # Locale(): Accept-Language → c.Set("locale")
    require_auth.go             # RequireAuth(Authenticator), RequirePermission(code), CurrentUser(c), RequireSameOrigin()
    request_logger.go           # bỏ query string khỏi log
  server/
    base.go                     # Initialize(): mở DB, migrate, seed, tạo repo/service/handler, trả *gin.Engine
    migrate.go                  # runner nhúng (port từ tool dbinit): schema_migrations, mỗi file 1 transaction
    seed.go                     # seed tenant nếu bảng trống
    create_user.go              # CLI tạo user (mật khẩu đọc từ stdin)
    static.go                   # phục vụ FE build + SPA fallback (Phase B6)
  component/
    api/
      response.go               # Response, Ok(), ErrorWithMessage(), ErrorWithData()
    models/
      auth_model.go             # AppUser, SessionUser, Role, Perm… constants
      tenant_model.go           # Tenant, TenantVersion, DTO request/response
      tenant_config_model.go    # TenantConfig + các section (JSON snake_case)
      claim_model.go            # Claim, ProcessResult, ClaimRecord
    tenantconfig/               # domain THUẦN, không import DB/gin
      module.go, registry.go, issues.go
      mod_branding.go, mod_claimtypes.go, mod_approval.go,
      mod_notifications.go, mod_sla.go, mod_customfields.go
      engine.go                 # ProcessClaim(cfg, claim, now)
      diff.go                   # Diff(a, b)
      businessdays.go
    repositories/
      db.go                     # Open(url): chọn driver theo scheme, set pragma, SetMaxOpenConns
      user_repo.go, session_repo.go, role_repo.go
      tenant_repo.go            # tenant + tenant_config_version (transaction)
      claim_repo.go
    services/
      auth_service.go, auth_errors.go
      tenant_service.go, tenant_errors.go
      claim_service.go
    handlers/rest/
      rest.go                   # APIRoutes(): mount mọi group dưới /api/v1 + /swagger
      health_handler.go
      auth_handler.go
      tenant_handler.go
      config_handler.go         # /meta, /configs/defaults, /configs/validate
      claim_handler.go          # /claims/process, /claims/preview, GET /claims
      diff_handler.go
      localize.go               # dịch error_text, issue.text, role/permission name
  API_DOCUMENT.md
  scripts/demo.sh
  Dockerfile
```

Hướng import (một chiều): `handlers/rest → services → repositories → models`, `services → tenantconfig → models`. `middleware` **không** import `services`: `RequireAuth` nhận interface `Authenticator` một method.

---

## 3. Những thứ dùng chung

### 3.1 Response envelope (luôn HTTP 200)
```go
// Response is the envelope every /api/v1 endpoint returns with HTTP 200.
type Response struct {
    Code      int    `json:"code" example:"200"`              // HTTP-like status
    Message   string `json:"message" example:"OK"`            // "OK" hoặc mã lỗi ổn định, vd TENANT_NOT_FOUND
    ErrorText string `json:"error_text,omitempty" example:""` // lỗi đã dịch theo locale
    Data      any    `json:"data"`                            // luôn là mảng
}

func Ok(c *gin.Context, data ...any)                                   // code 200, data = data (nil → [])
func ErrorWithMessage(c *gin.Context, status int, code string)          // data = []
func ErrorWithData(c *gin.Context, status int, code string, data any)   // vd CONFIG_INVALID + {issues}
```
`error_text` = `i18n.T(locale, "error."+code)`. Lỗi bind JSON → `BAD_REQUEST`, **không** trả `err.Error()`.

### 3.2 Config (`config/application.yml.example`)
```yaml
server:
  port: 8080
  static-dir: ../frontend/dist        # rỗng = không phục vụ FE
  swagger: true                       # bật /swagger/*
database:
  url: "${DATABASE_URL}"              # file:./data/app.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)
                                      # hoặc libsql://<db>.turso.io?authToken=...
  auto-migrate: true
  seed: true
auth:
  session-ttl-hours: 12
  remember-me-days: 7
  min-password-length: 10
  max-failed-logins: 5
  lockout-minutes: 15
  cookie-name: session
  cookie-secure: false
```

### 3.3 Error codes (mỗi mã có `error.<CODE>` trong `vi.json` và `en.json`)

| Mã | code | Khi nào |
|---|---|---|
| `BAD_REQUEST` | 400 | JSON sai hoặc thiếu trường bắt buộc |
| `CONFIG_INVALID` | 400 | Validate config thất bại, kèm `data[0].issues` |
| `TENANT_SLUG_INVALID` / `TENANT_SLUG_TAKEN` | 400 / 409 | Slug sai định dạng / đã tồn tại |
| `TENANT_NOT_FOUND` / `TENANT_ARCHIVED` | 404 / 409 | |
| `VERSION_NOT_FOUND` | 404 | Rollback hoặc diff tới version không có |
| `VERSION_CONFLICT` | 409 | `base_version` lệch, hoặc 2 lần lưu cùng lúc (SQLite trigger 1811 / UNIQUE) |
| `UNAUTHENTICATED` | 401 | Chưa đăng nhập hoặc session hết hạn |
| `FORBIDDEN` | 403 | Thiếu quyền |
| `CROSS_ORIGIN` | 403 | Request thay đổi dữ liệu nhưng Origin khác host |
| `INVALID_CREDENTIALS` | 401 | Sai email/mật khẩu (không lộ tài khoản có tồn tại hay không) |
| `ACCOUNT_LOCKED` | 423 | Sai mật khẩu quá 5 lần |
| `PASSWORD_TOO_SHORT` / `PASSWORD_MISMATCH` | 400 | Đổi mật khẩu |
| `INTERNAL_ERROR` | 500 | Lỗi không lường trước (log chi tiết, không trả ra client) |

Issue validation config (`issue.<CODE>`): xem mục 2 của `PLAN-challenge15.md`.

---

## 4. Swagger

### 4.1 Thiết lập
- General info trong `main.go`:
  ```go
  // @title           Multi-Tenant Claim Config API
  // @version         1.0
  // @description     Tenant configuration + runtime claim processing (AI Challenge 15).
  // @description     Every endpoint returns HTTP 200 with {code, message, error_text, data[]}; the real status is in `code`.
  // @description     Auth: call POST /auth/login first — the session cookie is then sent automatically by Swagger UI (same origin).
  // @BasePath        /api/v1
  // @tag.name        auth
  // @tag.name        tenants
  // @tag.name        configs
  // @tag.name        claims
  // @tag.name        health
  ```
- Mount trong `rest.go`: `r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))` khi `server.swagger: true`. Route này public (đúng như skill). Bật cả trên production để reviewer thử API.
- Sinh docs:
  ```bash
  swag fmt                                   # format annotation
  swag init -g main.go -o docs --parseInternal --requiredByDefault=false
  ```
  `docs/` **được commit**. `make swag` chạy cả hai lệnh. Thêm vào `make check` bước sinh lại docs rồi `git diff --exit-code docs/` để không quên cập nhật.
- **Xác thực bằng cookie:** Swagger 2.0 không mô tả được auth bằng cookie (`apiKey` chỉ hỗ trợ `header`/`query`). Cách xử lý: Swagger UI cùng origin với API, nên chỉ cần "Try it out" `POST /auth/login` là cookie được set, các request sau tự gửi kèm. Ghi rõ trong `@description`. Mỗi endpoint cần login ghi `Requires permission: tenants.manage` trong `@Description`.

### 4.2 Mô tả envelope bằng generic của swag
```go
// @Success 200 {object} api.Response{data=[]models.TenantDetail}
```
Vì luôn HTTP 200 nên không dùng `@Failure` theo mã HTTP. Thay vào đó liệt kê mã lỗi trong `@Description`. Mỗi DTO có `example:"..."` để Swagger UI điền sẵn request mẫu.

### 4.3 Mẫu annotation
```go
// Create godoc
// @Summary      Create a tenant
// @Description  Validates the config and stores it as version 1.
// @Description  Requires permission `tenants.manage` and a same-origin request.
// @Description  Errors (in `message`): BAD_REQUEST, CONFIG_INVALID (data[0].issues), TENANT_SLUG_INVALID, TENANT_SLUG_TAKEN, UNAUTHENTICATED, FORBIDDEN.
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        body  body      models.CreateTenantRequest  true  "Slug, full config and a note"
// @Success      200   {object}  api.Response{data=[]models.TenantDetail}
// @Router       /tenants [post]
func (h *TenantHandler) Create(c *gin.Context) { … }
```
Request mẫu cho claim (để ai cũng thử được ngay trên Swagger UI):
```go
type ProcessClaimRequest struct {
    TenantID string `json:"tenant_id" example:"60244e2b-abde-4cf1-ab77-c7bc8d4cf6d9" binding:"required"`
    Claim    Claim  `json:"claim" binding:"required"`
}
type Claim struct {
    ClaimType    string         `json:"claim_type" example:"OUTPATIENT" enums:"OUTPATIENT,INPATIENT,DENTAL,MATERNITY,OPTICAL"`
    Amount       float64        `json:"amount" example:"15000" minimum:"0"`
    SubmittedAt  *time.Time     `json:"submitted_at" example:"2026-10-16T09:00:00+07:00"`
    CustomFields map[string]any `json:"custom_fields" swaggertype:"object,string" example:"employee_id:EMP-12345"`
    Documents    []string       `json:"documents" example:"invoice,prescription"`
}
```

---

## 5. Danh sách endpoint (`/api/v1`)

| Tag | Method | Path | Quyền | Ghi chú |
|---|---|---|---|---|
| health | GET | `/health` | public | `{status, db, version}` |
| auth | POST | `/auth/login` | public + sameOrigin | `{email, password, remember_me}` → set cookie, trả SessionUser |
| auth | POST | `/auth/logout` | public + sameOrigin | Thu hồi session hiện tại |
| auth | POST | `/auth/logout-all` | login + sameOrigin | Thu hồi mọi session, trả `{revoked_sessions}` |
| auth | GET | `/auth/me` | login | user, role `{code,name}`, `permissions`, `permission_names`, `locale` |
| auth | PUT | `/auth/me/locale` | login + sameOrigin | `{locale: vi|en}` |
| auth | POST | `/auth/change-password` | login + sameOrigin | `{current_password, new_password}` |
| configs | GET | `/meta/tenant-config` | `tenants.read` | Enum claim types / events / channels / field types |
| configs | GET | `/configs/defaults` | `tenants.manage` | Config mẫu hợp lệ |
| configs | POST | `/configs/validate` | `tenants.manage` + sameOrigin | `{config}` → `{valid, issues}` |
| tenants | GET | `/tenants` | `tenants.read` | `?include_archived=1` |
| tenants | POST | `/tenants` | `tenants.manage` + sameOrigin | v1 |
| tenants | GET | `/tenants/:id` | `tenants.read` | Config hiện tại |
| tenants | PUT | `/tenants/:id` | `tenants.manage` + sameOrigin | `{config, note, base_version}` |
| tenants | DELETE | `/tenants/:id` | `tenants.manage` + sameOrigin | Archive |
| tenants | GET | `/tenants/:id/versions` | `tenants.read` | Không kèm config (nhẹ) |
| tenants | GET | `/tenants/:id/versions/:version` | `tenants.read` | Kèm config |
| tenants | POST | `/tenants/:id/rollback` | `tenants.manage` + sameOrigin | `{version, note}` → version mới |
| tenants | GET | `/tenants/diff` | `tenants.read` | `?a=<id>[@v]&b=<id>[@v]` |
| claims | POST | `/claims/preview` | `tenants.read` + sameOrigin | `{tenant_id}` hoặc `{config}` + `claim`, không lưu |
| claims | POST | `/claims/process` | `claims.process` + sameOrigin | Lưu vào `claim`, trả `ProcessResult` + `claim_id` |
| claims | POST | `/claims/process-batch` | `claims.process` + sameOrigin | 1 claim × nhiều tenant (demo "3 claim giống nhau") |
| claims | GET | `/claims` | `claims.read` | `?tenant_id=&limit=` |

Ngoài `/api/v1`: `/swagger/*any` (public), `/*` phục vụ FE build (SPA fallback, Phase B6).

---

## 6. Các tầng chính

### Repository (sqlx, SQL thuần, `?` placeholder)
- Không có dòng ⇒ trả `(nil, nil)`.
- `TenantRepo.Create(tx, tenant, version)`, `AppendVersion(tx, tenantID, baseVersion, cfgJSON, source, rolledBackFrom, userID)`:
  ```sql
  UPDATE tenant SET current_version = ?, name = ?, updated_at = ? WHERE id = ? AND current_version = ?
  ```
  Thứ tự: insert version `base+1` → update có điều kiện `current_version = base`. Nếu không dòng nào bị ảnh hưởng, hoặc gặp lỗi SQLite 1811/2067, thì trả `ErrVersionConflict`.
- Helper `WithTx(ctx, fn)` dùng chung.
- Mở DB: `SetMaxOpenConns(1)` cho file local (SQLite một writer); Turso dùng pool mặc định.

### Service (interface + Impl, phụ thuộc interface store private)
```go
type tenantStore interface {
    List(ctx context.Context, includeArchived bool) ([]models.Tenant, error)
    Get(ctx context.Context, id string) (*models.TenantDetail, error)
    Create(ctx context.Context, t models.Tenant, cfg []byte, note string, by *int64) error
    AppendVersion(ctx context.Context, in AppendVersionInput) (*models.TenantDetail, error)
    …
}
type TenantService interface { … }
type TenantServiceImpl struct { store tenantStore; now func() time.Time }
```
Lỗi nghiệp vụ là sentinel có `Code` và `Status` (theo mẫu `auth_errors.go`). Handler map lỗi bằng `errors.As` → `ErrorWithMessage`.

### Domain `tenantconfig`
Đúng theo mục 1–3 của `PLAN-challenge15.md`: interface `Module`, registry, `ValidateConfig`, `ProcessClaim`, `Diff`, business days. Không import gin/sqlx.

### Auth (nếu không copy code nền)
- Đăng nhập bằng mật khẩu: bcrypt, so sánh bcrypt giả khi email không tồn tại để thời gian phản hồi như nhau, khóa tài khoản sau 5 lần sai trong 15 phút.
- Session: token ngẫu nhiên 32 byte, DB chỉ lưu `sha256` hex. Cookie `HttpOnly`, `SameSite=Lax`, `Secure` theo config, có `Max-Age` chỉ khi remember-me.
- `RequireAuth`: đọc cookie → `GetValid` (join `status='active'`, chưa revoke, chưa hết hạn) → set `SessionUser` + locale của user.
- `RequireSameOrigin`: với POST/PUT/PATCH/DELETE, `Origin` (hoặc `Referer`) phải trùng host.
- Google OIDC: **ngoài phạm vi** (bảng `oauth_login_attempt` đã sẵn để thêm sau).

---

## 7. Các phase

| Phase | Nội dung | Thời gian | Xong khi |
|---|---|---|---|
| **B0** Scaffold | go.mod + deps; `utils/config.go`, logs; `repositories/db.go`; `server/migrate.go` + `seed.go` (port từ dbinit); `main.go` subcommand `serve`/`migrate`; `api/response.go`; `/health`; Swagger general info + `/swagger`; Makefile; `config/application.yml.example`; `.gitignore` (`config/application.yml`, `config/.env`, `data/`, `go.sum`) | 45' | `go run .` mở được `/swagger/index.html`; `/api/v1/health` trả `code 200`; `check.sh` xanh |
| **B1** Auth | i18n; models auth; user/session/role repo; `AuthService`; middleware Locale/RequireAuth/RequirePermission/RequireSameOrigin/request logger; auth handler + swagger; CLI `create-user` | 1h30 | Login `admin@papaya.com` trên Swagger UI rồi gọi `/auth/me` thành công; test: sai mật khẩu, khóa tài khoản, không lộ tài khoản, thiếu quyền, cross-origin |
| **B2** Domain | `tenantconfig/*` + table test 3 tenant × cùng claim, test biên, diff, business days; seed pass validate | 1h30 | `go test ./component/tenantconfig/` xanh, khớp bảng ở mục 9 của PLAN-challenge15 |
| **B3** Tenants API | tenant repo/service/handler: CRUD, versions, rollback, diff, validate, defaults, meta; mã lỗi + i18n | 1h15 | Handler test qua `APIRoutes`: tạo, sửa, conflict, rollback, archive, invalid config ra issues |
| **B4** Claims API | claim repo/service/handler: preview (cả config chưa lưu), process (lưu DB), process-batch, list | 45' | Demo trên Swagger: 1 claim → 3 tenant → 3 kết quả khác nhau |
| **B5** Docs | `swag fmt && swag init`; rà example/enum; `/backend-api-docs` → `API_DOCUMENT.md`; `scripts/demo.sh` (curl có cookie jar) | 30' | Swagger UI đủ 23 endpoint, mỗi endpoint có request mẫu chạy được |
| **B6** Deploy-ready | `server/static.go` phục vụ `../frontend/dist`; Dockerfile multi-stage (`CGO_ENABLED=0`); env Turso | 30' | `docker build` + chạy được với `DATABASE_URL` Turso |

**Tổng ~6h45** (chỉ backend).

---

## 8. Test

- `tenantconfig`: table-driven, không cần DB.
- Service: fake store (struct trong `_test.go`), có `now` cố định.
- Handler: `httptest` + `APIRoutes` thật, kiểm `code`/`message` trong body; session tạo sẵn qua repo.
- Repository: SQLite file tạm `t.TempDir()` + migration runner thật (không cần env như Postgres); có test cho trigger (append-only, conflict).
- i18n: test `vi.json` và `en.json` có cùng bộ key; mọi mã lỗi trong `*_errors.go` đều có key `error.<CODE>`.
- Swagger: test `GET /swagger/doc.json` trả 200 và có đủ path chính (phát hiện quên `swag init`).

---

## 9. Lệnh

```bash
cd workspace/backend
make run          # go run . serve
make migrate      # go run . migrate --dry-run, rồi go run . migrate
make swag         # swag fmt && swag init -g main.go -o docs --parseInternal
make test         # go test ./...
go test ./component/tenantconfig/ -run TestProcessClaim_SameClaimThreeTenants -v
bash ../../.claude/skills/backend-check/check.sh   # chạy từ root workspace
echo 'a-long-password' | go run . create-user --email an@papaya.com --name "An" --role operator --password-stdin
```
Swagger UI: http://localhost:8080/swagger/index.html

---

## 10. Rủi ro

| Rủi ro | Cách xử lý |
|---|---|
| swag không hiểu `any` / `map[string]any` | Dùng `swaggertype:"object"` trên field |
| Swagger 2.0 không mô tả được cookie auth | Login qua Swagger UI cùng origin; ghi chú trong `@description` |
| Quên chạy `swag init` sau khi sửa handler | `make check` sinh lại docs rồi `git diff --exit-code docs/`; có test `doc.json` |
| SQLite chỉ một writer → `database is locked` | `busy_timeout(5000)`, WAL, `SetMaxOpenConns(1)` cho file |
| Libsql (Turso) khác modernc ở pragma | `foreign_keys` set bằng `PRAGMA` sau khi mở thay vì qua DSN khi scheme là `libsql://` |
| Code nền copy vào khác cấu trúc này | Cấu trúc ở đây bám theo skill, nên auth của code nền thay thẳng vào B1 |
