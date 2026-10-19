package services

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"claimconfig/component/models"
	"claimconfig/component/repositories"
	"claimconfig/component/tenantconfig"
	"claimconfig/utils"

	"github.com/google/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// TenantService manages tenants and their versioned configs. Every save and
// rollback creates a new version; nothing is overwritten.
type TenantService interface {
	List(ctx context.Context, includeArchived bool) ([]models.Tenant, error)
	Get(ctx context.Context, idOrSlug string) (*models.TenantDetail, error)
	Create(ctx context.Context, req models.CreateTenantRequest, by *int64) (*models.TenantDetail, error)
	Update(ctx context.Context, idOrSlug string, req models.UpdateTenantRequest, by *int64) (*models.TenantDetail, error)
	Archive(ctx context.Context, idOrSlug string) error
	Versions(ctx context.Context, idOrSlug string) ([]models.TenantVersion, error)
	Version(ctx context.Context, idOrSlug string, version int) (*models.TenantVersion, error)
	Rollback(ctx context.Context, idOrSlug string, req models.RollbackRequest, by *int64) (*models.TenantDetail, error)
	// Diff compares two refs of the form "<id or slug>" or "<id or slug>@<version>".
	Diff(ctx context.Context, a, b string) (*models.DiffResult, error)
	Validate(cfg *models.TenantConfig) []models.Issue
}

type tenantStore interface {
	List(ctx context.Context, includeArchived bool) ([]models.Tenant, error)
	Get(ctx context.Context, idOrSlug string) (*models.TenantDetail, error)
	Create(ctx context.Context, t models.Tenant, configJSON, note string) error
	AppendVersion(ctx context.Context, in repositories.AppendVersionInput) error
	Archive(ctx context.Context, id, now string) error
	Versions(ctx context.Context, tenantID string) ([]models.TenantVersion, error)
	Version(ctx context.Context, tenantID string, version int) (*models.TenantVersion, error)
}

type TenantServiceImpl struct {
	store tenantStore
	now   func() time.Time
}

func NewTenantService(store tenantStore) *TenantServiceImpl {
	return &TenantServiceImpl{store: store, now: time.Now}
}

func (s *TenantServiceImpl) List(ctx context.Context, includeArchived bool) ([]models.Tenant, error) {
	return s.store.List(ctx, includeArchived)
}

func (s *TenantServiceImpl) Get(ctx context.Context, idOrSlug string) (*models.TenantDetail, error) {
	t, err := s.store.Get(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	return t, nil
}

func (s *TenantServiceImpl) Validate(cfg *models.TenantConfig) []models.Issue {
	return tenantconfig.ValidateConfig(cfg)
}

// checkedConfig validates cfg and returns its JSON for storage.
func (s *TenantServiceImpl) checkedConfig(cfg *models.TenantConfig) (string, error) {
	if cfg == nil {
		return "", ErrConfigRequired
	}
	if issues := tenantconfig.ValidateConfig(cfg); len(issues) > 0 {
		return "", &ConfigInvalidError{Issues: issues}
	}
	raw, err := json.Marshal(cfg)
	return string(raw), err
}

func (s *TenantServiceImpl) Create(ctx context.Context, req models.CreateTenantRequest, by *int64) (*models.TenantDetail, error) {
	slug := strings.TrimSpace(req.Slug)
	if !slugPattern.MatchString(slug) || len(slug) < 2 {
		return nil, ErrTenantSlugInvalid
	}
	cfgJSON, err := s.checkedConfig(req.Config)
	if err != nil {
		return nil, err
	}
	now := utils.FormatTime(s.now())
	t := models.Tenant{
		ID: uuid.NewString(), Slug: slug, Name: strings.TrimSpace(req.Config.Branding.CompanyName),
		CurrentVersion: 1, CreatedBy: by, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.Create(ctx, t, cfgJSON, strings.TrimSpace(req.Note)); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, ErrTenantSlugTaken
		}
		return nil, err
	}
	return s.Get(ctx, t.ID)
}

// editable loads a tenant that may receive a new version.
func (s *TenantServiceImpl) editable(ctx context.Context, idOrSlug string) (*models.TenantDetail, error) {
	t, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t.ArchivedAt != nil {
		return nil, ErrTenantArchived
	}
	return t, nil
}

func (s *TenantServiceImpl) Update(ctx context.Context, idOrSlug string, req models.UpdateTenantRequest, by *int64) (*models.TenantDetail, error) {
	t, err := s.editable(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	if req.BaseVersion <= 0 {
		return nil, ErrVersionRequired
	}
	if req.BaseVersion != t.CurrentVersion {
		return nil, ErrVersionConflict
	}
	cfgJSON, err := s.checkedConfig(req.Config)
	if err != nil {
		return nil, err
	}
	return s.appendVersion(ctx, t, req.Config, cfgJSON, strings.TrimSpace(req.Note), "update", nil, by)
}

func (s *TenantServiceImpl) Rollback(ctx context.Context, idOrSlug string, req models.RollbackRequest, by *int64) (*models.TenantDetail, error) {
	t, err := s.editable(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	old, err := s.store.Version(ctx, t.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, ErrVersionNotFound
	}
	// Re-validate: validation rules may have become stricter since that version was saved.
	cfgJSON, err := s.checkedConfig(old.Config)
	if err != nil {
		return nil, err
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "Rollback to v" + strconv.Itoa(req.Version)
	}
	from := req.Version
	return s.appendVersion(ctx, t, old.Config, cfgJSON, note, "rollback", &from, by)
}

func (s *TenantServiceImpl) appendVersion(ctx context.Context, t *models.TenantDetail, cfg *models.TenantConfig, cfgJSON, note, source string, from *int, by *int64) (*models.TenantDetail, error) {
	err := s.store.AppendVersion(ctx, repositories.AppendVersionInput{
		TenantID: t.ID, BaseVersion: t.CurrentVersion, Name: strings.TrimSpace(cfg.Branding.CompanyName),
		ConfigJSON: cfgJSON, Note: note, Source: source, RolledBackFrom: from, CreatedBy: by,
		Now: utils.FormatTime(s.now()),
	})
	if errors.Is(err, repositories.ErrConflict) {
		return nil, ErrVersionConflict
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, t.ID)
}

func (s *TenantServiceImpl) Archive(ctx context.Context, idOrSlug string) error {
	t, err := s.editable(ctx, idOrSlug)
	if err != nil {
		return err
	}
	return s.store.Archive(ctx, t.ID, utils.FormatTime(s.now()))
}

func (s *TenantServiceImpl) Versions(ctx context.Context, idOrSlug string) ([]models.TenantVersion, error) {
	t, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	list, err := s.store.Versions(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].IsCurrent = list[i].Version == t.CurrentVersion
	}
	return list, nil
}

func (s *TenantServiceImpl) Version(ctx context.Context, idOrSlug string, version int) (*models.TenantVersion, error) {
	t, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	v, err := s.store.Version(ctx, t.ID, version)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrVersionNotFound
	}
	v.IsCurrent = v.Version == t.CurrentVersion
	return v, nil
}

func (s *TenantServiceImpl) Diff(ctx context.Context, a, b string) (*models.DiffResult, error) {
	left, lcfg, err := s.resolveRef(ctx, a)
	if err != nil {
		return nil, err
	}
	right, rcfg, err := s.resolveRef(ctx, b)
	if err != nil {
		return nil, err
	}
	return &models.DiffResult{A: left, B: right, Changes: tenantconfig.Diff(lcfg, rcfg)}, nil
}

func (s *TenantServiceImpl) resolveRef(ctx context.Context, ref string) (models.DiffSide, *models.TenantConfig, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return models.DiffSide{}, nil, ErrBadRequest
	}
	idOrSlug, version := ref, 0
	if i := strings.LastIndex(ref, "@"); i > 0 {
		v, err := strconv.Atoi(ref[i+1:])
		if err != nil || v <= 0 {
			return models.DiffSide{}, nil, ErrBadRequest
		}
		idOrSlug, version = ref[:i], v
	}
	t, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return models.DiffSide{}, nil, err
	}
	side := models.DiffSide{TenantID: t.ID, Slug: t.Slug, Name: t.Name, Version: t.CurrentVersion}
	if version == 0 || version == t.CurrentVersion {
		return side, &t.Config, nil
	}
	v, err := s.store.Version(ctx, t.ID, version)
	if err != nil {
		return side, nil, err
	}
	if v == nil {
		return side, nil, ErrVersionNotFound
	}
	side.Version = version
	return side, v.Config, nil
}
