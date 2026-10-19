package tenantconfig

import (
	"time"

	"claimconfig/component/models"
)

// ProcessClaim decides how one claim is handled under one tenant config:
// required documents, approval routing, notifications, SLA deadline and
// custom-field validation. Preview and the runtime API both call this, so a
// preview always predicts the real outcome. now is used when the claim has
// no submitted_at.
func ProcessClaim(cfg *models.TenantConfig, tenant models.TenantRef, claim models.Claim, now time.Time) models.ProcessResult {
	res := models.ProcessResult{
		Accepted:      true,
		Errors:        []models.Issue{},
		Tenant:        tenant,
		ClaimType:     claim.ClaimType,
		Amount:        claim.Amount,
		Documents:     models.DocumentsResult{Required: []string{}, Optional: []string{}, Missing: []string{}},
		Notifications: []models.NotificationResult{},
		CustomFields:  []models.CustomFieldResult{},
		Trace:         []models.TraceStep{},
	}
	submitted := now
	if claim.SubmittedAt != nil {
		submitted = *claim.SubmittedAt
	}
	ctx := &Ctx{SubmittedAt: submitted}

	for _, m := range Modules {
		m.Process(cfg, &claim, ctx, &res)
	}

	if !res.Accepted {
		res.Approval = models.ApprovalResult{
			Routing:   models.RoutingRejected,
			Threshold: cfg.Approval.AutoApprovalThreshold,
		}
		res.SLA = nil
		res.Notifications = []models.NotificationResult{}
		trace(&res, "claim", "CLAIM_REJECTED", map[string]any{"errors": len(res.Errors)})
	}
	return res
}
