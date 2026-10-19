package repositories

import (
	"context"
	"encoding/json"

	"claimconfig/component/models"

	"github.com/jmoiron/sqlx"
)

// ClaimRepo stores claims processed by the runtime API.
type ClaimRepo struct{ db *sqlx.DB }

func NewClaimRepo(db *sqlx.DB) *ClaimRepo { return &ClaimRepo{db: db} }

// NewClaim is one processed claim to store.
type NewClaim struct {
	ID          string
	Claim       models.Claim
	SubmittedAt string
	Result      models.ProcessResult
	CreatedBy   *int64
	CreatedAt   string
}

func (r *ClaimRepo) Create(ctx context.Context, c NewClaim) error {
	fields, err := json.Marshal(nonNilMap(c.Claim.CustomFields))
	if err != nil {
		return err
	}
	docs, err := json.Marshal(nonNilList(c.Claim.Documents))
	if err != nil {
		return err
	}
	result, err := json.Marshal(c.Result)
	if err != nil {
		return err
	}
	var role, deadline *string
	if c.Result.Approval.ApproverRole != "" {
		role = &c.Result.Approval.ApproverRole
	}
	if c.Result.SLA != nil {
		deadline = &c.Result.SLA.Deadline
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO claim (id, tenant_id, config_version, claim_type, amount, submitted_at, custom_fields, documents,
                   accepted, routing, approver_role, sla_deadline, result, created_by, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Result.Tenant.ID, c.Result.Tenant.ConfigVersion, c.Claim.ClaimType, c.Claim.Amount, c.SubmittedAt,
		string(fields), string(docs), c.Result.Accepted, c.Result.Approval.Routing, role, deadline,
		string(result), c.CreatedBy, c.CreatedAt)
	return err
}

// List returns the newest claims, optionally for one tenant, with results.
func (r *ClaimRepo) List(ctx context.Context, tenantID string, limit int) ([]models.ClaimRecord, error) {
	q := `
SELECT c.id, c.tenant_id, t.slug AS tenant_slug, c.config_version, c.claim_type, c.amount, c.submitted_at,
       c.accepted, c.routing, c.approver_role, c.sla_deadline, c.created_by, c.created_at, c.result
FROM claim c JOIN tenant t ON t.id = c.tenant_id`
	args := []any{}
	if tenantID != "" {
		q += ` WHERE c.tenant_id = ?`
		args = append(args, tenantID)
	}
	q += ` ORDER BY c.created_at DESC, c.rowid DESC LIMIT ?`
	args = append(args, limit)
	out := []models.ClaimRecord{}
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, err
	}
	for i := range out {
		var res models.ProcessResult
		if err := json.Unmarshal([]byte(out[i].ResultJSON), &res); err == nil {
			out[i].Result = &res
		}
	}
	return out, nil
}

func nonNilMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func nonNilList(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}
