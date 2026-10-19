package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// RoleRepo reads roles and their permission codes.
type RoleRepo struct{ db *sqlx.DB }

func NewRoleRepo(db *sqlx.DB) *RoleRepo { return &RoleRepo{db: db} }

// IDByCode returns 0 when the role does not exist.
func (r *RoleRepo) IDByCode(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `SELECT id FROM role WHERE code = ?`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// PermissionCodes returns the permission codes granted to a role, sorted.
func (r *RoleRepo) PermissionCodes(ctx context.Context, roleID int64) ([]string, error) {
	codes := []string{}
	err := r.db.SelectContext(ctx, &codes, `
SELECT p.code FROM role_permission rp JOIN permission p ON p.id = rp.permission_id
WHERE rp.role_id = ? ORDER BY p.id`, roleID)
	return codes, err
}
