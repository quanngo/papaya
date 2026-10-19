// Package server wires configuration, database, repositories, services and
// HTTP routes, and implements the CLI subcommands.
package server

import (
	"context"
	"os"
	"time"

	"claimconfig/component/handlers/rest"
	"claimconfig/component/repositories"
	"claimconfig/component/services"
	"claimconfig/middleware"
	"claimconfig/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Version is reported by /health.
const Version = "1.0.0"

var nowFunc = time.Now

// App holds the wired dependencies.
type App struct {
	Config  utils.AppConfig
	DB      *sqlx.DB
	Auth    *services.AuthServiceImpl
	Tenants *services.TenantServiceImpl
	Claims  *services.ClaimServiceImpl
	Engine  *gin.Engine
}

// Initialize opens the database, applies migrations and seeds (when enabled
// in config), builds every layer and returns the HTTP engine.
func Initialize(ctx context.Context, cfg utils.AppConfig) (*App, error) {
	db, err := repositories.Open(cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	if cfg.Database.AutoMigrate {
		if err := Migrate(ctx, db, false, os.Stdout); err != nil {
			db.Close()
			return nil, err
		}
	}

	tenantRepo := repositories.NewTenantRepo(db)
	app := &App{
		Config: cfg,
		DB:     db,
		Auth: services.NewAuthService(
			repositories.NewUserRepo(db), repositories.NewSessionRepo(db), repositories.NewRoleRepo(db), cfg.Auth),
		Tenants: services.NewTenantService(tenantRepo),
	}
	app.Claims = services.NewClaimService(tenantRepo, repositories.NewClaimRepo(db))

	if cfg.Database.Seed {
		if err := SeedTenants(ctx, tenantRepo, app.Tenants); err != nil {
			db.Close()
			return nil, err
		}
	}
	app.Auth.WarnDefaultPassword(ctx)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestLogger())
	_ = r.SetTrustedProxies(nil)
	rest.APIRoutes(r, rest.Deps{
		Config: cfg, DB: db, Auth: app.Auth, Tenants: app.Tenants, Claims: app.Claims, Version: Version,
	})
	serveFrontend(r, cfg.Server.StaticDir)
	app.Engine = r
	return app, nil
}

// Close releases the database.
func (a *App) Close() error { return a.DB.Close() }
