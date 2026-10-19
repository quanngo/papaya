package services

// Tenant and claim error codes. Each needs error.<CODE> in i18n/locales/{vi,en}.json.
var (
	ErrConfigRequired     = &AppError{Code: "CONFIG_REQUIRED", Status: 400}
	ErrTenantSlugInvalid  = &AppError{Code: "TENANT_SLUG_INVALID", Status: 400}
	ErrTenantSlugTaken    = &AppError{Code: "TENANT_SLUG_TAKEN", Status: 409}
	ErrTenantNotFound     = &AppError{Code: "TENANT_NOT_FOUND", Status: 404}
	ErrTenantArchived     = &AppError{Code: "TENANT_ARCHIVED", Status: 409}
	ErrTenantRequired     = &AppError{Code: "TENANT_REQUIRED", Status: 400}
	ErrVersionRequired    = &AppError{Code: "VERSION_REQUIRED", Status: 400}
	ErrVersionNotFound    = &AppError{Code: "VERSION_NOT_FOUND", Status: 404}
	ErrVersionConflict    = &AppError{Code: "VERSION_CONFLICT", Status: 409}
	ErrClaimTypeRequired  = &AppError{Code: "CLAIM_TYPE_REQUIRED", Status: 400}
	ErrClaimTypeUnknown   = &AppError{Code: "CLAIM_TYPE_UNKNOWN", Status: 400}
	ErrClaimAmountInvalid = &AppError{Code: "CLAIM_AMOUNT_INVALID", Status: 400}
)
