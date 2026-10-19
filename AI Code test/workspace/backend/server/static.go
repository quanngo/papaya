package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"claimconfig/component/handlers/rest"

	"github.com/gin-gonic/gin"
)

// serveFrontend serves the built SPA from dir on every non-API path, falling
// back to index.html so client-side routes work. Serving the FE from the same
// origin as the API is what lets the session cookie and sameOrigin work
// without CORS.
func serveFrontend(r *gin.Engine, dir string) {
	index := filepath.Join(dir, "index.html")
	_, err := os.Stat(index)
	hasFrontend := dir != "" && err == nil

	r.NoRoute(func(c *gin.Context) {
		if rest.APINotFound(c) {
			return
		}
		if !hasFrontend || (c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) {
			c.String(http.StatusNotFound, "not found")
			return
		}
		rel := filepath.Clean("/" + c.Request.URL.Path)
		file := filepath.Join(dir, rel)
		if !strings.HasPrefix(file, filepath.Clean(dir)) {
			c.String(http.StatusNotFound, "not found")
			return
		}
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			c.File(file)
			return
		}
		c.File(index)
	})
}
