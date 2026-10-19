package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/utils"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 1 << 20

// bindJSON decodes the body strictly (unknown fields are rejected). On
// failure it answers BAD_REQUEST without echoing the decoder error.
func bindJSON(c *gin.Context, dst any) bool {
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		api.ErrorWithMessage(c, http.StatusBadRequest, "BAD_REQUEST")
		return false
	}
	return true
}

// respondError maps service errors to the envelope.
func respondError(c *gin.Context, err error) {
	var invalid *services.ConfigInvalidError
	if errors.As(err, &invalid) {
		api.ErrorWithData(c, http.StatusBadRequest, "CONFIG_INVALID",
			models.ValidationResult{Valid: false, Issues: localizeIssues(c, invalid.Issues)})
		return
	}
	var appErr *services.AppError
	if errors.As(err, &appErr) {
		if errors.Unwrap(appErr) != nil {
			utils.ShowErrorLogs(c.Request.Method+" "+c.FullPath(), err)
		}
		api.ErrorWithMessage(c, appErr.Status, appErr.Code)
		return
	}
	utils.ShowErrorLogs(c.Request.Method+" "+c.FullPath(), err)
	api.ErrorWithMessage(c, http.StatusInternalServerError, "INTERNAL_ERROR")
}
