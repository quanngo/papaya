package middleware

import (
	"strings"

	"claimconfig/i18n"

	"github.com/gin-gonic/gin"
)

// Locale picks the request language from Accept-Language (vi when nothing
// matches). RequireAuth later switches to the signed-in user's saved locale.
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(i18n.ContextKey, fromAcceptLanguage(c.GetHeader("Accept-Language")))
		c.Next()
	}
}

func fromAcceptLanguage(header string) string {
	for _, part := range strings.Split(header, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if base := strings.SplitN(tag, "-", 2)[0]; i18n.IsSupported(base) {
			return base
		}
	}
	return i18n.DefaultLocale
}
