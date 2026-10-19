package tenantconfig

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"claimconfig/component/models"
	"claimconfig/seed"
)

func loadSeeds(t *testing.T) map[string]*models.TenantConfig {
	t.Helper()
	seeds, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*models.TenantConfig{}
	for _, s := range seeds {
		var cfg models.TenantConfig
		dec := json.NewDecoder(strings.NewReader(string(s.Config)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			t.Fatalf("seed %s: %v", s.Slug, err)
		}
		out[s.Slug] = &cfg
	}
	return out
}

func ts(t *testing.T, s string) *time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return &v
}

// The same claim contains every tenant's custom fields, so it is identical
// for all three tenants; extra fields are ignored by the ones that don't use them.
func sameClaim(t *testing.T, claimType string, amount float64) models.Claim {
	return models.Claim{
		ClaimType:   claimType,
		Amount:      amount,
		SubmittedAt: ts(t, "2026-10-16T09:00:00+07:00"), // Friday
		CustomFields: map[string]any{
			"employee_id": "EMP-12345",
			"department":  "Finance",
			"budget_code": "FIN-2026",
		},
		Documents: []string{"invoice", "prescription"},
	}
}

func process(cfg *models.TenantConfig, slug string, c models.Claim) models.ProcessResult {
	return ProcessClaim(cfg, models.TenantRef{Slug: slug, ConfigVersion: 1}, c, time.Now())
}

func TestSeedConfigsAreValid(t *testing.T) {
	for slug, cfg := range loadSeeds(t) {
		if issues := ValidateConfig(cfg); len(issues) != 0 {
			t.Errorf("%s: %+v", slug, issues)
		}
	}
}

func TestProcessClaim_SameClaimThreeTenants(t *testing.T) {
	seeds := loadSeeds(t)
	type want struct {
		routing  string
		role     string
		notified map[string][]string
		deadline string
		fields   []string
	}
	cases := map[string]want{
		"safeguard": {
			routing:  models.RoutingAutoApproved,
			notified: map[string][]string{"claim_submitted": {"email"}, "approved": {"email"}},
			deadline: "2026-10-23T09:00:00+07:00",
			fields:   []string{"employee_id"},
		},
		"healthfirst": {
			routing:  models.RoutingManualReview,
			role:     "assessor",
			notified: map[string][]string{"claim_submitted": {"email", "sms"}},
			deadline: "2026-10-27T09:00:00+07:00",
			fields:   []string{},
		},
		"govhealth": {
			routing:  models.RoutingManualReview,
			role:     "committee",
			notified: map[string][]string{"claim_submitted": {"email", "webhook"}},
			deadline: "2026-11-06T09:00:00+07:00",
			fields:   []string{"department", "budget_code"},
		},
	}
	for slug, w := range cases {
		t.Run(slug, func(t *testing.T) {
			r := process(seeds[slug], slug, sameClaim(t, "OUTPATIENT", 15000))
			if !r.Accepted || len(r.Errors) != 0 {
				t.Fatalf("not accepted: %+v", r.Errors)
			}
			if r.Approval.Routing != w.routing || r.Approval.ApproverRole != w.role {
				t.Errorf("routing = %s/%s, want %s/%s", r.Approval.Routing, r.Approval.ApproverRole, w.routing, w.role)
			}
			got := map[string][]string{}
			for _, n := range r.Notifications {
				got[n.Event] = n.Channels
			}
			if !reflect.DeepEqual(got, w.notified) {
				t.Errorf("notifications = %v, want %v", got, w.notified)
			}
			if r.SLA == nil || r.SLA.Deadline != w.deadline {
				t.Errorf("SLA = %+v, want deadline %s", r.SLA, w.deadline)
			}
			keys := []string{}
			for _, f := range r.CustomFields {
				keys = append(keys, f.Key)
			}
			if !reflect.DeepEqual(keys, w.fields) {
				t.Errorf("custom fields = %v, want %v", keys, w.fields)
			}
			if len(r.Trace) == 0 {
				t.Error("empty trace")
			}
		})
	}
}

func TestProcessClaim_InpatientTiers(t *testing.T) {
	seeds := loadSeeds(t)
	for slug, role := range map[string]string{"safeguard": "team_lead", "healthfirst": "manager", "govhealth": "committee"} {
		r := process(seeds[slug], slug, sameClaim(t, "INPATIENT", 150000))
		if r.Approval.ApproverRole != role {
			t.Errorf("%s: role = %q, want %q", slug, r.Approval.ApproverRole, role)
		}
	}
}

func TestProcessClaim_Boundaries(t *testing.T) {
	seeds := loadSeeds(t)
	sg, gov := seeds["safeguard"], seeds["govhealth"]
	cases := []struct {
		name    string
		cfg     *models.TenantConfig
		amount  float64
		routing string
		role    string
	}{
		{"just below threshold", sg, 19999.99, models.RoutingAutoApproved, ""},
		{"exactly threshold goes to first tier", sg, 20000, models.RoutingManualReview, "assessor"},
		{"tier max is exclusive", sg, 100000, models.RoutingManualReview, "team_lead"},
		{"last tier", sg, 500000, models.RoutingManualReview, "director"},
		{"threshold 0: zero amount still reviewed", gov, 0, models.RoutingManualReview, "committee"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := process(c.cfg, "x", sameClaim(t, "OUTPATIENT", c.amount))
			if r.Approval.Routing != c.routing || r.Approval.ApproverRole != c.role {
				t.Errorf("got %s/%s, want %s/%s", r.Approval.Routing, r.Approval.ApproverRole, c.routing, c.role)
			}
		})
	}
}

func TestProcessClaim_Rejections(t *testing.T) {
	seeds := loadSeeds(t)

	r := process(seeds["govhealth"], "govhealth", sameClaim(t, "DENTAL", 1000))
	assertRejected(t, r, "CLAIM_TYPE_NOT_ENABLED")

	c := sameClaim(t, "OUTPATIENT", 1000)
	delete(c.CustomFields, "employee_id")
	assertRejected(t, process(seeds["safeguard"], "safeguard", c), "FIELD_REQUIRED")

	c = sameClaim(t, "OUTPATIENT", 1000)
	c.CustomFields["budget_code"] = "fin-1"
	assertRejected(t, process(seeds["govhealth"], "govhealth", c), "FIELD_PATTERN_MISMATCH")
}

func assertRejected(t *testing.T, r models.ProcessResult, code string) {
	t.Helper()
	if r.Accepted || r.Approval.Routing != models.RoutingRejected || r.SLA != nil || len(r.Notifications) != 0 {
		t.Fatalf("expected rejection, got accepted=%v routing=%s", r.Accepted, r.Approval.Routing)
	}
	for _, e := range r.Errors {
		if e.Code == code {
			return
		}
	}
	t.Errorf("errors %+v do not contain %s", r.Errors, code)
}

func TestProcessClaim_MissingDocuments(t *testing.T) {
	r := process(loadSeeds(t)["safeguard"], "safeguard", sameClaim(t, "OUTPATIENT", 1000))
	if !reflect.DeepEqual(r.Documents.Missing, []string{"medical_report"}) {
		t.Errorf("missing = %v", r.Documents.Missing)
	}
}

func TestAddBusinessDays(t *testing.T) {
	cases := []struct {
		from string
		days int
		want string
	}{
		{"2026-10-16T09:00:00+07:00", 5, "2026-10-23T09:00:00+07:00"},  // Fri + 5
		{"2026-10-17T09:00:00+07:00", 1, "2026-10-19T09:00:00+07:00"},  // Sat + 1 = Mon
		{"2026-10-18T09:00:00+07:00", 5, "2026-10-23T09:00:00+07:00"},  // Sun + 5 = Fri
		{"2026-10-16T09:00:00+07:00", 15, "2026-11-06T09:00:00+07:00"}, // three weeks
	}
	for _, c := range cases {
		got := AddBusinessDays(*ts(t, c.from), c.days).Format(time.RFC3339)
		if got != c.want {
			t.Errorf("%s + %d = %s, want %s", c.from, c.days, got, c.want)
		}
	}
}
