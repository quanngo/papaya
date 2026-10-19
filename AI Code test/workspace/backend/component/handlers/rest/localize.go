package rest

import (
	"strconv"
	"strings"
	"time"

	"claimconfig/component/models"
	"claimconfig/i18n"

	"github.com/gin-gonic/gin"
)

// Codes never change with the language; only the Text fields are filled here.
// Params keep their raw values in the response (for programs); the text uses
// readable versions: claim types, events and channels by name, numbers with
// digit grouping, dates in the reader's format.

func localizeIssues(c *gin.Context, issues []models.Issue) []models.Issue {
	loc := i18n.Locale(c)
	out := make([]models.Issue, len(issues))
	for i, is := range issues {
		is.Text = issueText(loc, is.Code, is.Params)
		out[i] = is
	}
	return out
}

// issueText words one issue. A format the reader cannot be shown as an
// example (an advanced regex) gets the generic "wrong format" sentence.
func issueText(loc, code string, params map[string]any) string {
	readable := readableParams(loc, params)
	if code == "FIELD_PATTERN_MISMATCH" {
		if s, _ := readable["pattern"].(string); s == "" {
			code = "FIELD_PATTERN_MISMATCH_CUSTOM"
		}
	}
	return i18n.T(loc, "issue."+code, readable)
}

func localizeResult(c *gin.Context, r *models.ProcessResult) {
	loc := i18n.Locale(c)
	r.Errors = localizeIssues(c, r.Errors)
	for i := range r.CustomFields {
		f := &r.CustomFields[i]
		if f.Code != "" {
			// reuse the matching error's params (they carry the field's pattern)
			params := map[string]any{"field": f.Label}
			for _, e := range r.Errors {
				if e.Path == "claim.custom_fields."+f.Key && e.Code == f.Code {
					params = e.Params
				}
			}
			f.Text = issueText(loc, f.Code, params)
		}
	}
	for i := range r.Trace {
		r.Trace[i].Text = i18n.T(loc, "trace."+r.Trace[i].Code, readableParams(loc, r.Trace[i].Params))
	}
}

// readableParams returns a copy of params with values worded for loc.
func readableParams(loc string, params map[string]any) map[string]any {
	if len(params) == 0 {
		return params
	}
	out := make(map[string]any, len(params))
	for k, v := range params {
		switch k {
		case "claim_type":
			out[k] = nameOf(loc, "claim_type", v)
		case "enabled":
			out[k] = namesOf(loc, "claim_type", v)
		case "event":
			out[k] = nameOf(loc, "event", v)
		case "channels", "channel":
			out[k] = namesOf(loc, "channel", v)
		case "deadline":
			out[k] = readableDate(loc, v)
		case "pattern":
			s, _ := v.(string)
			out[k] = patternExample(s) // "" when it cannot be shown as an example
		case "role":
			out[k] = nameOf(loc, "role_name", v) // known approver roles by name; custom roles as typed
		default:
			out[k] = readableNumber(loc, v)
		}
	}
	return out
}

// nameOf translates one code ("OUTPATIENT") via "<prefix>.<code>", keeping the code if unknown.
func nameOf(loc, prefix string, v any) any {
	s, ok := v.(string)
	if !ok || s == "" {
		return v
	}
	if name := i18n.T(loc, prefix+"."+s, nil); name != "" {
		return name
	}
	return s
}

// namesOf translates a comma-separated list of codes.
func namesOf(loc, prefix string, v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = nameOf(loc, prefix, strings.TrimSpace(p)).(string)
	}
	return strings.Join(parts, ", ")
}

func readableDate(loc string, v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return v
	}
	if loc == "en" {
		return d.Format("Jan 2, 2006")
	}
	return d.Format("02/01/2006")
}

// readableNumber groups digits: 15000 -> "15.000" (vi) / "15,000" (en).
func readableNumber(loc string, v any) any {
	var f float64
	switch n := v.(type) {
	case float64:
		f = n
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	default:
		return v
	}
	group, decimal := ".", ","
	if loc == "en" {
		group, decimal = ",", "."
	}
	raw := strconv.FormatFloat(f, 'f', -1, 64)
	sign := ""
	if strings.HasPrefix(raw, "-") {
		sign, raw = "-", raw[1:]
	}
	intPart, frac, hasFrac := strings.Cut(raw, ".")
	if len(intPart) < 4 { // 0–999 need no grouping
		if hasFrac {
			return sign + intPart + decimal + frac
		}
		return sign + intPart
	}
	var b strings.Builder
	for i, d := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteString(group)
		}
		b.WriteRune(d)
	}
	if hasFrac {
		b.WriteString(decimal + frac)
	}
	return sign + b.String()
}

// localizeUser returns a copy with role and permission names in the request language.
func localizeUser(c *gin.Context, u *models.SessionUser) models.SessionUser {
	loc := i18n.Locale(c)
	out := *u
	if name := i18n.T(loc, "role."+u.Role.Code, nil); name != "" {
		out.Role.Name = name
	}
	out.PermissionNames = make(map[string]string, len(u.Permissions))
	for _, p := range u.Permissions {
		out.PermissionNames[p] = i18n.T(loc, "permission."+p, nil)
	}
	return out
}

// patternExample turns the simple regexes the admin UI builds from a format
// mask ("^[A-Z]{3}-[0-9]{4}$") into a valid example ("ABC-1234"), so messages
// never show regex. It returns "" for anything outside that subset.
func patternExample(re string) string {
	if !strings.HasPrefix(re, "^") || !strings.HasSuffix(re, "$") || strings.HasSuffix(re, `\$`) {
		return ""
	}
	s := re[1 : len(re)-1]
	classes := []struct{ cls, kind string }{
		{"[A-Za-z0-9]", "*"}, {"[a-zA-Z0-9]", "*"}, {"[A-Z]", "A"}, {"[a-z]", "a"}, {"[0-9]", "9"}, {`\d`, "9"},
	}
	const letters, digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "1234567890"
	var b strings.Builder
	l, d, mixed := 0, 0, 0
	emit := func(kind, literal string) {
		switch kind {
		case "A":
			b.WriteByte(letters[l%26])
			l++
		case "a":
			b.WriteString(strings.ToLower(string(letters[l%26])))
			l++
		case "9":
			b.WriteByte(digits[d%10])
			d++
		case "*":
			if mixed%2 == 0 {
				b.WriteByte(letters[l%26])
				l++
			} else {
				b.WriteByte(digits[d%10])
				d++
			}
			mixed++
		default:
			b.WriteString(literal)
		}
	}
	for i := 0; i < len(s); {
		kind, literal := "", ""
		matched := false
		for _, c := range classes {
			if strings.HasPrefix(s[i:], c.cls) {
				kind, matched = c.kind, true
				i += len(c.cls)
				break
			}
		}
		if !matched {
			switch {
			case s[i] == '\\' && i+1 < len(s):
				literal = string(s[i+1])
				i += 2
			case strings.ContainsRune(`.^$*+?()[]{}|`, rune(s[i])):
				return ""
			default:
				r := []rune(s[i:])[0]
				literal = string(r)
				i += len(string(r))
			}
		}
		n := 1
		if strings.HasPrefix(s[i:], "{") {
			end := strings.IndexByte(s[i:], '}')
			if end < 0 {
				return ""
			}
			v, err := strconv.Atoi(s[i+1 : i+end])
			if err != nil || v < 1 || v > 99 {
				return ""
			}
			n = v
			i += end + 1
		}
		for j := 0; j < n; j++ {
			emit(kind, literal)
		}
	}
	return b.String()
}
