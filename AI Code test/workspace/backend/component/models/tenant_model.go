package models

// Tenant is a row of the tenant table.
type Tenant struct {
	ID             string  `db:"id" json:"id" example:"60244e2b-abde-4cf1-ab77-c7bc8d4cf6d9"`
	Slug           string  `db:"slug" json:"slug" example:"safeguard"`
	Name           string  `db:"name" json:"name" example:"SafeGuard Insurance"`
	CurrentVersion int     `db:"current_version" json:"current_version" example:"1"`
	ArchivedAt     *string `db:"archived_at" json:"archived_at" example:""`
	CreatedBy      *int64  `db:"created_by" json:"created_by"`
	CreatedAt      string  `db:"created_at" json:"created_at" example:"2026-10-17T08:00:00.000Z"`
	UpdatedAt      string  `db:"updated_at" json:"updated_at" example:"2026-10-17T08:00:00.000Z"`
}

// TenantDetail is a tenant with its current config.
type TenantDetail struct {
	Tenant
	Config TenantConfig `json:"config"`
}

// TenantVersion is one entry of a tenant's config history.
type TenantVersion struct {
	TenantID       string        `db:"tenant_id" json:"tenant_id"`
	Version        int           `db:"version" json:"version" example:"2"`
	Note           string        `db:"note" json:"note" example:"Raise auto-approval threshold"`
	Source         string        `db:"source" json:"source" example:"update" enums:"create,update,rollback"`
	RolledBackFrom *int          `db:"rolled_back_from" json:"rolled_back_from"`
	CreatedBy      *int64        `db:"created_by" json:"created_by"`
	CreatedByEmail *string       `db:"created_by_email" json:"created_by_email" example:"operator@papaya.com"`
	CreatedAt      string        `db:"created_at" json:"created_at" example:"2026-10-17T08:00:00.000Z"`
	IsCurrent      bool          `db:"-" json:"is_current"`
	Config         *TenantConfig `db:"-" json:"config,omitempty"`
}

// CreateTenantRequest is the body of POST /tenants.
type CreateTenantRequest struct {
	Slug   string        `json:"slug" example:"carelink"`
	Config *TenantConfig `json:"config"`
	Note   string        `json:"note" example:"Onboard CareLink"`
}

// UpdateTenantRequest is the body of PUT /tenants/{id}. BaseVersion is the
// version the edit started from; a stale one fails with VERSION_CONFLICT.
type UpdateTenantRequest struct {
	Config      *TenantConfig `json:"config"`
	Note        string        `json:"note" example:"Raise auto-approval threshold"`
	BaseVersion int           `json:"base_version" example:"1"`
}

// RollbackRequest is the body of POST /tenants/{id}/rollback.
type RollbackRequest struct {
	Version int    `json:"version" example:"1"`
	Note    string `json:"note" example:"Revert threshold change"`
}

// ValidateConfigRequest is the body of POST /configs/validate.
type ValidateConfigRequest struct {
	Config *TenantConfig `json:"config"`
}

// ValidationResult is data[0] of POST /configs/validate and of CONFIG_INVALID errors.
type ValidationResult struct {
	Valid  bool    `json:"valid" example:"false"`
	Issues []Issue `json:"issues"`
}

// TenantList is data[0] of GET /tenants.
type TenantList struct {
	Tenants []Tenant `json:"tenants"`
}

// VersionList is data[0] of GET /tenants/{id}/versions.
type VersionList struct {
	Versions []TenantVersion `json:"versions"`
}

// DiffSide identifies one side of a comparison.
type DiffSide struct {
	TenantID string `json:"tenant_id"`
	Slug     string `json:"slug" example:"safeguard"`
	Name     string `json:"name" example:"SafeGuard Insurance"`
	Version  int    `json:"version" example:"1"`
}

// Change is one difference between two configs.
type Change struct {
	Path   string `json:"path" example:"approval.auto_approval_threshold"`
	Module string `json:"module" example:"approval"`
	Kind   string `json:"kind" example:"changed" enums:"added,removed,changed"`
	Left   any    `json:"left" swaggertype:"object"`
	Right  any    `json:"right" swaggertype:"object"`
}

// DiffResult is data[0] of GET /tenants/diff.
type DiffResult struct {
	A       DiffSide `json:"a"`
	B       DiffSide `json:"b"`
	Changes []Change `json:"changes"`
}

// TenantConfigMeta is data[0] of GET /meta/tenant-config.
type TenantConfigMeta struct {
	ClaimTypes         []string `json:"claim_types" example:"OUTPATIENT,INPATIENT,DENTAL,MATERNITY,OPTICAL"`
	NotificationEvents []string `json:"notification_events" example:"claim_submitted,approved,rejected,payment_sent"`
	Channels           []string `json:"channels" example:"email,sms,webhook"`
	CustomFieldTypes   []string `json:"custom_field_types" example:"text,number,select"`
	Modules            []string `json:"modules" example:"branding,claim_types,custom_fields,approval,sla,notifications"`
}
