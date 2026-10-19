package models

// Permission codes (rows of the permission table, seeded by 002_seed_rbac.sql).
// Code checks these, never role names.
const (
	PermUsersRead     = "users.read"
	PermUsersManage   = "users.manage"
	PermTenantsRead   = "tenants.read"
	PermTenantsManage = "tenants.manage"
	PermClaimsRead    = "claims.read"
	PermClaimsProcess = "claims.process"
)

// AppUser is a row of app_user joined with its role and locale.
type AppUser struct {
	ID               int64   `db:"id"`
	Email            string  `db:"email"`
	DisplayName      string  `db:"display_name"`
	PasswordHash     *string `db:"password_hash"`
	RoleID           int64   `db:"role_id"`
	RoleCode         string  `db:"role_code"`
	RoleName         string  `db:"role_name"`
	Status           string  `db:"status"`
	FailedLoginCount int     `db:"failed_login_count"`
	LockedUntil      *string `db:"locked_until"`
	Locale           string  `db:"locale"`
}

// SessionRecord is the part of user_session the auth flow needs.
type SessionRecord struct {
	ID         int64  `db:"id"`
	UserID     int64  `db:"user_id"`
	AuthMethod string `db:"auth_method"`
}

// RoleRef is a role code with its display name in the request's language.
type RoleRef struct {
	Code string `json:"code" example:"operator"`
	Name string `json:"name" example:"Operator"`
}

// SessionUser is the signed-in user, returned by /auth/me and /auth/login.
type SessionUser struct {
	ID              int64             `json:"id" example:"2"`
	Email           string            `json:"email" example:"operator@papaya.com"`
	DisplayName     string            `json:"display_name" example:"Operations Demo"`
	AuthMethod      string            `json:"auth_method" example:"password"`
	Role            RoleRef           `json:"role"`
	Permissions     []string          `json:"permissions" example:"tenants.read,tenants.manage,claims.read,claims.process"`
	PermissionNames map[string]string `json:"permission_names"`
	Locale          string            `json:"locale" example:"vi"`
}

// Can reports whether the user holds a permission code.
func (u *SessionUser) Can(code string) bool {
	for _, p := range u.Permissions {
		if p == code {
			return true
		}
	}
	return false
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email      string `json:"email" example:"admin@papaya.com"`
	Password   string `json:"password" example:"123456"`
	RememberMe bool   `json:"remember_me" example:"false"`
}

// ChangePasswordRequest is the body of POST /auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" example:"123456"`
	NewPassword     string `json:"new_password" example:"a-much-longer-password"`
}

// LocaleRequest is the body of PUT /auth/me/locale.
type LocaleRequest struct {
	Locale string `json:"locale" example:"en" enums:"vi,en"`
}

// LogoutAllResponse is data[0] of POST /auth/logout-all.
type LogoutAllResponse struct {
	RevokedSessions int64 `json:"revoked_sessions" example:"2"`
}
