-- 003: Multi-tenant configuration (AI Challenge 15).
--
-- tenant                 one row per insurer; points at its current config version.
-- tenant_config_version  append-only history of each tenant's full config (JSON).
--                        Every save and every rollback inserts a new version;
--                        rows are never updated or deleted (enforced by triggers).
--
-- SQLite notes:
--   * Foreign keys are only enforced when the connection sets
--     PRAGMA foreign_keys = ON (the DSN passes _pragma=foreign_keys(1)).
--   * Timestamps are TEXT in UTC ISO-8601 (e.g. 2026-10-17T08:00:00.000Z),
--     written by Go; the defaults only cover manual inserts.
--   * tenant.current_version -> tenant_config_version is DEFERRABLE, so the
--     tenant row and its version-1 row are inserted in the same transaction
--     and checked at COMMIT.

CREATE TABLE IF NOT EXISTS tenant (
  id              TEXT    PRIMARY KEY,                       -- UUID
  slug            TEXT    NOT NULL UNIQUE
                  CHECK (length(slug) BETWEEN 2 AND 64
                         AND slug NOT GLOB '*[^a-z0-9-]*'
                         AND slug NOT GLOB '-*'
                         AND slug NOT GLOB '*-'),
  name            TEXT    NOT NULL CHECK (length(trim(name)) > 0), -- copy of branding.company_name, for listing
  current_version INTEGER NOT NULL CHECK (current_version >= 1),
  archived_at     TEXT,                                      -- soft delete; NULL = active
  created_by      INTEGER REFERENCES app_user (id),          -- NULL for seed data
  created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  FOREIGN KEY (id, current_version)
    REFERENCES tenant_config_version (tenant_id, version)
    DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_tenant_active_name
  ON tenant (name) WHERE archived_at IS NULL;

CREATE TABLE IF NOT EXISTS tenant_config_version (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id        TEXT    NOT NULL REFERENCES tenant (id) DEFERRABLE INITIALLY DEFERRED,
  version          INTEGER NOT NULL CHECK (version >= 1),
  config           TEXT    NOT NULL CHECK (json_valid(config) AND json_type(config) = 'object'),
  note             TEXT    NOT NULL DEFAULT '',
  source           TEXT    NOT NULL CHECK (source IN ('create', 'update', 'rollback')),
  rolled_back_from INTEGER,                                  -- version copied by a rollback
  created_by       INTEGER REFERENCES app_user (id),         -- NULL for seed data
  created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  UNIQUE (tenant_id, version),
  CHECK ((source = 'rollback') = (rolled_back_from IS NOT NULL)),
  CHECK (source <> 'create' OR version = 1),
  CHECK (rolled_back_from IS NULL OR rolled_back_from < version),
  FOREIGN KEY (tenant_id, rolled_back_from)
    REFERENCES tenant_config_version (tenant_id, version)
);

-- Versions are numbered 1, 2, 3 … per tenant with no gaps. Two concurrent saves
-- based on the same version both compute the same number; this trigger (or, as
-- a backstop, UNIQUE (tenant_id, version)) rejects the second one, and the
-- service reports either error as VERSION_CONFLICT.
CREATE TRIGGER IF NOT EXISTS trg_tenant_config_version_sequential
BEFORE INSERT ON tenant_config_version
WHEN NEW.version <> COALESCE(
       (SELECT MAX(version) FROM tenant_config_version WHERE tenant_id = NEW.tenant_id), 0) + 1
BEGIN
  SELECT RAISE(ABORT, 'tenant_config_version: version must be previous max + 1');
END;

-- History is immutable: rollback means inserting a new version.
CREATE TRIGGER IF NOT EXISTS trg_tenant_config_version_no_update
BEFORE UPDATE ON tenant_config_version
BEGIN
  SELECT RAISE(ABORT, 'tenant_config_version is append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_tenant_config_version_no_delete
BEFORE DELETE ON tenant_config_version
BEGIN
  SELECT RAISE(ABORT, 'tenant_config_version is append-only');
END;

-- Tenants are archived, never deleted (their history must stay readable).
CREATE TRIGGER IF NOT EXISTS trg_tenant_no_delete
BEFORE DELETE ON tenant
BEGIN
  SELECT RAISE(ABORT, 'tenant cannot be deleted; set archived_at instead');
END;

-- current_version can only move to the newest version of that tenant
-- (a save or rollback always creates the newest version first).
CREATE TRIGGER IF NOT EXISTS trg_tenant_current_is_latest
BEFORE UPDATE OF current_version ON tenant
WHEN NEW.current_version <> (SELECT MAX(version) FROM tenant_config_version WHERE tenant_id = NEW.id)
BEGIN
  SELECT RAISE(ABORT, 'tenant.current_version must be the latest version');
END;
