package services

import (
	"testing"

	"claimconfig/i18n"
)

// Every error code the API can return must be translated in every language,
// otherwise the response's error_text is empty.
func TestEveryErrorCodeIsTranslated(t *testing.T) {
	all := []*AppError{
		ErrBadRequest,
		ErrUnauthenticated, ErrInvalidCredentials, ErrAccountLocked, ErrInvalidCurrentPassword,
		ErrPasswordTooShort, ErrPasswordTooLong, ErrLocaleUnsupported,
		ErrConfigRequired, ErrTenantSlugInvalid, ErrTenantSlugTaken, ErrTenantNotFound, ErrTenantArchived,
		ErrTenantRequired, ErrVersionRequired, ErrVersionNotFound, ErrVersionConflict,
		ErrClaimTypeRequired, ErrClaimTypeUnknown, ErrClaimAmountInvalid,
	}
	extra := []string{"CONFIG_INVALID", "FORBIDDEN", "CROSS_ORIGIN", "NOT_FOUND", "INTERNAL_ERROR"}
	codes := extra
	for _, e := range all {
		codes = append(codes, e.Code)
	}
	for _, loc := range i18n.Supported {
		for _, code := range codes {
			if !i18n.Has(loc, "error."+code) {
				t.Errorf("%s: missing error.%s", loc, code)
			}
		}
	}
}
