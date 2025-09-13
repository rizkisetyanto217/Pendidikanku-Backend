-- +migrate Up
BEGIN;

-- =========================================================
-- 1) USERS_PROFILE_FORMAL (fresh)
-- =========================================================
CREATE TABLE IF NOT EXISTS users_profile_formal (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  father_name    VARCHAR(50),
  father_phone   VARCHAR(20),
  mother_name    VARCHAR(50),
  mother_phone   VARCHAR(20),

  guardian       VARCHAR(50),
  guardian_phone VARCHAR(20),

  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ,
  deleted_at     TIMESTAMPTZ,

  CONSTRAINT uq_users_profile_formal_user UNIQUE (user_id)
);

-- Index (alive rows)
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_user_alive
  ON users_profile_formal(user_id) WHERE deleted_at IS NULL;

-- (opsional) exact-match phone lookups
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_father_phone
  ON users_profile_formal(father_phone) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_mother_phone
  ON users_profile_formal(mother_phone) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_guardian_phone
  ON users_profile_formal(guardian_phone) WHERE deleted_at IS NULL;

