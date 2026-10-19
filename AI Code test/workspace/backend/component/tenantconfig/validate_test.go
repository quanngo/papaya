package tenantconfig

import (
	"testing"

	"claimconfig/component/models"
)

func ptr(v float64) *float64 { return &v }

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()
	if issues := ValidateConfig(&cfg); len(issues) != 0 {
		t.Fatalf("%+v", issues)
	}
}

func TestValidateConfig_Issues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(c *models.TenantConfig)
		path   string
		code   string
	}{
		{"negative threshold", func(c *models.TenantConfig) {
			c.Approval.AutoApprovalThreshold = -1
			c.Approval.Tiers[0].MinAmount = -1
		}, "approval.auto_approval_threshold", "THRESHOLD_NEGATIVE"},
		{"no claim type", func(c *models.TenantConfig) { c.ClaimTypes.Enabled = nil }, "claim_types.enabled", "NO_CLAIM_TYPE"},
		{"SLA zero", func(c *models.TenantConfig) { c.SLA.Targets["OUTPATIENT"] = 0 }, "sla.targets.OUTPATIENT", "SLA_NOT_POSITIVE"},
		{"SLA missing for enabled type", func(c *models.TenantConfig) {
			c.ClaimTypes.Enabled = append(c.ClaimTypes.Enabled, "DENTAL")
			c.ClaimTypes.Documents["DENTAL"] = models.DocumentRules{Required: []string{"invoice"}}
		}, "sla.targets.DENTAL", "SLA_MISSING"},
		{"documents missing for enabled type", func(c *models.TenantConfig) {
			c.ClaimTypes.Enabled = append(c.ClaimTypes.Enabled, "OPTICAL")
			c.SLA.Targets["OPTICAL"] = 3
		}, "claim_types.documents.OPTICAL", "DOCUMENTS_MISSING"},
		{"unknown claim type", func(c *models.TenantConfig) { c.ClaimTypes.Enabled = []string{"VISION"} }, "claim_types.enabled[0]", "CLAIM_TYPE_UNKNOWN"},
		{"first tier not at threshold", func(c *models.TenantConfig) { c.Approval.AutoApprovalThreshold = 100 }, "approval.tiers[0].min_amount", "TIER_START_MISMATCH"},
		{"tier gap", func(c *models.TenantConfig) {
			c.Approval.Tiers = []models.Tier{{MinAmount: 0, MaxAmount: ptr(100), Role: "a"}, {MinAmount: 200, Role: "b"}}
		}, "approval.tiers[1].min_amount", "TIER_GAP"},
		{"tier overlap", func(c *models.TenantConfig) {
			c.Approval.Tiers = []models.Tier{{MinAmount: 0, MaxAmount: ptr(100), Role: "a"}, {MinAmount: 50, Role: "b"}}
		}, "approval.tiers[1].min_amount", "TIER_OVERLAP"},
		{"last tier bounded", func(c *models.TenantConfig) { c.Approval.Tiers[0].MaxAmount = ptr(10) }, "approval.tiers[0].max_amount", "TIER_LAST_BOUNDED"},
		{"no tier", func(c *models.TenantConfig) { c.Approval.Tiers = nil }, "approval.tiers", "NO_TIER"},
		{"webhook without URL", func(c *models.TenantConfig) {
			c.Notifications.Events["approved"] = models.EventRule{Enabled: true, Channels: []string{"webhook"}}
		}, "notifications.webhook_url", "WEBHOOK_URL_REQUIRED"},
		{"enabled event without channel", func(c *models.TenantConfig) {
			c.Notifications.Events["approved"] = models.EventRule{Enabled: true}
		}, "notifications.events.approved.channels", "EVENT_NO_CHANNEL"},
		{"bad color", func(c *models.TenantConfig) { c.Branding.PrimaryColor = "blue" }, "branding.primary_color", "COLOR_INVALID"},
		{"bad timezone", func(c *models.TenantConfig) { c.SLA.Timezone = "Mars/Base" }, "sla.timezone", "TIMEZONE_INVALID"},
		{"duplicate field key", func(c *models.TenantConfig) {
			c.CustomFields = []models.CustomField{{Key: "a", Label: "A", Type: "text"}, {Key: "a", Label: "B", Type: "text"}}
		}, "custom_fields[1].key", "FIELD_KEY_DUPLICATE"},
		{"bad regex", func(c *models.TenantConfig) {
			c.CustomFields = []models.CustomField{{Key: "a", Label: "A", Type: "text", Pattern: "("}}
		}, "custom_fields[0].pattern", "FIELD_PATTERN_INVALID"},
		{"select without options", func(c *models.TenantConfig) {
			c.CustomFields = []models.CustomField{{Key: "a", Label: "A", Type: "select"}}
		}, "custom_fields[0].options", "FIELD_OPTIONS_REQUIRED"},
		{"document both required and optional", func(c *models.TenantConfig) {
			c.ClaimTypes.Documents["OUTPATIENT"] = models.DocumentRules{Required: []string{"invoice"}, Optional: []string{"invoice"}}
		}, "claim_types.documents.OUTPATIENT.required[0]", "DOCUMENT_BOTH_REQUIRED_OPTIONAL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := DefaultConfig()
			c.mutate(&cfg)
			issues := ValidateConfig(&cfg)
			for _, i := range issues {
				if i.Path == c.path && i.Code == c.code {
					return
				}
			}
			t.Errorf("want %s at %s, got %+v", c.code, c.path, issues)
		})
	}
}
