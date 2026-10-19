package rest

import (
	"strconv"

	"claimconfig/component/api"
	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/middleware"

	"github.com/gin-gonic/gin"
)

type ClaimHandler struct {
	claims services.ClaimService
}

func NewClaimHandler(claims services.ClaimService) *ClaimHandler {
	return &ClaimHandler{claims: claims}
}

// Preview godoc
//
//	@Summary		Preview how a claim would be processed
//	@Description	Runs the same engine as /claims/process but stores nothing. Send tenant_id for a saved tenant, or config for an unsaved one (the edit form); with both, config wins and tenant_id only names it.
//	@Description	A claim the tenant does not accept still answers code 200 with data[0].accepted=false and data[0].errors.
//	@Description	Requires permission `tenants.read`. Errors: BAD_REQUEST, TENANT_REQUIRED, CONFIG_INVALID, CLAIM_TYPE_REQUIRED, CLAIM_TYPE_UNKNOWN, CLAIM_AMOUNT_INVALID, TENANT_NOT_FOUND, TENANT_ARCHIVED.
//	@Tags			claims
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.PreviewClaimRequest	true	"Tenant or config, plus the claim"
//	@Success		200		{object}	api.Response{data=[]models.ProcessResult}
//	@Router			/claims/preview [post]
func (h *ClaimHandler) Preview(c *gin.Context) {
	var req models.PreviewClaimRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.claims.Preview(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	localizeResult(c, res)
	api.Ok(c, res)
}

// Process godoc
//
//	@Summary		processClaim(tenantId, claimData)
//	@Description	Returns required documents, approval routing (tier and approver role), notifications with channels, the SLA deadline and custom-field validation, and stores the claim with the config version used.
//	@Description	Requires permission `claims.process`. Errors: BAD_REQUEST, TENANT_REQUIRED, CLAIM_TYPE_REQUIRED, CLAIM_TYPE_UNKNOWN, CLAIM_AMOUNT_INVALID, TENANT_NOT_FOUND, TENANT_ARCHIVED.
//	@Tags			claims
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.ProcessClaimRequest	true	"Tenant id or slug, and the claim"
//	@Success		200		{object}	api.Response{data=[]models.ProcessResult}
//	@Router			/claims/process [post]
func (h *ClaimHandler) Process(c *gin.Context) {
	var req models.ProcessClaimRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.claims.Process(c.Request.Context(), req, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	localizeResult(c, res)
	api.Ok(c, res)
}

// ProcessBatch godoc
//
//	@Summary		Process one claim against several tenants
//	@Description	Demonstrates that identical claims are handled differently per tenant. Empty tenant_ids = every active tenant. Each result is stored like /claims/process.
//	@Description	Requires permission `claims.process`. Errors: as /claims/process.
//	@Tags			claims
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.ProcessBatchRequest	true	"Tenants (ids or slugs) and the claim"
//	@Success		200		{object}	api.Response{data=[]models.BatchResult}
//	@Router			/claims/process-batch [post]
func (h *ClaimHandler) ProcessBatch(c *gin.Context) {
	var req models.ProcessBatchRequest
	if !bindJSON(c, &req) {
		return
	}
	results, err := h.claims.ProcessBatch(c.Request.Context(), req, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	for i := range results {
		localizeResult(c, &results[i])
	}
	api.Ok(c, models.BatchResult{Results: results})
}

// List godoc
//
//	@Summary		Processed claims
//	@Description	Newest first. Requires permission `claims.read`. Errors: TENANT_NOT_FOUND.
//	@Tags			claims
//	@Produce		json
//	@Param			tenant_id	query		string	false	"Only this tenant (id or slug)"
//	@Param			limit		query		int		false	"Max rows (default 50, max 200)"
//	@Success		200			{object}	api.Response{data=[]models.ClaimList}
//	@Router			/claims [get]
func (h *ClaimHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	list, err := h.claims.List(c.Request.Context(), c.Query("tenant_id"), limit)
	if err != nil {
		respondError(c, err)
		return
	}
	for i := range list {
		if list[i].Result != nil {
			localizeResult(c, list[i].Result)
		}
	}
	api.Ok(c, models.ClaimList{Claims: list})
}
