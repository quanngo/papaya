// Package i18n translates user-facing text (error_text, issue/trace text,
// role and permission names). Error codes themselves never change with the
// language: code and tests keep switching on the code.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// Supported lists the UI languages; the first one is the default.
var Supported = []string{"vi", "en"}

// DefaultLocale is used when nothing else is known about the user.
const DefaultLocale = "vi"

// ContextKey is where middleware stores the request's locale.
const ContextKey = "locale"

//go:embed locales/*.json
var localeFiles embed.FS

var catalogs = map[string]map[string]string{}

func init() {
	for _, loc := range Supported {
		raw, err := localeFiles.ReadFile("locales/" + loc + ".json")
		if err != nil {
			panic(fmt.Sprintf("i18n: missing locales/%s.json", loc))
		}
		var nested map[string]any
		if err := json.Unmarshal(raw, &nested); err != nil {
			panic(fmt.Sprintf("i18n: locales/%s.json: %v", loc, err))
		}
		flat := map[string]string{}
		flatten("", nested, flat)
		catalogs[loc] = flat
	}
}

func flatten(prefix string, in map[string]any, out map[string]string) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flatten(key, t, out)
		case string:
			out[key] = t
		}
	}
}

// IsSupported reports whether loc is one of Supported.
func IsSupported(loc string) bool {
	for _, s := range Supported {
		if s == loc {
			return true
		}
	}
	return false
}

// Has reports whether key exists in the given locale.
func Has(loc, key string) bool {
	_, ok := catalogs[loc][key]
	return ok
}

// Keys returns every flattened key of a locale (used by the parity test).
func Keys(loc string) []string {
	out := make([]string, 0, len(catalogs[loc]))
	for k := range catalogs[loc] {
		out = append(out, k)
	}
	return out
}

// T returns the text for key in loc, replacing {{name}} with params.
// It falls back to the default locale, then to "" when the key is unknown.
func T(loc, key string, params map[string]any) string {
	msg, ok := catalogs[loc][key]
	if !ok {
		msg, ok = catalogs[DefaultLocale][key]
	}
	if !ok {
		return ""
	}
	for k, v := range params {
		msg = strings.ReplaceAll(msg, "{{"+k+"}}", fmt.Sprint(v))
	}
	return msg
}

// Locale returns the request's locale set by middleware (default vi).
func Locale(c *gin.Context) string {
	if v := c.GetString(ContextKey); IsSupported(v) {
		return v
	}
	return DefaultLocale
}
