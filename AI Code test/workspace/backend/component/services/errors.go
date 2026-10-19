package services

import (
	"fmt"

	"claimconfig/component/models"
)

// AppError is a business error with a stable Code (sent in the response
// "message") and a Status (sent in "code"). Compare with errors.Is.
type AppError struct {
	Code   string
	Status int
	cause  error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	}
	return e.Code
}

func (e *AppError) Unwrap() error { return e.cause }

// HTTPStatus lets packages that must not import services (middleware) read the status.
func (e *AppError) HTTPStatus() int { return e.Status }

// ErrorCode lets packages that must not import services read the code.
func (e *AppError) ErrorCode() string { return e.Code }

// Is matches any AppError with the same code, so wrapped errors still compare.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	return ok && t.Code == e.Code
}

// wrapErr attaches a cause worth logging without changing the code.
func wrapErr(e *AppError, cause error) error {
	return &AppError{Code: e.Code, Status: e.Status, cause: cause}
}

var ErrBadRequest = &AppError{Code: "BAD_REQUEST", Status: 400}

// ConfigInvalidError carries every validation issue of a rejected config.
type ConfigInvalidError struct {
	Issues []models.Issue
}

func (e *ConfigInvalidError) Error() string {
	return fmt.Sprintf("CONFIG_INVALID: %d issue(s)", len(e.Issues))
}
