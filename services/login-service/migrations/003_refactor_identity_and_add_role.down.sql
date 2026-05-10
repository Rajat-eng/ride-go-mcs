ALTER TABLE users
  ADD COLUMN IF NOT EXISTS auth_provider VARCHAR(20) NOT NULL DEFAULT 'local',
  ADD COLUMN IF NOT EXISTS google_sub VARCHAR(255),
  ADD COLUMN IF NOT EXISTS avatar_url TEXT,
  ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;

UPDATE users u
SET
  auth_provider = 'google',
  google_sub = ui.provider_subject,
  avatar_url = ui.avatar_url,
  email_verified = ui.email_verified
FROM user_identities ui
WHERE ui.user_id = u.id AND ui.provider = 'google';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub) WHERE google_sub IS NOT NULL;

DROP TABLE IF EXISTS user_identities;

ALTER TABLE users
  DROP COLUMN IF EXISTS role;
