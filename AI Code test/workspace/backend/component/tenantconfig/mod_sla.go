package tenantconfig

import (
	"strconv"
	"strings"
	"time"

	"claimconfig/component/models"
)

// SLAModule owns target processing time per claim type (business days) and
// escalation rules, and computes each claim's deadline.
type SLAModule struct{}

func (SLAModule) Key() string { return "sla" }

func (SLAModule) Validate(cfg *models.TenantConfig) []models.Issue {
	s := cfg.SLA
	var out []models.Issue
	if _, err := time.LoadLocation(s.Timezone); s.Timezone == "" || err != nil {
		out = append(out, issue("sla.timezone", "TIMEZONE_INVALID", map[string]any{"timezone": s.Timezone}))
	}
	if s.DefaultDays < 0 {
		out = append(out, issue("sla.default_days", "SLA_NOT_POSITIVE", nil))
	}
	for _, t := range sortedKeys(s.Targets) {
		days := s.Targets[t]
		path := "sla.targets." + t
		if !contains(models.ClaimTypes, t) {
			out = append(out, issue(path, "CLAIM_TYPE_UNKNOWN", map[string]any{"claim_type": t}))
		} else if days <= 0 {
			out = append(out, issue(path, "SLA_NOT_POSITIVE", nil))
		}
	}
	for _, t := range cfg.ClaimTypes.Enabled {
		if _, ok := s.Targets[t]; !ok && s.DefaultDays <= 0 && contains(models.ClaimTypes, t) {
			out = append(out, issue("sla.targets."+t, "SLA_MISSING", map[string]any{"claim_type": t}))
		}
	}
	for i, e := range s.Escalation {
		path := idx("sla.escalation", i)
		if e.AfterBreachDays <= 0 {
			out = append(out, issue(path+".after_breach_days", "ESCALATION_DAYS_INVALID", nil))
		}
		empty := len(e.Notify) == 0
		for _, n := range e.Notify {
			if strings.TrimSpace(n) == "" {
				empty = true
			}
		}
		if empty {
			out = append(out, issue(path+".notify", "ESCALATION_NOTIFY_EMPTY", nil))
		}
	}
	return out
}

func (SLAModule) Process(cfg *models.TenantConfig, claim *models.Claim, ctx *Ctx, res *models.ProcessResult) {
	if !res.Accepted {
		return
	}
	s := cfg.SLA
	days, ok := s.Targets[claim.ClaimType]
	if !ok {
		days = s.DefaultDays
	}
	if days <= 0 {
		reject(res, "sla.targets."+claim.ClaimType, "SLA_MISSING", map[string]any{"claim_type": claim.ClaimType})
		return
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		loc = time.UTC
	}
	start := ctx.SubmittedAt.In(loc)
	deadline := AddBusinessDays(start, days)
	res.SLA = &models.SLAResult{
		BusinessDays: days,
		Timezone:     loc.String(),
		SubmittedAt:  start.Format(time.RFC3339),
		Deadline:     deadline.Format(time.RFC3339),
		Escalation:   append([]models.EscalationRule{}, s.Escalation...),
	}
	trace(res, "sla", "SLA_DEADLINE", map[string]any{"days": days, "deadline": deadline.Format("2006-01-02")})
}

func (SLAModule) DiffKeys() map[string]string { return nil }

func formatAmount(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
