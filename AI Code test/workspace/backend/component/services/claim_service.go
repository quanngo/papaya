package services

import (
	"context"
	"math"
	"strings"
	"time"

	"claimconfig/component/models"
	"claimconfig/component/repositories"
	"claimconfig/component/tenantconfig"
	"claimconfig/utils"

	"github.com/google/uuid"
)

// ClaimService runs processClaim for saved tenants (stored) or for an
// unsaved config (preview, never stored).
type ClaimService interface {
	Preview(ctx context.Context, req models.PreviewClaimRequest) (*models.ProcessResult, error)
	Process(ctx context.Context, req models.ProcessClaimRequest, by *int64) (*models.ProcessResult, error)
	ProcessBatch(ctx context.Context, req models.ProcessBatchRequest, by *int64) ([]models.ProcessResult, error)
	List(ctx context.Context, tenantRef string, limit int) ([]models.ClaimRecord, error)
}

type tenantReader interface {
	Get(ctx context.Context, idOrSlug string) (*models.TenantDetail, error)
	List(ctx context.Context, includeArchived bool) ([]models.Tenant, error)
}

type claimStore interface {
	Create(ctx context.Context, c repositories.NewClaim) error
	List(ctx context.Context, tenantID string, limit int) ([]models.ClaimRecord, error)
}

type ClaimServiceImpl struct {
	tenants tenantReader
	claims  claimStore
	now     func() time.Time
}

func NewClaimService(tenants tenantReader, claims claimStore) *ClaimServiceImpl {
	return &ClaimServiceImpl{tenants: tenants, claims: claims, now: time.Now}
}

func validateClaim(c models.Claim) error {
	switch {
	case strings.TrimSpace(c.ClaimType) == "":
		return ErrClaimTypeRequired
	case !contains(models.ClaimTypes, c.ClaimType):
		return ErrClaimTypeUnknown
	case c.Amount < 0 || math.IsNaN(c.Amount) || math.IsInf(c.Amount, 0):
		return ErrClaimAmountInvalid
	}
	return nil
}

func ref(t *models.TenantDetail) models.TenantRef {
	return models.TenantRef{ID: t.ID, Slug: t.Slug, Name: t.Name, ConfigVersion: t.CurrentVersion}
}

func (s *ClaimServiceImpl) Preview(ctx context.Context, req models.PreviewClaimRequest) (*models.ProcessResult, error) {
	if err := validateClaim(req.Claim); err != nil {
		return nil, err
	}
	var (
		cfg    *models.TenantConfig
		tenant models.TenantRef
	)
	if req.TenantID != "" {
		t, err := s.activeTenant(ctx, req.TenantID)
		if err != nil {
			return nil, err
		}
		cfg, tenant = &t.Config, ref(t)
	}
	if req.Config != nil {
		// Unsaved config from the edit form: version 0 marks it as not stored.
		if issues := tenantconfig.ValidateConfig(req.Config); len(issues) > 0 {
			return nil, &ConfigInvalidError{Issues: issues}
		}
		cfg = req.Config
		tenant.ConfigVersion = 0
		if tenant.Name == "" {
			tenant.Name = req.Config.Branding.CompanyName
		}
	}
	if cfg == nil {
		return nil, ErrTenantRequired
	}
	res := tenantconfig.ProcessClaim(cfg, tenant, req.Claim, s.now())
	return &res, nil
}

func (s *ClaimServiceImpl) Process(ctx context.Context, req models.ProcessClaimRequest, by *int64) (*models.ProcessResult, error) {
	if err := validateClaim(req.Claim); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return nil, ErrTenantRequired
	}
	t, err := s.activeTenant(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	return s.processAndStore(ctx, t, req.Claim, by)
}

func (s *ClaimServiceImpl) ProcessBatch(ctx context.Context, req models.ProcessBatchRequest, by *int64) ([]models.ProcessResult, error) {
	if err := validateClaim(req.Claim); err != nil {
		return nil, err
	}
	refs := req.TenantIDs
	if len(refs) == 0 {
		active, err := s.tenants.List(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, t := range active {
			refs = append(refs, t.ID)
		}
	}
	// Resolve every tenant first so a bad id fails the batch before anything is stored.
	tenants := make([]*models.TenantDetail, 0, len(refs))
	for _, r := range refs {
		t, err := s.activeTenant(ctx, r)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	out := make([]models.ProcessResult, 0, len(tenants))
	for _, t := range tenants {
		res, err := s.processAndStore(ctx, t, req.Claim, by)
		if err != nil {
			return nil, err
		}
		out = append(out, *res)
	}
	return out, nil
}

func (s *ClaimServiceImpl) processAndStore(ctx context.Context, t *models.TenantDetail, claim models.Claim, by *int64) (*models.ProcessResult, error) {
	now := s.now()
	if claim.SubmittedAt == nil {
		claim.SubmittedAt = &now
	}
	res := tenantconfig.ProcessClaim(&t.Config, ref(t), claim, now)
	res.ClaimID = uuid.NewString()
	err := s.claims.Create(ctx, repositories.NewClaim{
		ID: res.ClaimID, Claim: claim, SubmittedAt: utils.FormatTime(*claim.SubmittedAt),
		Result: res, CreatedBy: by, CreatedAt: utils.FormatTime(now),
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *ClaimServiceImpl) List(ctx context.Context, tenantRef string, limit int) ([]models.ClaimRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tenantID := ""
	if tenantRef != "" {
		t, err := s.tenants.Get(ctx, tenantRef)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, ErrTenantNotFound
		}
		tenantID = t.ID
	}
	return s.claims.List(ctx, tenantID, limit)
}

func (s *ClaimServiceImpl) activeTenant(ctx context.Context, idOrSlug string) (*models.TenantDetail, error) {
	t, err := s.tenants.Get(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	if t.ArchivedAt != nil {
		return nil, ErrTenantArchived
	}
	return t, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
