package services

// Auth error codes. Each needs error.<CODE> in i18n/locales/{vi,en}.json.
var (
	ErrUnauthenticated        = &AppError{Code: "UNAUTHENTICATED", Status: 401}
	ErrInvalidCredentials     = &AppError{Code: "INVALID_CREDENTIALS", Status: 401}
	ErrAccountLocked          = &AppError{Code: "ACCOUNT_LOCKED", Status: 423}
	ErrInvalidCurrentPassword = &AppError{Code: "INVALID_CURRENT_PASSWORD", Status: 400}
	ErrPasswordTooShort       = &AppError{Code: "PASSWORD_TOO_SHORT", Status: 400}
	ErrPasswordTooLong        = &AppError{Code: "PASSWORD_TOO_LONG", Status: 400}
	ErrLocaleUnsupported      = &AppError{Code: "LOCALE_UNSUPPORTED", Status: 400}
)
