-- =========================================================
-- UP: USERS_TEACHER (simple profile, fresh start)
-- =========================================================
BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Utility: updated_at trigger fn (aman berulang)
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Tabel inti
CREATE TABLE IF NOT EXISTS users_teacher (
  users_teacher_id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  users_teacher_user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Konten profil (sederhana)
  users_teacher_field           VARCHAR(80),
  users_teacher_short_bio       VARCHAR(300),
  users_teacher_greeting        TEXT,
  users_teacher_education       TEXT,
  users_teacher_activity        TEXT,
  users_teacher_experience_years SMALLINT,

  -- Metadata fleksibel
  users_teacher_specialties     JSONB,
  users_teacher_certificates    JSONB,
  users_teacher_links           JSONB,

  -- Status ringkas
  users_teacher_is_verified     BOOLEAN     NOT NULL DEFAULT FALSE,
  users_teacher_is_active       BOOLEAN     NOT NULL DEFAULT TRUE,

  -- Audit
  users_teacher_created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_teacher_updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_teacher_deleted_at      TIMESTAMPTZ,

  -- Satu profil per user
  CONSTRAINT uq_users_teacher_user UNIQUE (users_teacher_user_id)
);

-- Indexing & Search
CREATE INDEX IF NOT EXISTS idx_users_teacher_field_trgm
  ON users_teacher USING gin (users_teacher_field gin_trgm_ops)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_teacher_field_lower
  ON users_teacher (lower(users_teacher_field))
  WHERE users_teacher_deleted_at IS NULL;

ALTER TABLE users_teacher
  ADD COLUMN IF NOT EXISTS users_teacher_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(users_teacher_field,'')),     'A') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_short_bio,'')), 'B') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_education,'')), 'C')
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_users_teacher_search
  ON users_teacher USING gin (users_teacher_search);

CREATE INDEX IF NOT EXISTS idx_users_teacher_active
  ON users_teacher (users_teacher_is_active)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_users_teacher_created_at
  ON users_teacher USING brin (users_teacher_created_at);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_set_updated_at_users_teacher ON users_teacher;
CREATE TRIGGER trg_set_updated_at_users_teacher
BEFORE UPDATE ON users_teacher
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
