package rest

import (
	"claimconfig/component/api"
	"claimconfig/component/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type HealthHandler struct {
	db      *sqlx.DB
	version string
}

func NewHealthHandler(db *sqlx.DB, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// Get godoc
//
//	@Summary		Health check
//	@Description	Liveness and database connectivity. Public.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	api.Response{data=[]models.HealthStatus}
//	@Router			/health [get]
func (h *HealthHandler) Get(c *gin.Context) {
	status := models.HealthStatus{Status: "ok", DB: "ok", Version: h.version}
	if err := h.db.PingContext(c.Request.Context()); err != nil {
		status.Status, status.DB = "degraded", "unreachable"
	}
	api.Ok(c, status)
}
