package tenantconfig

import (
	"net/url"
	"regexp"
	"strings"

	"claimconfig/component/models"
)

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// BrandingModule owns branding: company name, logo, colors. It has no effect
// on claim processing.
type BrandingModule struct{}

func (BrandingModule) Key() string { return "branding" }

func (BrandingModule) Validate(cfg *models.TenantConfig) []models.Issue {
	b := cfg.Branding
	var out []models.Issue
	if strings.TrimSpace(b.CompanyName) == "" {
		out = append(out, issue("branding.company_name", "COMPANY_NAME_REQUIRED", nil))
	}
	if b.LogoURL != "" && !validHTTPURL(b.LogoURL) {
		out = append(out, issue("branding.logo_url", "URL_INVALID", nil))
	}
	if !hexColor.MatchString(b.PrimaryColor) {
		out = append(out, issue("branding.primary_color", "COLOR_INVALID", nil))
	}
	if !hexColor.MatchString(b.SecondaryColor) {
		out = append(out, issue("branding.secondary_color", "COLOR_INVALID", nil))
	}
	return out
}

func (BrandingModule) Process(*models.TenantConfig, *models.Claim, *Ctx, *models.ProcessResult) {}

func (BrandingModule) DiffKeys() map[string]string { return nil }

func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
