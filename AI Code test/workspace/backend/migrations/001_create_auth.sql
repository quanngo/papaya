-- 001: Admin-app authentication and RBAC (SQLite port of the base code's
-- 002–006 Postgres migrations: auth, remember-me, RBAC, user_config).
--
-- role / permission / role_permission  RBAC: one role per user, a role grants
--                                      permission codes. Code checks permission
--                                      codes, never role names.
-- app_user                             internal (bcrypt) and/or Google accounts.
-- user_session                         server-side sessions; only sha256(token) is stored.
-- oauth_login_attempt                  one-shot OAuth state/nonce/PKCE, consumed on callback.
-- user_config                          per-user settings, 1-1 with app_user (locale, EC-7).
--
-- SQLite conventions used across all migrations:
--   * timestamps: TEXT, UTC ISO-8601 with milliseconds, written by Go;
--   * booleans: INTEGER 0/1 with a CHECK;
--   * foreign keys need PRAGMA foreign_keys = ON (set in the DSN).

CREATE TABLE IF NOT EXISTS role (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  code       TEXT    NOT NULL UNIQUE CHECK (code GLOB '[a-z]*' AND code NOT GLOB '*[^a-z0-9_]*'),
  name       TEXT    NOT NULL,                -- fallback; UI uses i18n role.<code>
  is_system  INTEGER NOT NULL DEFAULT 0 CHECK (is_system IN (0, 1)),
  created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS permission (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  code       TEXT    NOT NULL UNIQUE CHECK (code GLOB '[a-z]*.[a-z]*' AND code NOT GLOB '*[^a-z0-9_.]*'),
  name       TEXT    NOT NULL,                -- fallback; UI uses i18n permission.<code>
  created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS role_permission (
  role_id       INTEGER NOT NULL REFERENCES role (id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES permission (id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS app_user (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  email               TEXT    NOT NULL UNIQUE CHECK (email = lower(trim(email)) AND email LIKE '%_@_%'),
  display_name        TEXT    NOT NULL CHECK (length(trim(display_name)) > 0),
  password_hash       TEXT,                   -- bcrypt; NULL = Google-only account
  google_sub          TEXT    UNIQUE,         -- Google account link; NULL until first Google login
  role_id             INTEGER NOT NULL REFERENCES role (id),
  status              TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  failed_login_count  INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
  locked_until        TEXT,                   -- set after too many wrong passwords
  last_login_at       TEXT,
  password_changed_at TEXT,
  created_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_app_user_role ON app_user (role_id);

CREATE TABLE IF NOT EXISTS user_session (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id      INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
  token_hash   TEXT    NOT NULL UNIQUE,       -- hex sha256 of the cookie token
  auth_method  TEXT    NOT NULL CHECK (auth_method IN ('password', 'google')),
  remember_me  INTEGER NOT NULL DEFAULT 0 CHECK (remember_me IN (0, 1)),
  user_agent   TEXT,
  ip           TEXT,
  created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  last_seen_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  expires_at   TEXT    NOT NULL,
  revoked_at   TEXT,
  CHECK (expires_at > created_at)
);

-- Valid-session lookup is by token_hash (UNIQUE above); these serve
-- logout-all / revoke-on-disable and the cleanup job.
CREATE INDEX IF NOT EXISTS idx_user_session_user_active ON user_session (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_session_expires ON user_session (expires_at);

CREATE TABLE IF NOT EXISTS oauth_login_attempt (
  state         TEXT    PRIMARY KEY,          -- random, single use (DELETE … RETURNING)
  nonce         TEXT    NOT NULL,
  code_verifier TEXT    NOT NULL,             -- PKCE
  next_path     TEXT    NOT NULL DEFAULT '/', -- already sanitized (no open redirect)
  remember_me   INTEGER NOT NULL DEFAULT 0 CHECK (remember_me IN (0, 1)),
  created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  expires_at    TEXT    NOT NULL
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_oauth_login_attempt_expires ON oauth_login_attempt (expires_at);

CREATE TABLE IF NOT EXISTS user_config (
  user_id    INTEGER PRIMARY KEY REFERENCES app_user (id) ON DELETE CASCADE,
  locale     TEXT    NOT NULL DEFAULT 'vi'
             CONSTRAINT ck_user_config_locale CHECK (locale IN ('vi', 'en')),
  updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
