// Package api holds the response envelope shared by every /api/v1 endpoint.
package api

import (
	"net/http"

	"claimconfig/i18n"

	"github.com/gin-gonic/gin"
)

// Response is the envelope every /api/v1 endpoint returns, always with HTTP 200.
// The real status is in Code, a stable error code in Message, and Data is always an array.
type Response struct {
	Code      int    `json:"code" example:"200"`
	Message   string `json:"message" example:"OK"`
	ErrorText string `json:"error_text,omitempty" example:""`
	Data      any    `json:"data"`
}

// Ok answers code 200. Each argument becomes one element of data, so the
// frontend's apiRequest reads the payload as data[0] (wrap lists in an object,
// e.g. gin.H{"tenants": list}).
func Ok(c *gin.Context, data ...any) {
	c.JSON(http.StatusOK, Response{Code: http.StatusOK, Message: "OK", Data: asArray(data)})
}

// ErrorWithMessage answers a failure: status goes in code, the stable error
// code in message, and its translation (error.<CODE>) in error_text.
func ErrorWithMessage(c *gin.Context, status int, code string) {
	ErrorWithData(c, status, code)
}

// ErrorWithData is ErrorWithMessage with a payload, e.g. CONFIG_INVALID + {issues}.
func ErrorWithData(c *gin.Context, status int, code string, data ...any) {
	c.JSON(http.StatusOK, Response{
		Code:      status,
		Message:   code,
		ErrorText: i18n.T(i18n.Locale(c), "error."+code, nil),
		Data:      asArray(data),
	})
}

func asArray(data []any) []any {
	if data == nil {
		return []any{}
	}
	return data
}
