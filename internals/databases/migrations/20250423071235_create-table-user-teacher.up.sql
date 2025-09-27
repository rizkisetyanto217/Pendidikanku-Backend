BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram ops

-- =========================================
-- TABLE: user_teachers
-- =========================================
CREATE TABLE IF NOT EXISTS user_teachers (
  user_teacher_id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_teacher_user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Profil ringkas
  user_teacher_name             VARCHAR(80) NOT NULL,
  user_teacher_field            VARCHAR(80),
  user_teacher_short_bio        VARCHAR(300),
  user_teacher_long_bio         TEXT,
  user_teacher_greeting         TEXT,
  user_teacher_education        TEXT,
  user_teacher_activity         TEXT,
  user_teacher_experience_years SMALLINT,

  -- Demografis (opsional)
  user_teacher_gender           VARCHAR(10),
  user_teacher_location         VARCHAR(100),
  user_teacher_city             VARCHAR(100),

  -- Metadata fleksibel
  user_teacher_specialties      JSONB,
  user_teacher_certificates     JSONB,

  -- Sosial media (opsional)
  user_teacher_instagram_url     TEXT,
  user_teacher_whatsapp_url      TEXT,
  user_teacher_youtube_url       TEXT,
  user_teacher_linkedin_url      TEXT,
  user_teacher_github_url        TEXT,
  user_teacher_telegram_username VARCHAR(50),

  -- Avatar (single file, 2-slot + retensi 30 hari)
  user_teacher_avatar_url                   TEXT,
  user_teacher_avatar_object_key            TEXT,
  user_teacher_avatar_url_old               TEXT,
  user_teacher_avatar_object_key_old        TEXT,
  user_teacher_avatar_delete_pending_until  TIMESTAMPTZ,

  -- Title
  user_teacher_title_prefix     VARCHAR(60),
  user_teacher_title_suffix     VARCHAR(60),

  -- Status
  user_teacher_is_verified      BOOLEAN     NOT NULL DEFAULT FALSE,
  user_teacher_is_active        BOOLEAN     NOT NULL DEFAULT TRUE,

  -- Audit
  user_teacher_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_teacher_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_teacher_deleted_at       TIMESTAMPTZ,

  -- Unik per user
  CONSTRAINT uq_user_teachers_user UNIQUE (user_teacher_user_id),

  -- Guards
  CONSTRAINT ck_ut_exp_years_range
    CHECK (user_teacher_experience_years IS NULL
           OR user_teacher_experience_years BETWEEN 0 AND 80),
  CONSTRAINT ck_ut_specialties_type
    CHECK (user_teacher_specialties IS NULL
           OR jsonb_typeof(user_teacher_specialties) = 'array'),
  CONSTRAINT ck_ut_certificates_type
    CHECK (user_teacher_certificates IS NULL
           OR jsonb_typeof(user_teacher_certificates) = 'array')
);

-- =========================================
-- INDEXES (SEARCH HANYA DI NAME)
-- =========================================
-- ILIKE cepat untuk name
CREATE INDEX IF NOT EXISTS idx_ut_name_trgm
  ON user_teachers USING gin (user_teacher_name gin_trgm_ops)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ut_name_lower
  ON user_teachers (lower(user_teacher_name))
  WHERE user_teacher_deleted_at IS NULL;

-- Status & waktu
CREATE INDEX IF NOT EXISTS idx_user_teachers_active
  ON user_teachers (user_teacher_is_active)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_teachers_active_verified_created
  ON user_teachers (user_teacher_is_active, user_teacher_is_verified, user_teacher_created_at DESC)
  WHERE user_teacher_deleted_at IS NULL;

-- JSONB (opsional; untuk filter tag/isi)
CREATE INDEX IF NOT EXISTS gin_user_teachers_specialties
  ON user_teachers USING gin (user_teacher_specialties jsonb_path_ops)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_user_teachers_certificates
  ON user_teachers USING gin (user_teacher_certificates jsonb_path_ops)
  WHERE user_teacher_deleted_at IS NULL;

-- Arsip waktu
CREATE INDEX IF NOT EXISTS brin_user_teachers_created_at
  ON user_teachers USING brin (user_teacher_created_at);

COMMIT;
