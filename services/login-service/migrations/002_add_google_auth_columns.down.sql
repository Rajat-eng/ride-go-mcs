DROP INDEX IF EXISTS idx_users_google_sub;

ALTER TABLE users
  DROP COLUMN IF EXISTS email_verified,
  DROP COLUMN IF EXISTS avatar_url,
  DROP COLUMN IF EXISTS google_sub,
  DROP COLUMN IF EXISTS auth_provider,
  ALTER COLUMN password_hash SET NOT NULL;

 // this file is intentionally left blank as the previous migration (002_add_google_auth_columns.up.sql) only added columns and an index, so we just need to remove those in the down migration.

