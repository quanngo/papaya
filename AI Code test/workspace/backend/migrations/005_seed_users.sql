-- 005: Seed login accounts (password login, bcrypt cost 12).
--
-- | Email               | Role     | Password      | Purpose                              |
-- |---------------------|----------|---------------|--------------------------------------|
-- | admin@papaya.com    | admin    | 123456        | default admin (base-code convention) |
-- | operator@papaya.com | operator | Operator@2026 | ops team: onboard/edit tenants       |
-- | viewer@papaya.com   | viewer   | Viewer@2026!  | read-only: list, diff, preview       |
--
-- SECURITY: these are seeded in EVERY environment. Change the passwords right
-- after the first login on any shared/deployed instance (the backend logs a
-- [SECURITY] warning while admin@papaya.com still uses the default password).
-- Seeded hashes bypass auth.min-password-length; every password set through
-- the app must satisfy it.
--
-- Existing accounts are never touched (INSERT OR IGNORE on email), so
-- re-running this cannot reset a changed password.

INSERT OR IGNORE INTO app_user (email, display_name, password_hash, role_id, status)
SELECT 'admin@papaya.com', 'Administrator',
       '$2a$12$4KNnm.Uza0Ziz2p8sCbq9O6LGIfnjik.ySFxlYs70xz7vXvP.i/aa',
       id, 'active'
FROM role WHERE code = 'admin';

INSERT OR IGNORE INTO app_user (email, display_name, password_hash, role_id, status)
SELECT 'operator@papaya.com', 'Operations Demo',
       '$2a$12$Kx1BPR.wiG1mWFDFrTfv/.htrKm2Z.Q.3eAK8ENeubEcx4Y3q2Xg.',
       id, 'active'
FROM role WHERE code = 'operator';

INSERT OR IGNORE INTO app_user (email, display_name, password_hash, role_id, status)
SELECT 'viewer@papaya.com', 'Viewer Demo',
       '$2a$12$tAxnvxauWDMR3XWZVLSx..aqhdvIMsJKSNQEQ3ZZ9iCFm1Ss41G16',
       id, 'active'
FROM role WHERE code = 'viewer';

-- Every user has a user_config row (code also treats a missing row as 'vi').
INSERT OR IGNORE INTO user_config (user_id, locale)
SELECT id, 'vi' FROM app_user;
