DROP INDEX IF EXISTS idx_users_google_sub;

ALTER TABLE users
  DROP COLUMN IF EXISTS email_verified,
  DROP COLUMN IF EXISTS avatar_url,
  DROP COLUMN IF EXISTS google_sub,
  DROP COLUMN IF EXISTS auth_provider,
  ALTER COLUMN password_hash SET NOT NULL;
