-- 002: Seed roles, permissions and their grants.
--
-- | Role     | Permissions                                                      |
-- |----------|------------------------------------------------------------------|
-- | admin    | everything                                                       |
-- | operator | tenants.read, tenants.manage, claims.read, claims.process        |
-- |          | (the operations team onboards insurers without code changes)     |
-- | viewer   | tenants.read, claims.read                                        |
--
-- Names here are fallbacks; the UI shows role.<code> / permission.<code>
-- from i18n/locales/{vi,en}.json. Re-runnable (INSERT OR IGNORE).

INSERT OR IGNORE INTO role (code, name, is_system) VALUES
  ('admin',    'Administrator', 1),
  ('operator', 'Operator',      1),
  ('viewer',   'Viewer',        1);

INSERT OR IGNORE INTO permission (code, name) VALUES
  ('users.read',      'View users and roles'),
  ('users.manage',    'Create users, change roles and status, reset passwords'),
  ('tenants.read',    'View tenants, config history, diff and preview'),
  ('tenants.manage',  'Create, edit, archive and roll back tenant configs'),
  ('claims.read',     'View processed claims'),
  ('claims.process',  'Submit claims to processClaim');

INSERT OR IGNORE INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r
JOIN permission p ON
     r.code = 'admin'
  OR (r.code = 'operator' AND p.code IN ('tenants.read', 'tenants.manage', 'claims.read', 'claims.process'))
  OR (r.code = 'viewer'   AND p.code IN ('tenants.read', 'claims.read'));
