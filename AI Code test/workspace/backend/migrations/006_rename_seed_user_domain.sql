-- 006: Move the seeded login accounts from @mq.com to @papaya.com.
--
-- 005 now seeds @papaya.com directly, so on a fresh database this is a no-op.
-- Databases created before that change still have the @mq.com accounts; this
-- renames them in place, keeping their ids, passwords, sessions and history.
-- An account is skipped if the @papaya.com address already exists.

UPDATE app_user
SET email      = replace(email, '@mq.com', '@papaya.com'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE email IN ('admin@mq.com', 'operator@mq.com', 'viewer@mq.com')
  AND NOT EXISTS (
    SELECT 1 FROM app_user p WHERE p.email = replace(app_user.email, '@mq.com', '@papaya.com')
  );
