package models

import "time"

// Claim is the input to the processing engine.
type Claim struct {
	ClaimType    string         `json:"claim_type" example:"OUTPATIENT" enums:"OUTPATIENT,INPATIENT,DENTAL,MATERNITY,OPTICAL"`
	Amount       float64        `json:"amount" example:"15000" minimum:"0"`
	SubmittedAt  *time.Time     `json:"submitted_at" example:"2026-10-16T09:00:00+07:00"` // nil = now
	CustomFields map[string]any `json:"custom_fields" swaggertype:"object"`
	Documents    []string       `json:"documents" example:"invoice,prescription"`
}

// Routing values of a processed claim.
const (
	RoutingAutoApproved = "auto_approved"
	RoutingManualReview = "manual_review"
	RoutingRejected     = "rejected" // claim not accepted: type not enabled, invalid custom fields...
)

// ProcessResult is everything processClaim decides for one claim under one tenant config.
type ProcessResult struct {
	ClaimID       string               `json:"claim_id,omitempty" example:"0d6c3f0e-6a51-4d55-9a8f-93a4b0e6c1a2"` // set when the claim was stored
	Accepted      bool                 `json:"accepted" example:"true"`
	Errors        []Issue              `json:"errors"`
	Tenant        TenantRef            `json:"tenant"`
	ClaimType     string               `json:"claim_type" example:"OUTPATIENT"`
	Amount        float64              `json:"amount" example:"15000"`
	Documents     DocumentsResult      `json:"documents"`
	Approval      ApprovalResult       `json:"approval"`
	Notifications []NotificationResult `json:"notifications"`
	SLA           *SLAResult           `json:"sla"`
	CustomFields  []CustomFieldResult  `json:"custom_fields"`
	Trace         []TraceStep          `json:"trace"`
}

type TenantRef struct {
	ID            string `json:"id" example:"60244e2b-abde-4cf1-ab77-c7bc8d4cf6d9"`
	Slug          string `json:"slug" example:"safeguard"`
	Name          string `json:"name" example:"SafeGuard Insurance"`
	ConfigVersion int    `json:"config_version" example:"1"` // 0 = unsaved config (preview)
}

type DocumentsResult struct {
	Required []string `json:"required" example:"invoice,prescription,medical_report"`
	Optional []string `json:"optional" example:"lab_results"`
	Missing  []string `json:"missing" example:"medical_report"`
}

type ApprovalResult struct {
	Routing      string  `json:"routing" example:"manual_review" enums:"auto_approved,manual_review,rejected"`
	AutoApproved bool    `json:"auto_approved" example:"false"`
	ApproverRole string  `json:"approver_role,omitempty" example:"assessor"`
	Tier         *Tier   `json:"tier,omitempty"`
	TierIndex    *int    `json:"tier_index,omitempty" example:"0"`
	Threshold    float64 `json:"threshold" example:"5000"`
}

type NotificationResult struct {
	Event           string   `json:"event" example:"claim_submitted"`
	Channels        []string `json:"channels" example:"email,sms"`
	Template        string   `json:"template" example:"Your claim has been received and is being processed."`
	DefaultTemplate bool     `json:"default_template" example:"true"`
}

type SLAResult struct {
	BusinessDays int              `json:"business_days" example:"7"`
	Timezone     string           `json:"timezone" example:"Asia/Ho_Chi_Minh"`
	SubmittedAt  string           `json:"submitted_at" example:"2026-10-16T09:00:00+07:00"`
	Deadline     string           `json:"deadline" example:"2026-10-27T09:00:00+07:00"`
	Escalation   []EscalationRule `json:"escalation"`
}

type CustomFieldResult struct {
	Key      string `json:"key" example:"employee_id"`
	Label    string `json:"label" example:"Employee ID"`
	Type     string `json:"type" example:"text"`
	Required bool   `json:"required" example:"true"`
	Value    any    `json:"value" swaggertype:"string" example:"EMP-12345"`
	Valid    bool   `json:"valid" example:"true"`
	Code     string `json:"code,omitempty" example:""` // issue code when invalid
	Text     string `json:"text,omitempty" example:""`
}

// TraceStep explains one decision, for the preview screen.
type TraceStep struct {
	Module string         `json:"module" example:"approval"`
	Code   string         `json:"code" example:"TIER_MATCHED"`
	Params map[string]any `json:"params,omitempty" swaggertype:"object"`
	Text   string         `json:"text" example:"Amount 15000 falls in tier [5000, 50000): assessor."`
}

// ProcessClaimRequest is the body of POST /claims/process.
type ProcessClaimRequest struct {
	TenantID string `json:"tenant_id" example:"60244e2b-abde-4cf1-ab77-c7bc8d4cf6d9"` // id or slug
	Claim    Claim  `json:"claim"`
}

// ProcessBatchRequest is the body of POST /claims/process-batch: one claim
// against several tenants (empty tenant_ids = every active tenant).
type ProcessBatchRequest struct {
	TenantIDs []string `json:"tenant_ids" example:"safeguard,healthfirst,govhealth"`
	Claim     Claim    `json:"claim"`
}

// PreviewClaimRequest is the body of POST /claims/preview: either a saved
// tenant (tenant_id) or an unsaved config, plus the claim. Nothing is stored.
type PreviewClaimRequest struct {
	TenantID string        `json:"tenant_id,omitempty" example:"healthfirst"`
	Config   *TenantConfig `json:"config,omitempty"`
	Claim    Claim         `json:"claim"`
}

// BatchResult is data[0] of POST /claims/process-batch.
type BatchResult struct {
	Results []ProcessResult `json:"results"`
}

// ClaimRecord is a row of the claim table (GET /claims).
type ClaimRecord struct {
	ID            string         `db:"id" json:"id"`
	TenantID      string         `db:"tenant_id" json:"tenant_id"`
	TenantSlug    string         `db:"tenant_slug" json:"tenant_slug" example:"safeguard"`
	ConfigVersion int            `db:"config_version" json:"config_version" example:"1"`
	ClaimType     string         `db:"claim_type" json:"claim_type" example:"OUTPATIENT"`
	Amount        float64        `db:"amount" json:"amount" example:"15000"`
	SubmittedAt   string         `db:"submitted_at" json:"submitted_at"`
	Accepted      bool           `db:"accepted" json:"accepted"`
	Routing       string         `db:"routing" json:"routing" example:"auto_approved"`
	ApproverRole  *string        `db:"approver_role" json:"approver_role"`
	SLADeadline   *string        `db:"sla_deadline" json:"sla_deadline"`
	CreatedBy     *int64         `db:"created_by" json:"created_by"`
	CreatedAt     string         `db:"created_at" json:"created_at"`
	ResultJSON    string         `db:"result" json:"-"`
	Result        *ProcessResult `db:"-" json:"result,omitempty"`
}

// ClaimList is data[0] of GET /claims.
type ClaimList struct {
	Claims []ClaimRecord `json:"claims"`
}

// HealthStatus is data[0] of GET /health.
type HealthStatus struct {
	Status  string `json:"status" example:"ok"`
	DB      string `json:"db" example:"ok"`
	Version string `json:"version" example:"1.0.0"`
}
