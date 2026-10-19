package tenantconfig

import "claimconfig/component/models"

// Modules lists every config section in processing order: acceptance checks
// (claim type, custom fields) run before routing, SLA and notifications.
var Modules = []Module{
	BrandingModule{},
	ClaimTypesModule{},
	CustomFieldsModule{},
	ApprovalModule{},
	SLAModule{},
	NotificationsModule{},
}

// ModuleKeys returns the section names in processing order.
func ModuleKeys() []string {
	keys := make([]string, len(Modules))
	for i, m := range Modules {
		keys[i] = m.Key()
	}
	return keys
}

// ValidateConfig returns every issue of every module; empty means valid.
func ValidateConfig(cfg *models.TenantConfig) []models.Issue {
	issues := []models.Issue{}
	for _, m := range Modules {
		issues = append(issues, m.Validate(cfg)...)
	}
	return issues
}

// DefaultConfig is a minimal valid config used as the starting point of the
// "new tenant" form.
func DefaultConfig() models.TenantConfig {
	return models.TenantConfig{
		Branding: models.BrandingConfig{
			CompanyName:    "New Insurer",
			LogoURL:        "",
			PrimaryColor:   "#1E88E5",
			SecondaryColor: "#FFC107",
		},
		ClaimTypes: models.ClaimTypesConfig{
			Enabled: []string{"OUTPATIENT"},
			Documents: map[string]models.DocumentRules{
				"OUTPATIENT": {Required: []string{"invoice"}, Optional: []string{}},
			},
		},
		Approval: models.ApprovalConfig{
			AutoApprovalThreshold: 0,
			Tiers:                 []models.Tier{{MinAmount: 0, MaxAmount: nil, Role: "assessor"}},
		},
		Notifications: models.NotificationsConfig{
			Events: map[string]models.EventRule{
				"claim_submitted": {Enabled: true, Channels: []string{"email"}},
				"approved":        {Enabled: true, Channels: []string{"email"}},
				"rejected":        {Enabled: true, Channels: []string{"email"}},
				"payment_sent":    {Enabled: true, Channels: []string{"email"}},
			},
		},
		SLA: models.SLAConfig{
			Timezone:   "Asia/Ho_Chi_Minh",
			Targets:    map[string]int{"OUTPATIENT": 5},
			Escalation: []models.EscalationRule{},
		},
		CustomFields: []models.CustomField{},
	}
}
