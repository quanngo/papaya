package rest

import (
	"net/http"
	"strconv"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/middleware"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	tenants services.TenantService
}

func NewTenantHandler(tenants services.TenantService) *TenantHandler {
	return &TenantHandler{tenants: tenants}
}

// List godoc
//
//	@Summary		List tenants
//	@Description	Active tenants by name; include_archived=1 adds archived ones. Requires permission `tenants.read`.
//	@Tags			tenants
//	@Produce		json
//	@Param			include_archived	query		bool	false	"Also list archived tenants"
//	@Success		200					{object}	api.Response{data=[]models.TenantList}
//	@Router			/tenants [get]
func (h *TenantHandler) List(c *gin.Context) {
	list, err := h.tenants.List(c.Request.Context(), c.Query("include_archived") == "1" || c.Query("include_archived") == "true")
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, models.TenantList{Tenants: list})
}

// Create godoc
//
//	@Summary		Create (onboard) a tenant
//	@Description	Validates the config and stores it as version 1. No code change is needed to onboard an insurer.
//	@Description	Requires permission `tenants.manage`. Errors: BAD_REQUEST, CONFIG_REQUIRED, CONFIG_INVALID (data[0].issues), TENANT_SLUG_INVALID, TENANT_SLUG_TAKEN.
//	@Tags			tenants
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.CreateTenantRequest	true	"Slug, full config and a note"
//	@Success		200		{object}	api.Response{data=[]models.TenantDetail}
//	@Router			/tenants [post]
func (h *TenantHandler) Create(c *gin.Context) {
	var req models.CreateTenantRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.tenants.Create(c.Request.Context(), req, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, t)
}

// Get godoc
//
//	@Summary		Get a tenant with its current config
//	@Description	id may be the tenant id or its slug. Requires permission `tenants.read`. Errors: TENANT_NOT_FOUND.
//	@Tags			tenants
//	@Produce		json
//	@Param			id	path		string	true	"Tenant id or slug"	example(safeguard)
//	@Success		200	{object}	api.Response{data=[]models.TenantDetail}
//	@Router			/tenants/{id} [get]
func (h *TenantHandler) Get(c *gin.Context) {
	t, err := h.tenants.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, t)
}

// Update godoc
//
//	@Summary		Save a new config version
//	@Description	base_version must be the version the edit started from; otherwise VERSION_CONFLICT (someone saved in between).
//	@Description	Requires permission `tenants.manage`. Errors: BAD_REQUEST, CONFIG_REQUIRED, CONFIG_INVALID, VERSION_REQUIRED, VERSION_CONFLICT, TENANT_NOT_FOUND, TENANT_ARCHIVED.
//	@Tags			tenants
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Tenant id or slug"
//	@Param			body	body		models.UpdateTenantRequest	true	"Full config, note and base version"
//	@Success		200		{object}	api.Response{data=[]models.TenantDetail}
//	@Router			/tenants/{id} [put]
func (h *TenantHandler) Update(c *gin.Context) {
	var req models.UpdateTenantRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.tenants.Update(c.Request.Context(), c.Param("id"), req, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, t)
}

// Archive godoc
//
//	@Summary		Archive a tenant
//	@Description	Soft delete: the tenant disappears from lists and stops accepting claims, its history stays readable.
//	@Description	Requires permission `tenants.manage`. Errors: TENANT_NOT_FOUND, TENANT_ARCHIVED.
//	@Tags			tenants
//	@Produce		json
//	@Param			id	path		string	true	"Tenant id or slug"
//	@Success		200	{object}	api.Response
//	@Router			/tenants/{id} [delete]
func (h *TenantHandler) Archive(c *gin.Context) {
	if err := h.tenants.Archive(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c)
}

// Versions godoc
//
//	@Summary		Config history
//	@Description	Every version, newest first, without configs. Requires permission `tenants.read`. Errors: TENANT_NOT_FOUND.
//	@Tags			tenants
//	@Produce		json
//	@Param			id	path		string	true	"Tenant id or slug"
//	@Success		200	{object}	api.Response{data=[]models.VersionList}
//	@Router			/tenants/{id}/versions [get]
func (h *TenantHandler) Versions(c *gin.Context) {
	list, err := h.tenants.Versions(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, models.VersionList{Versions: list})
}

// Version godoc
//
//	@Summary		One config version
//	@Description	Requires permission `tenants.read`. Errors: BAD_REQUEST, TENANT_NOT_FOUND, VERSION_NOT_FOUND.
//	@Tags			tenants
//	@Produce		json
//	@Param			id		path		string	true	"Tenant id or slug"
//	@Param			version	path		int		true	"Version number"
//	@Success		200		{object}	api.Response{data=[]models.TenantVersion}
//	@Router			/tenants/{id}/versions/{version} [get]
func (h *TenantHandler) Version(c *gin.Context) {
	v, err := strconv.Atoi(c.Param("version"))
	if err != nil || v <= 0 {
		api.ErrorWithMessage(c, http.StatusBadRequest, "BAD_REQUEST")
		return
	}
	ver, err := h.tenants.Version(c.Request.Context(), c.Param("id"), v)
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, ver)
}

// Rollback godoc
//
//	@Summary		Roll back to a previous version
//	@Description	Creates a NEW version whose config equals the chosen one; history is never rewritten.
//	@Description	Requires permission `tenants.manage`. Errors: BAD_REQUEST, VERSION_NOT_FOUND, VERSION_CONFLICT, CONFIG_INVALID, TENANT_NOT_FOUND, TENANT_ARCHIVED.
//	@Tags			tenants
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Tenant id or slug"
//	@Param			body	body		models.RollbackRequest	true	"Version to restore"
//	@Success		200		{object}	api.Response{data=[]models.TenantDetail}
//	@Router			/tenants/{id}/rollback [post]
func (h *TenantHandler) Rollback(c *gin.Context) {
	var req models.RollbackRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.tenants.Rollback(c.Request.Context(), c.Param("id"), req, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, t)
}

// Diff godoc
//
//	@Summary		Compare two configs
//	@Description	a and b are "<id or slug>" (current version) or "<id or slug>@<version>", so it compares two tenants or two versions of one tenant.
//	@Description	Lists of scalars are compared as sets; tiers are matched by role and custom fields by key.
//	@Description	Requires permission `tenants.read`. Errors: BAD_REQUEST, TENANT_NOT_FOUND, VERSION_NOT_FOUND.
//	@Tags			tenants
//	@Produce		json
//	@Param			a	query		string	true	"Left side"		example(safeguard)
//	@Param			b	query		string	true	"Right side"	example(govhealth)
//	@Success		200	{object}	api.Response{data=[]models.DiffResult}
//	@Router			/tenants/diff [get]
func (h *TenantHandler) Diff(c *gin.Context) {
	d, err := h.tenants.Diff(c.Request.Context(), c.Query("a"), c.Query("b"))
	if err != nil {
		respondError(c, err)
		return
	}
	api.Ok(c, d)
}
