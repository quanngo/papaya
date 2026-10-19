package rest

import (
	"net/http"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/component/tenantconfig"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	tenants services.TenantService
}

func NewConfigHandler(tenants services.TenantService) *ConfigHandler {
	return &ConfigHandler{tenants: tenants}
}

// Meta godoc
//
//	@Summary		Config enumerations
//	@Description	Claim types, notification events, channels, custom field types and module names, so the UI hard-codes none of them.
//	@Description	Requires permission `tenants.read`.
//	@Tags			configs
//	@Produce		json
//	@Success		200	{object}	api.Response{data=[]models.TenantConfigMeta}
//	@Router			/meta/tenant-config [get]
func (h *ConfigHandler) Meta(c *gin.Context) {
	api.Ok(c, models.TenantConfigMeta{
		ClaimTypes:         models.ClaimTypes,
		NotificationEvents: models.NotificationEvents,
		Channels:           models.Channels,
		CustomFieldTypes:   models.CustomFieldTypes,
		Modules:            tenantconfig.ModuleKeys(),
	})
}

// Defaults godoc
//
//	@Summary		Starting config for a new tenant
//	@Description	A minimal valid config to prefill the "new tenant" form. Requires permission `tenants.manage`.
//	@Tags			configs
//	@Produce		json
//	@Success		200	{object}	api.Response{data=[]models.TenantConfig}
//	@Router			/configs/defaults [get]
func (h *ConfigHandler) Defaults(c *gin.Context) {
	api.Ok(c, tenantconfig.DefaultConfig())
}

// Validate godoc
//
//	@Summary		Validate a config without saving
//	@Description	Returns every issue with its path (e.g. approval.tiers[1].min_amount), code and translated text. code is 200 even when the config is invalid; check data[0].valid.
//	@Description	Requires permission `tenants.manage`. Errors: BAD_REQUEST, CONFIG_REQUIRED.
//	@Tags			configs
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.ValidateConfigRequest	true	"Config to check"
//	@Success		200		{object}	api.Response{data=[]models.ValidationResult}
//	@Router			/configs/validate [post]
func (h *ConfigHandler) Validate(c *gin.Context) {
	var req models.ValidateConfigRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Config == nil {
		api.ErrorWithMessage(c, http.StatusBadRequest, services.ErrConfigRequired.Code)
		return
	}
	issues := h.tenants.Validate(req.Config)
	api.Ok(c, models.ValidationResult{Valid: len(issues) == 0, Issues: localizeIssues(c, issues)})
}
