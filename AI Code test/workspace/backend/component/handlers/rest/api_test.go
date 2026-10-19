package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"claimconfig/component/models"
	"claimconfig/server"
	"claimconfig/utils"
)

type envelope struct {
	Code      int               `json:"code"`
	Message   string            `json:"message"`
	ErrorText string            `json:"error_text"`
	Data      []json.RawMessage `json:"data"`
}

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

// newTestServer starts the real app (migrations + seed) on a temp SQLite file.
func newTestServer(t *testing.T) string {
	t.Helper()
	cfg := utils.DefaultConfig()
	cfg.Database.URL = "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	app, err := server.Initialize(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Engine)
	t.Cleanup(func() { srv.Close(); app.Close() })
	return srv.URL
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: base, http: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, headers ...string) envelope {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.base+path, r)
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("%s %s: HTTP %d (the envelope convention is always 200)", method, path, resp.StatusCode)
	}
	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	return env
}

// ok asserts code 200 and decodes data[0] into out.
func (c *client) ok(env envelope, out any) {
	c.t.Helper()
	if env.Code != 200 {
		c.t.Fatalf("code=%d message=%s error_text=%s", env.Code, env.Message, env.ErrorText)
	}
	if out != nil {
		if len(env.Data) == 0 {
			c.t.Fatal("empty data")
		}
		if err := json.Unmarshal(env.Data[0], out); err != nil {
			c.t.Fatal(err)
		}
	}
}

func (c *client) fails(env envelope, code int, message string) {
	c.t.Helper()
	if env.Code != code || env.Message != message {
		c.t.Fatalf("want %d %s, got %d %s", code, message, env.Code, env.Message)
	}
	if env.ErrorText == "" {
		c.t.Errorf("%s has no error_text (missing i18n key?)", message)
	}
}

func (c *client) login(email, password string) models.SessionUser {
	c.t.Helper()
	var u models.SessionUser
	c.ok(c.do("POST", "/api/v1/auth/login", models.LoginRequest{Email: email, Password: password}), &u)
	return u
}

func TestHealthAndSwagger(t *testing.T) {
	base := newTestServer(t)
	c := newClient(t, base)
	var h models.HealthStatus
	c.ok(c.do("GET", "/api/v1/health", nil), &h)
	if h.Status != "ok" || h.DB != "ok" {
		t.Errorf("%+v", h)
	}
	resp, err := http.Get(base + "/swagger/doc.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	doc, _ := io.ReadAll(resp.Body)
	for _, p := range []string{"/claims/process", "/tenants/{id}/rollback", "/tenants/diff", "/auth/login"} {
		if !strings.Contains(string(doc), p) {
			t.Errorf("swagger doc misses %s (run swag init)", p)
		}
	}
	c.fails(c.do("GET", "/api/v1/nope", nil), 404, "NOT_FOUND")
}

func TestAuthFlow(t *testing.T) {
	c := newClient(t, newTestServer(t))
	c.fails(c.do("GET", "/api/v1/tenants", nil), 401, "UNAUTHENTICATED")
	c.fails(c.do("POST", "/api/v1/auth/login", models.LoginRequest{Email: "admin@papaya.com", Password: "wrong"}), 401, "INVALID_CREDENTIALS")
	c.fails(c.do("POST", "/api/v1/auth/login", models.LoginRequest{Email: "nobody@papaya.com", Password: "x"}), 401, "INVALID_CREDENTIALS")
	c.fails(c.do("POST", "/api/v1/auth/login", map[string]any{"email": "a", "unknown": 1}), 400, "BAD_REQUEST")

	u := c.login("Admin@Papaya.com", "123456")
	if u.Role.Code != "admin" || u.Role.Name != "Quản trị viên" || len(u.Permissions) != 6 {
		t.Errorf("%+v", u)
	}
	if u.PermissionNames[models.PermTenantsManage] == "" {
		t.Error("permission names not localized")
	}

	var me models.SessionUser
	c.ok(c.do("PUT", "/api/v1/auth/me/locale", models.LocaleRequest{Locale: "en"}), &me)
	if me.Role.Name != "Administrator" {
		t.Errorf("role name after switching to en = %q", me.Role.Name)
	}
	c.fails(c.do("PUT", "/api/v1/auth/me/locale", models.LocaleRequest{Locale: "fr"}), 400, "LOCALE_UNSUPPORTED")

	c.ok(c.do("POST", "/api/v1/auth/logout", nil), nil)
	c.fails(c.do("GET", "/api/v1/auth/me", nil), 401, "UNAUTHENTICATED")
}

func TestAccountLockout(t *testing.T) {
	c := newClient(t, newTestServer(t))
	for i := 0; i < 5; i++ {
		c.fails(c.do("POST", "/api/v1/auth/login", models.LoginRequest{Email: "viewer@papaya.com", Password: "bad"}), 401, "INVALID_CREDENTIALS")
	}
	c.fails(c.do("POST", "/api/v1/auth/login", models.LoginRequest{Email: "viewer@papaya.com", Password: "Viewer@2026!"}), 423, "ACCOUNT_LOCKED")
}

func TestPermissionsAndSameOrigin(t *testing.T) {
	base := newTestServer(t)
	viewer := newClient(t, base)
	viewer.login("viewer@papaya.com", "Viewer@2026!")
	var list models.TenantList
	viewer.ok(viewer.do("GET", "/api/v1/tenants", nil), &list)
	if len(list.Tenants) != 3 {
		t.Fatalf("seeded tenants = %d", len(list.Tenants))
	}
	viewer.fails(viewer.do("POST", "/api/v1/tenants", models.CreateTenantRequest{Slug: "x"}), 403, "FORBIDDEN")
	viewer.fails(viewer.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{}), 403, "FORBIDDEN")

	admin := newClient(t, base)
	admin.login("admin@papaya.com", "123456")
	admin.fails(admin.do("DELETE", "/api/v1/tenants/safeguard", nil, "Origin", "https://evil.example"), 403, "CROSS_ORIGIN")
	admin.ok(admin.do("GET", "/api/v1/tenants/safeguard", nil, "Origin", "https://evil.example"), nil) // reads are not blocked
}

func batchClaim() models.Claim {
	return models.Claim{
		ClaimType: "OUTPATIENT",
		Amount:    15000,
		CustomFields: map[string]any{
			"employee_id": "EMP-12345", "department": "Finance", "budget_code": "FIN-2026",
		},
		Documents: []string{"invoice", "prescription"},
	}
}

func TestSameClaimThreeTenantsBehaveDifferently(t *testing.T) {
	c := newClient(t, newTestServer(t))
	c.login("operator@papaya.com", "Operator@2026")
	var batch models.BatchResult
	c.ok(c.do("POST", "/api/v1/claims/process-batch", models.ProcessBatchRequest{
		TenantIDs: []string{"safeguard", "healthfirst", "govhealth"}, Claim: batchClaim(),
	}), &batch)
	want := map[string]string{"safeguard": "auto_approved:", "healthfirst": "manual_review:assessor", "govhealth": "manual_review:committee"}
	for _, r := range batch.Results {
		got := r.Approval.Routing + ":" + r.Approval.ApproverRole
		if got != want[r.Tenant.Slug] || r.ClaimID == "" || r.SLA == nil {
			t.Errorf("%s: %s (claim %q)", r.Tenant.Slug, got, r.ClaimID)
		}
		if len(r.Trace) == 0 || r.Trace[0].Text == "" {
			t.Errorf("%s: trace not localized", r.Tenant.Slug)
		}
	}
	var claims models.ClaimList
	c.ok(c.do("GET", "/api/v1/claims", nil), &claims)
	if len(claims.Claims) != 3 {
		t.Errorf("stored claims = %d", len(claims.Claims))
	}

	var rejected models.ProcessResult
	dental := batchClaim()
	dental.ClaimType = "DENTAL"
	c.ok(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "govhealth", Claim: dental}), &rejected)
	if rejected.Accepted || len(rejected.Errors) != 1 || rejected.Errors[0].Code != "CLAIM_TYPE_NOT_ENABLED" || rejected.Errors[0].Text == "" {
		t.Errorf("%+v", rejected.Errors)
	}
	bad := batchClaim()
	bad.ClaimType = "VISION"
	c.fails(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "govhealth", Claim: bad}), 400, "CLAIM_TYPE_UNKNOWN")
	c.fails(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "nope", Claim: batchClaim()}), 404, "TENANT_NOT_FOUND")
}

func TestOnboardFourthTenantWithoutCodeChanges(t *testing.T) {
	c := newClient(t, newTestServer(t))
	c.login("operator@papaya.com", "Operator@2026")

	var cfg models.TenantConfig
	c.ok(c.do("GET", "/api/v1/configs/defaults", nil), &cfg)
	cfg.Branding.CompanyName = "CareLink Insurance"
	cfg.ClaimTypes.Enabled = []string{"OUTPATIENT", "OPTICAL"}
	cfg.ClaimTypes.Documents["OPTICAL"] = models.DocumentRules{Required: []string{"invoice", "eye_prescription"}}
	cfg.SLA.Targets = map[string]int{"OUTPATIENT": 3, "OPTICAL": 3}
	cfg.CustomFields = []models.CustomField{{Key: "policy_number", Label: "Policy Number", Type: "text", Required: true, Pattern: "^CL-[0-9]{6}$"}}

	invalid := cfg
	invalid.Approval.AutoApprovalThreshold = -1
	env := c.do("POST", "/api/v1/tenants", models.CreateTenantRequest{Slug: "carelink", Config: &invalid})
	c.fails(env, 400, "CONFIG_INVALID")
	var vr models.ValidationResult
	_ = json.Unmarshal(env.Data[0], &vr)
	if len(vr.Issues) == 0 || vr.Issues[0].Text == "" {
		t.Fatalf("issues = %+v", vr.Issues)
	}

	var check models.ValidationResult
	c.ok(c.do("POST", "/api/v1/configs/validate", models.ValidateConfigRequest{Config: &cfg}), &check)
	if !check.Valid {
		t.Fatalf("%+v", check.Issues)
	}

	var created models.TenantDetail
	c.ok(c.do("POST", "/api/v1/tenants", models.CreateTenantRequest{Slug: "carelink", Config: &cfg, Note: "Onboard"}), &created)
	if created.CurrentVersion != 1 || created.Name != "CareLink Insurance" {
		t.Errorf("%+v", created.Tenant)
	}
	c.fails(c.do("POST", "/api/v1/tenants", models.CreateTenantRequest{Slug: "carelink", Config: &cfg}), 409, "TENANT_SLUG_TAKEN")
	c.fails(c.do("POST", "/api/v1/tenants", models.CreateTenantRequest{Slug: "Bad Slug", Config: &cfg}), 400, "TENANT_SLUG_INVALID")

	var res models.ProcessResult
	claim := models.Claim{ClaimType: "OPTICAL", Amount: 2500, CustomFields: map[string]any{"policy_number": "CL-123456"}}
	c.ok(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "carelink", Claim: claim}), &res)
	if !res.Accepted || res.Approval.ApproverRole != "assessor" || res.SLA.BusinessDays != 3 {
		t.Errorf("%+v", res)
	}
}

func TestHistoryRollbackAndDiff(t *testing.T) {
	c := newClient(t, newTestServer(t))
	c.login("admin@papaya.com", "123456")

	var sg models.TenantDetail
	c.ok(c.do("GET", "/api/v1/tenants/safeguard", nil), &sg)
	cfg := sg.Config
	cfg.Approval.AutoApprovalThreshold = 30000
	cfg.Approval.Tiers[0].MinAmount = 30000

	var v2 models.TenantDetail
	c.ok(c.do("PUT", "/api/v1/tenants/safeguard", models.UpdateTenantRequest{Config: &cfg, Note: "raise", BaseVersion: 1}), &v2)
	if v2.CurrentVersion != 2 || v2.Config.Approval.AutoApprovalThreshold != 30000 {
		t.Fatalf("%+v", v2.Tenant)
	}
	c.fails(c.do("PUT", "/api/v1/tenants/safeguard", models.UpdateTenantRequest{Config: &cfg, BaseVersion: 1}), 409, "VERSION_CONFLICT")
	c.fails(c.do("PUT", "/api/v1/tenants/safeguard", models.UpdateTenantRequest{Config: &cfg}), 400, "VERSION_REQUIRED")

	var diff models.DiffResult
	c.ok(c.do("GET", "/api/v1/tenants/diff?a=safeguard@1&b=safeguard@2", nil), &diff)
	if len(diff.Changes) != 2 || diff.A.Version != 1 || diff.B.Version != 2 {
		t.Errorf("%+v", diff)
	}

	var v3 models.TenantDetail
	c.ok(c.do("POST", "/api/v1/tenants/safeguard/rollback", models.RollbackRequest{Version: 1}), &v3)
	if v3.CurrentVersion != 3 || v3.Config.Approval.AutoApprovalThreshold != 20000 {
		t.Fatalf("%+v", v3.Tenant)
	}
	c.fails(c.do("POST", "/api/v1/tenants/safeguard/rollback", models.RollbackRequest{Version: 9}), 404, "VERSION_NOT_FOUND")

	var hist models.VersionList
	c.ok(c.do("GET", "/api/v1/tenants/safeguard/versions", nil), &hist)
	sources := []string{}
	for _, v := range hist.Versions {
		sources = append(sources, v.Source)
	}
	if strings.Join(sources, ",") != "rollback,update,create" || !hist.Versions[0].IsCurrent || *hist.Versions[0].RolledBackFrom != 1 {
		t.Errorf("history = %v", sources)
	}
	if hist.Versions[1].CreatedByEmail == nil || *hist.Versions[1].CreatedByEmail != "admin@papaya.com" {
		t.Error("created_by_email missing")
	}

	c.ok(c.do("GET", "/api/v1/tenants/diff?a=safeguard@1&b=safeguard", nil), &diff)
	if len(diff.Changes) != 0 {
		t.Errorf("v1 vs current after rollback: %+v", diff.Changes)
	}

	c.ok(c.do("DELETE", "/api/v1/tenants/safeguard", nil), nil)
	c.fails(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "safeguard", Claim: batchClaim()}), 409, "TENANT_ARCHIVED")
	var list models.TenantList
	c.ok(c.do("GET", "/api/v1/tenants", nil), &list)
	if len(list.Tenants) != 2 {
		t.Errorf("active tenants after archive = %d", len(list.Tenants))
	}
}

func TestPreviewUnsavedConfig(t *testing.T) {
	c := newClient(t, newTestServer(t))
	c.login("viewer@papaya.com", "Viewer@2026!")
	var hf models.TenantDetail
	c.ok(c.do("GET", "/api/v1/tenants/healthfirst", nil), &hf)
	cfg := hf.Config
	cfg.Approval.AutoApprovalThreshold = 20000
	cfg.Approval.Tiers[0].MinAmount = 20000

	var saved, unsaved models.ProcessResult
	c.ok(c.do("POST", "/api/v1/claims/preview", models.PreviewClaimRequest{TenantID: "healthfirst", Claim: batchClaim()}), &saved)
	c.ok(c.do("POST", "/api/v1/claims/preview", models.PreviewClaimRequest{TenantID: "healthfirst", Config: &cfg, Claim: batchClaim()}), &unsaved)
	if saved.Approval.Routing != "manual_review" || unsaved.Approval.Routing != "auto_approved" || unsaved.Tenant.ConfigVersion != 0 {
		t.Errorf("saved=%s unsaved=%s v=%d", saved.Approval.Routing, unsaved.Approval.Routing, unsaved.Tenant.ConfigVersion)
	}
	c.fails(c.do("POST", "/api/v1/claims/preview", models.PreviewClaimRequest{Claim: batchClaim()}), 400, "TENANT_REQUIRED")

	var claims models.ClaimList
	c.fails(c.do("POST", "/api/v1/claims/process", models.ProcessClaimRequest{TenantID: "healthfirst", Claim: batchClaim()}), 403, "FORBIDDEN")
	c.ok(c.do("GET", "/api/v1/claims", nil), &claims)
	if len(claims.Claims) != 0 {
		t.Error("preview must not store claims")
	}
}
