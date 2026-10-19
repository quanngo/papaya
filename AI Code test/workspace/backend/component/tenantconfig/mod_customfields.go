package tenantconfig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"claimconfig/component/models"
)

var fieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// CustomFieldsModule owns the extra data a tenant requires on submission
// (e.g. Employee ID, Department) and validates the claim's values.
type CustomFieldsModule struct{}

func (CustomFieldsModule) Key() string { return "custom_fields" }

func (CustomFieldsModule) Validate(cfg *models.TenantConfig) []models.Issue {
	var out []models.Issue
	seen := map[string]bool{}
	for i, f := range cfg.CustomFields {
		path := idx("custom_fields", i)
		if !fieldKey.MatchString(f.Key) {
			out = append(out, issue(path+".key", "FIELD_KEY_INVALID", map[string]any{"key": f.Key}))
		} else if seen[f.Key] {
			out = append(out, issue(path+".key", "FIELD_KEY_DUPLICATE", map[string]any{"key": f.Key}))
		}
		seen[f.Key] = true
		if strings.TrimSpace(f.Label) == "" {
			out = append(out, issue(path+".label", "FIELD_LABEL_EMPTY", nil))
		}
		if !contains(models.CustomFieldTypes, f.Type) {
			out = append(out, issue(path+".type", "FIELD_TYPE_UNKNOWN", map[string]any{"type": f.Type}))
		}
		if f.Type == "select" && len(f.Options) == 0 {
			out = append(out, issue(path+".options", "FIELD_OPTIONS_REQUIRED", nil))
		}
		if f.Pattern != "" {
			if _, err := regexp.Compile(f.Pattern); err != nil {
				out = append(out, issue(path+".pattern", "FIELD_PATTERN_INVALID", nil))
			}
		}
	}
	return out
}

func (CustomFieldsModule) Process(cfg *models.TenantConfig, claim *models.Claim, _ *Ctx, res *models.ProcessResult) {
	for _, f := range cfg.CustomFields {
		value, present := claim.CustomFields[f.Key]
		r := models.CustomFieldResult{Key: f.Key, Label: f.Label, Type: f.Type, Required: f.Required, Value: value, Valid: true}
		if code := checkField(f, value, present); code != "" {
			r.Valid = false
			r.Code = code
			reject(res, "claim.custom_fields."+f.Key, code, map[string]any{"field": f.Label, "pattern": f.Pattern})
		}
		res.CustomFields = append(res.CustomFields, r)
	}
	if len(cfg.CustomFields) > 0 {
		trace(res, "custom_fields", "CUSTOM_FIELDS_CHECKED", map[string]any{"count": len(cfg.CustomFields)})
	}
}

// checkField returns the issue code for an invalid value, or "".
func checkField(f models.CustomField, value any, present bool) string {
	text := ""
	if present && value != nil {
		text = strings.TrimSpace(fmt.Sprint(value))
	}
	if text == "" {
		if f.Required {
			return "FIELD_REQUIRED"
		}
		return ""
	}
	switch f.Type {
	case "number":
		switch value.(type) {
		case float64, int, int64:
		default:
			if _, err := strconv.ParseFloat(text, 64); err != nil {
				return "FIELD_NOT_NUMBER"
			}
		}
	case "select":
		if !contains(f.Options, text) {
			return "FIELD_NOT_IN_OPTIONS"
		}
	}
	if f.Pattern != "" {
		if re, err := regexp.Compile(f.Pattern); err == nil && !re.MatchString(text) {
			return "FIELD_PATTERN_MISMATCH"
		}
	}
	return ""
}

func (CustomFieldsModule) DiffKeys() map[string]string {
	return map[string]string{"custom_fields": "key"}
}
