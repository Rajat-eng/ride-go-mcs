ALTER TABLE users
  ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'rider';

CREATE TABLE IF NOT EXISTS user_identities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider VARCHAR(32) NOT NULL,
  provider_subject VARCHAR(255) NOT NULL,
  avatar_url TEXT,
  email_verified BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE (provider, provider_subject),
  UNIQUE (user_id, provider)
);

INSERT INTO user_identities (user_id, provider, provider_subject, avatar_url, email_verified)
SELECT id, 'google', google_sub, avatar_url, COALESCE(email_verified, false)
FROM users
WHERE google_sub IS NOT NULL
ON CONFLICT (provider, provider_subject) DO NOTHING;

DROP INDEX IF EXISTS idx_users_google_sub;

ALTER TABLE users
  DROP COLUMN IF EXISTS auth_provider,
  DROP COLUMN IF EXISTS google_sub,
  DROP COLUMN IF EXISTS avatar_url,
  DROP COLUMN IF EXISTS email_verified;
