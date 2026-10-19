package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// AppConfig mirrors config/application.yml. Secrets are never written in the
// yml itself: it holds "${VAR}" placeholders expanded from the environment
// (or config/.env).
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	StaticDir string `yaml:"static-dir"` // built frontend to serve on "/"; empty = API only
	Swagger   bool   `yaml:"swagger"`
}

type DatabaseConfig struct {
	URL         string `yaml:"url"`
	AutoMigrate bool   `yaml:"auto-migrate"`
	Seed        bool   `yaml:"seed"`
}

type AuthConfig struct {
	SessionTTLHours   int    `yaml:"session-ttl-hours"`
	RememberMeDays    int    `yaml:"remember-me-days"`
	MinPasswordLength int    `yaml:"min-password-length"`
	MaxFailedLogins   int    `yaml:"max-failed-logins"`
	LockoutMinutes    int    `yaml:"lockout-minutes"`
	CookieName        string `yaml:"cookie-name"`
	CookieSecure      bool   `yaml:"cookie-secure"`
}

const DefaultDatabaseURL = "file:./data/app.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

// DefaultConfig is used for anything the yml leaves out.
func DefaultConfig() AppConfig {
	return AppConfig{
		Server:   ServerConfig{Port: 8080, Swagger: true},
		Database: DatabaseConfig{URL: DefaultDatabaseURL, AutoMigrate: true, Seed: true},
		Auth: AuthConfig{
			SessionTTLHours:   12,
			RememberMeDays:    7,
			MinPasswordLength: 10,
			MaxFailedLogins:   5,
			LockoutMinutes:    15,
			CookieName:        "session",
		},
	}
}

// LoadConfig reads config/.env (optional) and config/application.yml
// (optional), expanding ${VAR} placeholders. DATABASE_URL, STATIC_DIR and PORT
// from the environment always win, so a PaaS can configure the service without a yml.
func LoadConfig(path string) (AppConfig, error) {
	cfg := DefaultConfig()
	if err := godotenv.Load("config/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, fmt.Errorf("config/.env: %w", err)
	}
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &cfg); err != nil {
			return cfg, fmt.Errorf("%s: %w", path, err)
		}
	case !errors.Is(err, os.ErrNotExist):
		return cfg, err
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("STATIC_DIR"); v != "" {
		cfg.Server.StaticDir = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	cfg.WithDefaults()
	return cfg, nil
}

// WithDefaults fills zero values left by a partial yml.
func (c *AppConfig) WithDefaults() {
	d := DefaultConfig()
	if c.Server.Port == 0 {
		c.Server.Port = d.Server.Port
	}
	if c.Database.URL == "" {
		c.Database.URL = d.Database.URL
	}
	a := &c.Auth
	if a.SessionTTLHours <= 0 {
		a.SessionTTLHours = d.Auth.SessionTTLHours
	}
	if a.RememberMeDays <= 0 {
		a.RememberMeDays = d.Auth.RememberMeDays
	}
	if a.MinPasswordLength <= 0 {
		a.MinPasswordLength = d.Auth.MinPasswordLength
	}
	if a.MaxFailedLogins <= 0 {
		a.MaxFailedLogins = d.Auth.MaxFailedLogins
	}
	if a.LockoutMinutes <= 0 {
		a.LockoutMinutes = d.Auth.LockoutMinutes
	}
	if a.CookieName == "" {
		a.CookieName = d.Auth.CookieName
	}
}
