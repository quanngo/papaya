package tenantconfig

import (
	"strings"

	"claimconfig/component/models"
)

// ApprovalModule owns the auto-approval threshold and the approval tiers.
// Claims below the threshold are auto-approved; the rest go to the role of
// the tier whose [min, max) range contains the amount.
type ApprovalModule struct{}

func (ApprovalModule) Key() string { return "approval" }

func (ApprovalModule) Validate(cfg *models.TenantConfig) []models.Issue {
	a := cfg.Approval
	var out []models.Issue
	if a.AutoApprovalThreshold < 0 {
		out = append(out, issue("approval.auto_approval_threshold", "THRESHOLD_NEGATIVE", nil))
	}
	if len(a.Tiers) == 0 {
		return append(out, issue("approval.tiers", "NO_TIER", nil))
	}
	if a.Tiers[0].MinAmount != a.AutoApprovalThreshold {
		out = append(out, issue("approval.tiers[0].min_amount", "TIER_START_MISMATCH",
			map[string]any{"expected": a.AutoApprovalThreshold}))
	}
	last := len(a.Tiers) - 1
	for i, t := range a.Tiers {
		path := idx("approval.tiers", i)
		if strings.TrimSpace(t.Role) == "" {
			out = append(out, issue(path+".role", "TIER_ROLE_EMPTY", nil))
		}
		if t.MaxAmount == nil && i != last {
			out = append(out, issue(path+".max_amount", "TIER_UNBOUNDED_NOT_LAST", nil))
		}
		if t.MaxAmount != nil && *t.MaxAmount <= t.MinAmount {
			out = append(out, issue(path+".max_amount", "TIER_RANGE_INVALID", nil))
		}
		if i == last && t.MaxAmount != nil {
			out = append(out, issue(path+".max_amount", "TIER_LAST_BOUNDED", nil))
		}
		if i > 0 && a.Tiers[i-1].MaxAmount != nil {
			prevMax := *a.Tiers[i-1].MaxAmount
			switch {
			case t.MinAmount > prevMax:
				out = append(out, issue(path+".min_amount", "TIER_GAP", map[string]any{"expected": prevMax}))
			case t.MinAmount < prevMax:
				out = append(out, issue(path+".min_amount", "TIER_OVERLAP", map[string]any{"expected": prevMax}))
			}
		}
	}
	return out
}

func (ApprovalModule) Process(cfg *models.TenantConfig, claim *models.Claim, _ *Ctx, res *models.ProcessResult) {
	if !res.Accepted {
		return
	}
	a := cfg.Approval
	res.Approval.Threshold = a.AutoApprovalThreshold
	if claim.Amount < a.AutoApprovalThreshold {
		res.Approval.Routing = models.RoutingAutoApproved
		res.Approval.AutoApproved = true
		trace(res, "approval", "AUTO_APPROVED", map[string]any{"amount": claim.Amount, "threshold": a.AutoApprovalThreshold})
		return
	}
	for i, t := range a.Tiers {
		if claim.Amount >= t.MinAmount && (t.MaxAmount == nil || claim.Amount < *t.MaxAmount) {
			tier, n := t, i
			res.Approval.Routing = models.RoutingManualReview
			res.Approval.ApproverRole = t.Role
			res.Approval.Tier = &tier
			res.Approval.TierIndex = &n
			// Numbers stay numbers: the HTTP layer formats them for the reader's language.
			if t.MaxAmount == nil {
				trace(res, "approval", "TIER_MATCHED_OPEN", map[string]any{"amount": claim.Amount, "min": t.MinAmount, "role": t.Role})
			} else {
				trace(res, "approval", "TIER_MATCHED", map[string]any{"amount": claim.Amount, "min": t.MinAmount, "max": *t.MaxAmount, "role": t.Role})
			}
			return
		}
	}
	reject(res, "claim.amount", "NO_MATCHING_TIER", map[string]any{"amount": claim.Amount})
}

func (ApprovalModule) DiffKeys() map[string]string {
	return map[string]string{"approval.tiers": "role"}
}
