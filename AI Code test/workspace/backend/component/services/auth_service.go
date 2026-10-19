package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"claimconfig/component/models"
	"claimconfig/component/repositories"
	"claimconfig/i18n"
	"claimconfig/utils"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthService handles password login and server-side sessions.
// Google OIDC is out of scope for this challenge (oauth_login_attempt is ready for it).
type AuthService interface {
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
	// Authenticate resolves a session cookie token to the signed-in user.
	Authenticate(ctx context.Context, token string) (*models.SessionUser, error)
	Logout(ctx context.Context, token string) error
	LogoutAll(ctx context.Context, userID int64) (int64, error)
	SetLocale(ctx context.Context, userID int64, locale string) error
	ChangePassword(ctx context.Context, userID int64, current, next, currentToken string) error
	CreateUser(ctx context.Context, in CreateUserInput) (int64, error)
	// WarnDefaultPassword logs a [SECURITY] line while the seeded admin keeps its default password.
	WarnDefaultPassword(ctx context.Context)
}

type LoginInput struct {
	Email      string
	Password   string
	RememberMe bool
	UserAgent  string
	IP         string
}

type LoginResult struct {
	User       *models.SessionUser
	Token      string // goes only into the HttpOnly cookie
	ExpiresAt  time.Time
	RememberMe bool
}

type CreateUserInput struct {
	Email    string
	Name     string
	RoleCode string
	Password string // empty = Google-only account
	Locale   string
}

type userStore interface {
	GetByEmail(ctx context.Context, email string) (*models.AppUser, error)
	GetByID(ctx context.Context, id int64) (*models.AppUser, error)
	RegisterFailedLogin(ctx context.Context, id int64, max int, lockUntil, now string) error
	RecordLogin(ctx context.Context, id int64, now string) error
	SetPassword(ctx context.Context, id int64, hash, now string) error
	SetLocale(ctx context.Context, id int64, locale, now string) error
	Create(ctx context.Context, email, name string, passwordHash *string, roleID int64, locale, now string) (int64, error)
}

type sessionStore interface {
	Create(ctx context.Context, s repositories.NewSession) error
	GetValid(ctx context.Context, tokenHash, now string) (*models.SessionRecord, error)
	Revoke(ctx context.Context, tokenHash, now string) error
	RevokeAllForUser(ctx context.Context, userID int64, exceptHash, now string) (int64, error)
}

type roleStore interface {
	IDByCode(ctx context.Context, code string) (int64, error)
	PermissionCodes(ctx context.Context, roleID int64) ([]string, error)
}

type AuthServiceImpl struct {
	users    userStore
	sessions sessionStore
	roles    roleStore
	cfg      utils.AuthConfig
	now      func() time.Time
}

func NewAuthService(users userStore, sessions sessionStore, roles roleStore, cfg utils.AuthConfig) *AuthServiceImpl {
	return &AuthServiceImpl{users: users, sessions: sessions, roles: roles, cfg: cfg, now: time.Now}
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// compareDummy spends the same time as a real bcrypt check, so a login for an
// unknown or disabled account can't be told apart by timing.
func compareDummy(password string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

func (s *AuthServiceImpl) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(in.Email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil || u.Status != "active" || u.PasswordHash == nil {
		compareDummy(in.Password)
		return nil, ErrInvalidCredentials
	}
	if u.LockedUntil != nil && *u.LockedUntil > utils.FormatTime(now) {
		return nil, ErrAccountLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.Password)) != nil {
		lockUntil := utils.FormatTime(now.Add(time.Duration(s.cfg.LockoutMinutes) * time.Minute))
		if err := s.users.RegisterFailedLogin(ctx, u.ID, s.cfg.MaxFailedLogins, lockUntil, utils.FormatTime(now)); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}
	if err := s.users.RecordLogin(ctx, u.ID, utils.FormatTime(now)); err != nil {
		return nil, err
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}
	ttl := time.Duration(s.cfg.SessionTTLHours) * time.Hour
	if in.RememberMe {
		ttl = time.Duration(s.cfg.RememberMeDays) * 24 * time.Hour
	}
	expires := now.Add(ttl)
	err = s.sessions.Create(ctx, repositories.NewSession{
		UserID: u.ID, TokenHash: hashToken(token), AuthMethod: "password", RememberMe: in.RememberMe,
		UserAgent: truncate(in.UserAgent, 255), IP: in.IP,
		CreatedAt: utils.FormatTime(now), ExpiresAt: utils.FormatTime(expires),
	})
	if err != nil {
		return nil, err
	}
	user, err := s.sessionUser(ctx, u, "password")
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: user, Token: token, ExpiresAt: expires, RememberMe: in.RememberMe}, nil
}

func (s *AuthServiceImpl) Authenticate(ctx context.Context, token string) (*models.SessionUser, error) {
	if token == "" {
		return nil, ErrUnauthenticated
	}
	sess, err := s.sessions.GetValid(ctx, hashToken(token), utils.FormatTime(s.now()))
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrUnauthenticated
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUnauthenticated
	}
	return s.sessionUser(ctx, u, sess.AuthMethod)
}

func (s *AuthServiceImpl) sessionUser(ctx context.Context, u *models.AppUser, method string) (*models.SessionUser, error) {
	perms, err := s.roles.PermissionCodes(ctx, u.RoleID)
	if err != nil {
		return nil, err
	}
	return &models.SessionUser{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, AuthMethod: method,
		Role:        models.RoleRef{Code: u.RoleCode, Name: u.RoleName},
		Permissions: perms, PermissionNames: map[string]string{}, Locale: u.Locale,
	}, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Revoke(ctx, hashToken(token), utils.FormatTime(s.now()))
}

func (s *AuthServiceImpl) LogoutAll(ctx context.Context, userID int64) (int64, error) {
	return s.sessions.RevokeAllForUser(ctx, userID, "", utils.FormatTime(s.now()))
}

func (s *AuthServiceImpl) SetLocale(ctx context.Context, userID int64, locale string) error {
	if !i18n.IsSupported(locale) {
		return ErrLocaleUnsupported
	}
	return s.users.SetLocale(ctx, userID, locale, utils.FormatTime(s.now()))
}

// ChangePassword verifies the current password, stores the new one and signs
// out the user's other sessions (the current one stays).
func (s *AuthServiceImpl) ChangePassword(ctx context.Context, userID int64, current, next, currentToken string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUnauthenticated
	}
	if u.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCurrentPassword
	}
	if err := s.checkPassword(next); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}
	now := utils.FormatTime(s.now())
	if err := s.users.SetPassword(ctx, userID, string(hash), now); err != nil {
		return err
	}
	_, err = s.sessions.RevokeAllForUser(ctx, userID, hashToken(currentToken), now)
	return err
}

func (s *AuthServiceImpl) checkPassword(p string) error {
	if len([]rune(p)) < s.cfg.MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(p) > 72 { // bcrypt ignores everything after 72 bytes
		return ErrPasswordTooLong
	}
	return nil
}

func (s *AuthServiceImpl) CreateUser(ctx context.Context, in CreateUserInput) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") || strings.TrimSpace(in.Name) == "" {
		return 0, ErrBadRequest
	}
	if in.Locale == "" {
		in.Locale = i18n.DefaultLocale
	}
	if !i18n.IsSupported(in.Locale) {
		return 0, ErrLocaleUnsupported
	}
	roleID, err := s.roles.IDByCode(ctx, in.RoleCode)
	if err != nil {
		return 0, err
	}
	if roleID == 0 {
		return 0, wrapErr(ErrBadRequest, errors.New("unknown role "+in.RoleCode))
	}
	var hash *string
	if in.Password != "" {
		if err := s.checkPassword(in.Password); err != nil {
			return 0, err
		}
		h, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
		if err != nil {
			return 0, err
		}
		v := string(h)
		hash = &v
	}
	id, err := s.users.Create(ctx, email, strings.TrimSpace(in.Name), hash, roleID, in.Locale, utils.FormatTime(s.now()))
	if errors.Is(err, repositories.ErrDuplicate) {
		return 0, wrapErr(ErrBadRequest, errors.New("email already exists"))
	}
	return id, err
}

func (s *AuthServiceImpl) WarnDefaultPassword(ctx context.Context) {
	u, err := s.users.GetByEmail(ctx, "admin@papaya.com")
	if err != nil || u == nil || u.PasswordHash == nil {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte("123456")) == nil {
		utils.ShowInfoLogs("[SECURITY] admin@papaya.com still uses the default password; change it after the first login")
	}
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
