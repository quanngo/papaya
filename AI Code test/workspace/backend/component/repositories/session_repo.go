package repositories

import (
	"context"
	"database/sql"
	"errors"

	"claimconfig/component/models"

	"github.com/jmoiron/sqlx"
)

// SessionRepo stores server-side sessions. Only sha256(token) is ever stored.
type SessionRepo struct{ db *sqlx.DB }

func NewSessionRepo(db *sqlx.DB) *SessionRepo { return &SessionRepo{db: db} }

// NewSession is the data needed to open a session.
type NewSession struct {
	UserID     int64
	TokenHash  string
	AuthMethod string
	RememberMe bool
	UserAgent  string
	IP         string
	CreatedAt  string
	ExpiresAt  string
}

func (r *SessionRepo) Create(ctx context.Context, s NewSession) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_session (user_id, token_hash, auth_method, remember_me, user_agent, ip, created_at, last_seen_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.UserID, s.TokenHash, s.AuthMethod, s.RememberMe, s.UserAgent, s.IP, s.CreatedAt, s.CreatedAt, s.ExpiresAt)
	return err
}

// GetValid returns the session for tokenHash if it is not revoked, not
// expired and its user is active; (nil, nil) otherwise.
func (r *SessionRepo) GetValid(ctx context.Context, tokenHash, now string) (*models.SessionRecord, error) {
	var s models.SessionRecord
	err := r.db.GetContext(ctx, &s, `
SELECT s.id, s.user_id, s.auth_method
FROM user_session s
JOIN app_user u ON u.id = s.user_id
WHERE s.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > ? AND u.status = 'active'`, tokenHash, now)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Revoke ends one session (logout).
func (r *SessionRepo) Revoke(ctx context.Context, tokenHash, now string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_session SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`, now, tokenHash)
	return err
}

// RevokeAllForUser ends every open session of a user except exceptHash
// (empty = all) and returns how many were revoked.
func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID int64, exceptHash, now string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE user_session SET revoked_at = ?
WHERE user_id = ? AND revoked_at IS NULL AND token_hash <> ?`, now, userID, exceptHash)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
