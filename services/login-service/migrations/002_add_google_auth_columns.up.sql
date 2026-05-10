ALTER TABLE users
  ALTER COLUMN password_hash DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS auth_provider VARCHAR(20) NOT NULL DEFAULT 'local',
  ADD COLUMN IF NOT EXISTS google_sub VARCHAR(255),
  ADD COLUMN IF NOT EXISTS avatar_url TEXT,
  ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub) WHERE google_sub IS NOT NULL;

// Note: We don't backfill google_sub for existing users here because we don't have a reliable way to link them to their Google accounts without the data from the user_identities table that we introduce in the next migration. We'll handle backfilling google_sub in the 003_refactor_identity_and_add_role migration after we create the user_identities table and populate it with data from the existing columns.
