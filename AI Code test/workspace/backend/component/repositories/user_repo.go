package repositories

import (
	"context"
	"database/sql"
	"errors"

	"claimconfig/component/models"

	"github.com/jmoiron/sqlx"
)

// UserRepo reads and writes app_user (+ user_config for the locale).
type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

const appUserColumns = `
  u.id, u.email, u.display_name, u.password_hash, u.role_id,
  r.code AS role_code, r.name AS role_name, u.status,
  u.failed_login_count, u.locked_until, COALESCE(uc.locale, 'vi') AS locale
FROM app_user u
JOIN role r ON r.id = u.role_id
LEFT JOIN user_config uc ON uc.user_id = u.id`

// GetByEmail returns (nil, nil) when no user has that email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT`+appUserColumns+` WHERE u.email = ?`, email)
}

// GetByID returns (nil, nil) when the user does not exist.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.AppUser, error) {
	return r.getOne(ctx, `SELECT`+appUserColumns+` WHERE u.id = ?`, id)
}

func (r *UserRepo) getOne(ctx context.Context, q string, arg any) (*models.AppUser, error) {
	var u models.AppUser
	if err := r.db.GetContext(ctx, &u, q, arg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// RegisterFailedLogin counts a wrong password; reaching max locks the account
// until lockUntil and restarts the count.
func (r *UserRepo) RegisterFailedLogin(ctx context.Context, id int64, max int, lockUntil, now string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE app_user SET
  locked_until       = CASE WHEN failed_login_count + 1 >= ? THEN ? ELSE locked_until END,
  failed_login_count = CASE WHEN failed_login_count + 1 >= ? THEN 0 ELSE failed_login_count + 1 END,
  updated_at = ?
WHERE id = ?`, max, lockUntil, max, now, id)
	return err
}

// RecordLogin clears the failure counter and lock after a successful login.
func (r *UserRepo) RecordLogin(ctx context.Context, id int64, now string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE app_user SET failed_login_count = 0, locked_until = NULL, last_login_at = ?, updated_at = ?
WHERE id = ?`, now, now, id)
	return err
}

// SetPassword stores a new bcrypt hash.
func (r *UserRepo) SetPassword(ctx context.Context, id int64, hash, now string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE app_user SET password_hash = ?, password_changed_at = ?, updated_at = ? WHERE id = ?`, hash, now, now, id)
	return err
}

// SetLocale upserts the user's UI language.
func (r *UserRepo) SetLocale(ctx context.Context, id int64, locale, now string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_config (user_id, locale, updated_at) VALUES (?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET locale = excluded.locale, updated_at = excluded.updated_at`, id, locale, now)
	return err
}

// Create inserts a user and its user_config row; ErrDuplicate if the email exists.
func (r *UserRepo) Create(ctx context.Context, email, name string, passwordHash *string, roleID int64, locale, now string) (int64, error) {
	var id int64
	err := WithTx(ctx, r.db, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, `
INSERT INTO app_user (email, display_name, password_hash, role_id, status, created_at, updated_at)
VALUES (?, ?, ?, ?, 'active', ?, ?)`, email, name, passwordHash, roleID, now, now)
		if err != nil {
			if isUnique(err) {
				return ErrDuplicate
			}
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO user_config (user_id, locale, updated_at) VALUES (?, ?, ?)`, id, locale, now)
		return err
	})
	return id, err
}
