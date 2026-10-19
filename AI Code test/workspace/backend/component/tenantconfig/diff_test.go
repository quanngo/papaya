package tenantconfig

import (
	"testing"

	"claimconfig/component/models"
)

func findChange(changes []models.Change, path, kind string) *models.Change {
	for i := range changes {
		if changes[i].Path == path && changes[i].Kind == kind {
			return &changes[i]
		}
	}
	return nil
}

func TestDiff_IdenticalIsEmpty(t *testing.T) {
	a, b := DefaultConfig(), DefaultConfig()
	if d := Diff(&a, &b); len(d) != 0 {
		t.Fatalf("%+v", d)
	}
}

func TestDiff_SeedTenants(t *testing.T) {
	s := loadSeeds(t)
	d := Diff(s["safeguard"], s["govhealth"])
	checks := []struct{ path, kind string }{
		{"approval.auto_approval_threshold", "changed"},
		{"claim_types.enabled", "removed"}, // DENTAL
		{"approval.tiers[role=assessor]", "removed"},
		{"approval.tiers[role=committee]", "added"},
		{"notifications.webhook_url", "added"},
		{"custom_fields[key=employee_id]", "removed"},
		{"custom_fields[key=budget_code]", "added"},
		{"branding.company_name", "changed"},
	}
	for _, c := range checks {
		if findChange(d, c.path, c.kind) == nil {
			t.Errorf("missing %s %s", c.kind, c.path)
		}
	}
	if c := findChange(d, "claim_types.enabled", "removed"); c != nil && c.Left != "DENTAL" {
		t.Errorf("removed enabled = %v", c.Left)
	}
	if c := findChange(d, "approval.auto_approval_threshold", "changed"); c != nil && (c.Left != 20000.0 || c.Right != 0.0) {
		t.Errorf("threshold %v -> %v", c.Left, c.Right)
	}
}

func TestDiff_OrderOfScalarListsIgnored(t *testing.T) {
	a, b := DefaultConfig(), DefaultConfig()
	a.Notifications.Events["approved"] = models.EventRule{Enabled: true, Channels: []string{"email", "sms"}}
	b.Notifications.Events["approved"] = models.EventRule{Enabled: true, Channels: []string{"sms", "email"}}
	if d := Diff(&a, &b); len(d) != 0 {
		t.Fatalf("%+v", d)
	}
}

func TestDiff_TiersMatchedByRole(t *testing.T) {
	a, b := DefaultConfig(), DefaultConfig()
	a.Approval.Tiers = []models.Tier{{MinAmount: 0, MaxAmount: ptr(100), Role: "assessor"}, {MinAmount: 100, Role: "lead"}}
	b.Approval.Tiers = []models.Tier{{MinAmount: 0, MaxAmount: ptr(200), Role: "assessor"}, {MinAmount: 200, Role: "lead"}}
	d := Diff(&a, &b)
	if findChange(d, "approval.tiers[role=assessor].max_amount", "changed") == nil ||
		findChange(d, "approval.tiers[role=lead].min_amount", "changed") == nil || len(d) != 2 {
		t.Fatalf("%+v", d)
	}
	if d[0].Module != "approval" {
		t.Errorf("module = %s", d[0].Module)
	}
}
