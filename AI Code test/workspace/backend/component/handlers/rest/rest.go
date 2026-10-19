package rest

import (
	"net/http"
	"strings"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	_ "claimconfig/docs" // registers the generated Swagger spec
	"claimconfig/middleware"
	"claimconfig/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Deps is everything the HTTP layer needs (built in server/base.go).
type Deps struct {
	Config  utils.AppConfig
	DB      *sqlx.DB
	Auth    services.AuthService
	Tenants services.TenantService
	Claims  services.ClaimService
	Version string
}

// APIRoutes mounts /api/v1 and, when enabled, /swagger.
func APIRoutes(r *gin.Engine, d Deps) {
	if d.Config.Server.Swagger {
		r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	health := NewHealthHandler(d.DB, d.Version)
	auth := NewAuthHandler(d.Auth, d.Config.Auth)
	configs := NewConfigHandler(d.Tenants)
	tenants := NewTenantHandler(d.Tenants)
	claims := NewClaimHandler(d.Claims)

	requireAuth := middleware.RequireAuth(d.Auth, d.Config.Auth.CookieName)
	sameOrigin := middleware.RequireSameOrigin()
	perm := middleware.RequirePermission

	v1 := r.Group("/api/v1", middleware.Locale())
	v1.GET("/health", health.Get)

	a := v1.Group("/auth")
	a.POST("/login", sameOrigin, auth.Login)
	a.POST("/logout", sameOrigin, auth.Logout)
	signedIn := a.Group("", requireAuth)
	signedIn.GET("/me", auth.Me)
	signedIn.POST("/logout-all", sameOrigin, auth.LogoutAll)
	signedIn.PUT("/me/locale", sameOrigin, auth.SetLocale)
	signedIn.POST("/change-password", sameOrigin, auth.ChangePassword)

	p := v1.Group("", requireAuth)
	p.GET("/meta/tenant-config", perm(models.PermTenantsRead), configs.Meta)
	p.GET("/configs/defaults", perm(models.PermTenantsManage), configs.Defaults)
	p.POST("/configs/validate", sameOrigin, perm(models.PermTenantsManage), configs.Validate)

	p.GET("/tenants", perm(models.PermTenantsRead), tenants.List)
	p.POST("/tenants", sameOrigin, perm(models.PermTenantsManage), tenants.Create)
	p.GET("/tenants/diff", perm(models.PermTenantsRead), tenants.Diff)
	p.GET("/tenants/:id", perm(models.PermTenantsRead), tenants.Get)
	p.PUT("/tenants/:id", sameOrigin, perm(models.PermTenantsManage), tenants.Update)
	p.DELETE("/tenants/:id", sameOrigin, perm(models.PermTenantsManage), tenants.Archive)
	p.GET("/tenants/:id/versions", perm(models.PermTenantsRead), tenants.Versions)
	p.GET("/tenants/:id/versions/:version", perm(models.PermTenantsRead), tenants.Version)
	p.POST("/tenants/:id/rollback", sameOrigin, perm(models.PermTenantsManage), tenants.Rollback)

	p.POST("/claims/preview", sameOrigin, perm(models.PermTenantsRead), claims.Preview)
	p.POST("/claims/process", sameOrigin, perm(models.PermClaimsProcess), claims.Process)
	p.POST("/claims/process-batch", sameOrigin, perm(models.PermClaimsProcess), claims.ProcessBatch)
	p.GET("/claims", perm(models.PermClaimsRead), claims.List)
}

// APINotFound answers unknown /api paths with the envelope.
func APINotFound(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		api.ErrorWithMessage(c, http.StatusNotFound, "NOT_FOUND")
		return true
	}
	return false
}
