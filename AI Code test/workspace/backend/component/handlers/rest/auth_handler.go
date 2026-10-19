package rest

import (
	"net/http"
	"time"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/i18n"
	"claimconfig/middleware"
	"claimconfig/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth services.AuthService
	cfg  utils.AuthConfig
}

func NewAuthHandler(auth services.AuthService, cfg utils.AuthConfig) *AuthHandler {
	return &AuthHandler{auth: auth, cfg: cfg}
}

func (h *AuthHandler) setCookie(c *gin.Context, value string, expires time.Time, persistent bool) {
	cookie := &http.Cookie{
		Name: h.cfg.CookieName, Value: value, Path: "/",
		HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	}
	switch {
	case value == "":
		cookie.MaxAge = -1
	case persistent:
		cookie.Expires = expires
		cookie.MaxAge = int(time.Until(expires).Seconds())
	}
	http.SetCookie(c.Writer, cookie)
}

// Login godoc
//
//	@Summary		Sign in with email and password
//	@Description	Sets an HttpOnly session cookie (persistent for remember-me days when remember_me is true, otherwise until the browser closes).
//	@Description	In Swagger UI, call this first: the cookie is then sent with every other request.
//	@Description	Errors (in `message`): BAD_REQUEST, INVALID_CREDENTIALS (also for unknown/disabled accounts), ACCOUNT_LOCKED, CROSS_ORIGIN.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.LoginRequest	true	"Credentials (seeded: admin@papaya.com / 123456)"
//	@Success		200		{object}	api.Response{data=[]models.SessionUser}
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.auth.Login(c.Request.Context(), services.LoginInput{
		Email: req.Email, Password: req.Password, RememberMe: req.RememberMe,
		UserAgent: c.Request.UserAgent(), IP: c.ClientIP(),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	h.setCookie(c, res.Token, res.ExpiresAt, res.RememberMe)
	if i18n.IsSupported(res.User.Locale) {
		c.Set(i18n.ContextKey, res.User.Locale)
	}
	api.Ok(c, localizeUser(c, res.User))
}

// Logout godoc
//
//	@Summary		Sign out this browser
//	@Description	Revokes the current session and clears the cookie. Always succeeds.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	api.Response
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.CookieName)
	if err := h.auth.Logout(c.Request.Context(), token); err != nil {
		utils.ShowErrorLogs("logout", err)
	}
	h.setCookie(c, "", time.Time{}, false)
	api.Ok(c)
}

// LogoutAll godoc
//
//	@Summary		Sign out every session of the current user
//	@Description	Requires login. Errors: UNAUTHENTICATED, CROSS_ORIGIN.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	api.Response{data=[]models.LogoutAllResponse}
//	@Router			/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	n, err := h.auth.LogoutAll(c.Request.Context(), middleware.CurrentUser(c).ID)
	if err != nil {
		respondError(c, err)
		return
	}
	h.setCookie(c, "", time.Time{}, false)
	api.Ok(c, models.LogoutAllResponse{RevokedSessions: n})
}

// Me godoc
//
//	@Summary		Current user
//	@Description	The signed-in user with role, permission codes, their names in the user's language, and locale.
//	@Description	Errors: UNAUTHENTICATED.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	api.Response{data=[]models.SessionUser}
//	@Router			/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	api.Ok(c, localizeUser(c, middleware.CurrentUser(c)))
}

// SetLocale godoc
//
//	@Summary		Save the UI language
//	@Description	Errors: BAD_REQUEST, LOCALE_UNSUPPORTED, UNAUTHENTICATED, CROSS_ORIGIN.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.LocaleRequest	true	"vi or en"
//	@Success		200		{object}	api.Response{data=[]models.SessionUser}
//	@Router			/auth/me/locale [put]
func (h *AuthHandler) SetLocale(c *gin.Context) {
	var req models.LocaleRequest
	if !bindJSON(c, &req) {
		return
	}
	user := middleware.CurrentUser(c)
	if err := h.auth.SetLocale(c.Request.Context(), user.ID, req.Locale); err != nil {
		respondError(c, err)
		return
	}
	updated := *user
	updated.Locale = req.Locale
	c.Set(i18n.ContextKey, req.Locale)
	api.Ok(c, localizeUser(c, &updated))
}

// ChangePassword godoc
//
//	@Summary		Change own password
//	@Description	Signs out the user's other sessions; this one stays.
//	@Description	Errors: BAD_REQUEST, INVALID_CURRENT_PASSWORD, PASSWORD_TOO_SHORT, PASSWORD_TOO_LONG, UNAUTHENTICATED, CROSS_ORIGIN.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.ChangePasswordRequest	true	"Current and new password"
//	@Success		200		{object}	api.Response
//	@Router			/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	err := h.auth.ChangePassword(c.Request.Context(), middleware.CurrentUser(c).ID,
		req.CurrentPassword, req.NewPassword, middleware.SessionToken(c))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c)
}
