package middleware

import (
	"time"

	"claimconfig/utils"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs method, path, status and latency. The query string is
// dropped on purpose: it can carry secrets (e.g. an OAuth code).
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		utils.ShowInfoLogs("%s %s -> %d (%s)", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start).Round(time.Millisecond))
	}
}
