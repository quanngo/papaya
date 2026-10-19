// Package tenantconfig is the pure domain of the platform: validating a
// tenant's configuration, processing a claim under it, and diffing two
// configurations. It does no I/O and must not import gin, sqlx or services.
//
// Every config section is a Module. Adding a new dimension (e.g. currency)
// means: a field on models.TenantConfig, one mod_<name>.go implementing
// Module, one line in Modules (registry.go), and its i18n keys.
package tenantconfig

import (
	"fmt"
	"sort"
	"time"

	"claimconfig/component/models"
)

// Module owns one section of models.TenantConfig.
type Module interface {
	// Key is the section's JSON name, e.g. "approval".
	Key() string
	// Validate returns every problem in the section, including rules that
	// depend on other sections (e.g. SLA must cover every enabled claim type).
	Validate(cfg *models.TenantConfig) []models.Issue
	// Process writes this module's part of the result. Modules run in the
	// order of Modules; a module that depends on acceptance checks res.Accepted.
	Process(cfg *models.TenantConfig, claim *models.Claim, ctx *Ctx, res *models.ProcessResult)
	// DiffKeys maps array paths (without indices) to the field identifying an
	// element, so the diff matches elements by identity instead of position.
	DiffKeys() map[string]string
}

// Ctx carries per-claim values shared by modules.
type Ctx struct {
	SubmittedAt time.Time
}

func issue(path, code string, params map[string]any) models.Issue {
	return models.Issue{Path: path, Code: code, Params: params}
}

func trace(res *models.ProcessResult, module, code string, params map[string]any) {
	res.Trace = append(res.Trace, models.TraceStep{Module: module, Code: code, Params: params})
}

func reject(res *models.ProcessResult, path, code string, params map[string]any) {
	res.Accepted = false
	res.Errors = append(res.Errors, issue(path, code, params))
}

func idx(path string, i int) string { return fmt.Sprintf("%s[%d]", path, i) }

// sortedKeys keeps issue order stable across runs (Go map order is random).
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
