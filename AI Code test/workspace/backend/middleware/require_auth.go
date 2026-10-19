package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/i18n"
	"claimconfig/utils"

	"github.com/gin-gonic/gin"
)

const (
	userKey  = "session_user"
	tokenKey = "session_token"
)

// Authenticator resolves a session token. It is an interface so middleware
// does not import services (services may import middleware, not the reverse).
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*models.SessionUser, error)
}

type statusError interface {
	HTTPStatus() int
	ErrorCode() string
}

// RequireAuth rejects requests without a valid session cookie and stores the
// user (and switches the request locale to the user's saved one).
func RequireAuth(auth Authenticator, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(cookieName)
		user, err := auth.Authenticate(c.Request.Context(), token)
		if err != nil || user == nil {
			var se statusError
			if err != nil && !errors.As(err, &se) {
				utils.ShowErrorLogs("authenticate", err)
				api.ErrorWithMessage(c, http.StatusInternalServerError, "INTERNAL_ERROR")
			} else {
				api.ErrorWithMessage(c, http.StatusUnauthorized, "UNAUTHENTICATED")
			}
			c.Abort()
			return
		}
		c.Set(userKey, user)
		c.Set(tokenKey, token)
		if i18n.IsSupported(user.Locale) {
			c.Set(i18n.ContextKey, user.Locale)
		}
		c.Next()
	}
}

// CurrentUser returns the user set by RequireAuth (nil on public routes).
func CurrentUser(c *gin.Context) *models.SessionUser {
	if v, ok := c.Get(userKey); ok {
		if u, ok := v.(*models.SessionUser); ok {
			return u
		}
	}
	return nil
}

// CurrentUserID is CurrentUser's id as a pointer, for created_by columns.
func CurrentUserID(c *gin.Context) *int64 {
	if u := CurrentUser(c); u != nil {
		id := u.ID
		return &id
	}
	return nil
}

// SessionToken returns the cookie token RequireAuth accepted.
func SessionToken(c *gin.Context) string {
	return c.GetString(tokenKey)
}

// RequirePermission rejects users without the permission code. Use it after RequireAuth.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if u := CurrentUser(c); u == nil || !u.Can(code) {
			api.ErrorWithMessage(c, http.StatusForbidden, "FORBIDDEN")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSameOrigin blocks state-changing requests whose Origin (or Referer)
// is another host: a CSRF guard on top of the SameSite=Lax cookie. Requests
// with neither header (curl, server-to-server) carry no browser cookie
// ambiently and are allowed.
func RequireSameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}
		if origin != "" && !sameHost(origin, requestHost(c)) {
			api.ErrorWithMessage(c, http.StatusForbidden, "CROSS_ORIGIN")
			c.Abort()
			return
		}
		c.Next()
	}
}

func requestHost(c *gin.Context) string {
	if h := c.GetHeader("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	return c.Request.Host
}

func sameHost(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, host)
}
