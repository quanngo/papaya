package tenantconfig

import (
	"strings"

	"claimconfig/component/models"
)

// ClaimTypesModule owns which claim types are enabled and the documents each
// one requires. It rejects claims of a type the tenant does not offer.
type ClaimTypesModule struct{}

func (ClaimTypesModule) Key() string { return "claim_types" }

func (ClaimTypesModule) Validate(cfg *models.TenantConfig) []models.Issue {
	ct := cfg.ClaimTypes
	var out []models.Issue
	if len(ct.Enabled) == 0 {
		out = append(out, issue("claim_types.enabled", "NO_CLAIM_TYPE", nil))
	}
	seen := map[string]bool{}
	for i, t := range ct.Enabled {
		switch {
		case !contains(models.ClaimTypes, t):
			out = append(out, issue(idx("claim_types.enabled", i), "CLAIM_TYPE_UNKNOWN", map[string]any{"claim_type": t}))
		case seen[t]:
			out = append(out, issue(idx("claim_types.enabled", i), "CLAIM_TYPE_DUPLICATE", map[string]any{"claim_type": t}))
		}
		seen[t] = true
	}
	for _, t := range ct.Enabled {
		if _, ok := ct.Documents[t]; !ok && contains(models.ClaimTypes, t) {
			out = append(out, issue("claim_types.documents."+t, "DOCUMENTS_MISSING", map[string]any{"claim_type": t}))
		}
	}
	for _, t := range sortedKeys(ct.Documents) {
		rules := ct.Documents[t]
		path := "claim_types.documents." + t
		if !contains(models.ClaimTypes, t) {
			out = append(out, issue(path, "CLAIM_TYPE_UNKNOWN", map[string]any{"claim_type": t}))
			continue
		}
		for i, d := range rules.Required {
			if strings.TrimSpace(d) == "" {
				out = append(out, issue(idx(path+".required", i), "DOCUMENT_NAME_EMPTY", nil))
			} else if contains(rules.Optional, d) {
				out = append(out, issue(idx(path+".required", i), "DOCUMENT_BOTH_REQUIRED_OPTIONAL", map[string]any{"document": d}))
			}
		}
		for i, d := range rules.Optional {
			if strings.TrimSpace(d) == "" {
				out = append(out, issue(idx(path+".optional", i), "DOCUMENT_NAME_EMPTY", nil))
			}
		}
	}
	return out
}

func (ClaimTypesModule) Process(cfg *models.TenantConfig, claim *models.Claim, _ *Ctx, res *models.ProcessResult) {
	if !contains(cfg.ClaimTypes.Enabled, claim.ClaimType) {
		reject(res, "claim.claim_type", "CLAIM_TYPE_NOT_ENABLED", map[string]any{
			"claim_type": claim.ClaimType,
			"enabled":    strings.Join(cfg.ClaimTypes.Enabled, ", "),
		})
		return
	}
	trace(res, "claim_types", "CLAIM_TYPE_ENABLED", map[string]any{"claim_type": claim.ClaimType})

	rules := cfg.ClaimTypes.Documents[claim.ClaimType]
	res.Documents.Required = append([]string{}, rules.Required...)
	res.Documents.Optional = append([]string{}, rules.Optional...)
	for _, d := range rules.Required {
		if !contains(claim.Documents, d) {
			res.Documents.Missing = append(res.Documents.Missing, d)
		}
	}
	trace(res, "claim_types", "DOCUMENTS_REQUIRED", map[string]any{
		"count":   len(rules.Required),
		"missing": len(res.Documents.Missing),
	})
}

func (ClaimTypesModule) DiffKeys() map[string]string { return nil }
