package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"claimconfig/component/models"

	"github.com/jmoiron/sqlx"
)

// TenantRepo reads and writes tenant and its append-only tenant_config_version.
type TenantRepo struct{ db *sqlx.DB }

func NewTenantRepo(db *sqlx.DB) *TenantRepo { return &TenantRepo{db: db} }

const tenantColumns = `t.id, t.slug, t.name, t.current_version, t.archived_at, t.created_by, t.created_at, t.updated_at`

func (r *TenantRepo) List(ctx context.Context, includeArchived bool) ([]models.Tenant, error) {
	q := `SELECT ` + tenantColumns + ` FROM tenant t`
	if !includeArchived {
		q += ` WHERE t.archived_at IS NULL`
	}
	q += ` ORDER BY t.name COLLATE NOCASE`
	out := []models.Tenant{}
	return out, r.db.SelectContext(ctx, &out, q)
}

// Get returns a tenant by id or slug with its current config; (nil, nil) if none.
func (r *TenantRepo) Get(ctx context.Context, idOrSlug string) (*models.TenantDetail, error) {
	var row struct {
		models.Tenant
		Config string `db:"config"`
	}
	err := r.db.GetContext(ctx, &row, `
SELECT `+tenantColumns+`, v.config
FROM tenant t
JOIN tenant_config_version v ON v.tenant_id = t.id AND v.version = t.current_version
WHERE t.id = ? OR t.slug = ?`, idOrSlug, idOrSlug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := &models.TenantDetail{Tenant: row.Tenant}
	if err := json.Unmarshal([]byte(row.Config), &d.Config); err != nil {
		return nil, fmt.Errorf("tenant %s config: %w", row.ID, err)
	}
	return d, nil
}

// Create inserts the tenant and its version 1 in one transaction (the
// deferred foreign key between them is checked at commit).
func (r *TenantRepo) Create(ctx context.Context, t models.Tenant, configJSON, note string) error {
	return WithTx(ctx, r.db, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO tenant (id, slug, name, current_version, created_by, created_at, updated_at)
VALUES (?, ?, ?, 1, ?, ?, ?)`, t.ID, t.Slug, t.Name, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
		if err != nil {
			if isUnique(err) {
				return ErrDuplicate
			}
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO tenant_config_version (tenant_id, version, config, note, source, created_by, created_at)
VALUES (?, 1, ?, ?, 'create', ?, ?)`, t.ID, configJSON, note, t.CreatedBy, t.CreatedAt)
		return err
	})
}

// AppendVersionInput describes a save or rollback.
type AppendVersionInput struct {
	TenantID       string
	BaseVersion    int // the version the edit started from
	Name           string
	ConfigJSON     string
	Note           string
	Source         string // update | rollback
	RolledBackFrom *int
	CreatedBy      *int64
	Now            string
}

// AppendVersion inserts version BaseVersion+1 and moves current_version to it.
// It returns ErrConflict when BaseVersion is no longer the current version.
func (r *TenantRepo) AppendVersion(ctx context.Context, in AppendVersionInput) error {
	return WithTx(ctx, r.db, func(tx *sqlx.Tx) error {
		next := in.BaseVersion + 1
		_, err := tx.ExecContext(ctx, `
INSERT INTO tenant_config_version (tenant_id, version, config, note, source, rolled_back_from, created_by, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, in.TenantID, next, in.ConfigJSON, in.Note, in.Source, in.RolledBackFrom, in.CreatedBy, in.Now)
		if err != nil {
			if isVersionRace(err) || isUnique(err) {
				return ErrConflict
			}
			return err
		}
		res, err := tx.ExecContext(ctx, `
UPDATE tenant SET current_version = ?, name = ?, updated_at = ?
WHERE id = ? AND current_version = ?`, next, in.Name, in.Now, in.TenantID, in.BaseVersion)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrConflict
		}
		return nil
	})
}

// Archive soft-deletes a tenant (history stays readable).
func (r *TenantRepo) Archive(ctx context.Context, id, now string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenant SET archived_at = ?, updated_at = ? WHERE id = ? AND archived_at IS NULL`, now, now, id)
	return err
}

const versionColumns = `v.tenant_id, v.version, v.note, v.source, v.rolled_back_from, v.created_by, u.email AS created_by_email, v.created_at`

// Versions lists a tenant's history, newest first, without the configs.
func (r *TenantRepo) Versions(ctx context.Context, tenantID string) ([]models.TenantVersion, error) {
	out := []models.TenantVersion{}
	err := r.db.SelectContext(ctx, &out, `
SELECT `+versionColumns+`
FROM tenant_config_version v LEFT JOIN app_user u ON u.id = v.created_by
WHERE v.tenant_id = ? ORDER BY v.version DESC`, tenantID)
	return out, err
}

// Version returns one version with its config; (nil, nil) if it does not exist.
func (r *TenantRepo) Version(ctx context.Context, tenantID string, version int) (*models.TenantVersion, error) {
	var row struct {
		models.TenantVersion
		Config string `db:"config"`
	}
	err := r.db.GetContext(ctx, &row, `
SELECT `+versionColumns+`, v.config
FROM tenant_config_version v LEFT JOIN app_user u ON u.id = v.created_by
WHERE v.tenant_id = ? AND v.version = ?`, tenantID, version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := row.TenantVersion
	var cfg models.TenantConfig
	if err := json.Unmarshal([]byte(row.Config), &cfg); err != nil {
		return nil, fmt.Errorf("tenant %s v%d config: %w", tenantID, version, err)
	}
	v.Config = &cfg
	return &v, nil
}

// Count returns the number of tenants (used to decide whether to seed).
func (r *TenantRepo) Count(ctx context.Context) (int, error) {
	var n int
	return n, r.db.GetContext(ctx, &n, `SELECT count(*) FROM tenant`)
}
