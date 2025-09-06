BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram (untuk index opsional di bawah)


-- =========================================
-- CREATE: user_teachers (fresh)
-- =========================================
CREATE TABLE user_teachers (
  users_teacher_id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  users_teacher_user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Profil ringkas
  users_teacher_field            VARCHAR(80),
  users_teacher_short_bio        VARCHAR(300),
  users_teacher_greeting         TEXT,
  users_teacher_education        TEXT,
  users_teacher_activity         TEXT,
  users_teacher_experience_years SMALLINT,

  -- Metadata fleksibel
  users_teacher_specialties      JSONB,
  users_teacher_certificates     JSONB,
  users_teacher_links            JSONB,

  -- Status
  users_teacher_is_verified      BOOLEAN     NOT NULL DEFAULT FALSE,
  users_teacher_is_active        BOOLEAN     NOT NULL DEFAULT TRUE,

  -- Audit
  users_teacher_created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_teacher_updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_teacher_deleted_at       TIMESTAMPTZ,

  -- Satu profil per user
  CONSTRAINT uq_user_teachers_user UNIQUE (users_teacher_user_id),

  -- Guards sederhana
  CONSTRAINT ck_ut_exp_years_range
    CHECK (users_teacher_experience_years IS NULL
           OR users_teacher_experience_years BETWEEN 0 AND 80),
  CONSTRAINT ck_ut_specialties_type
    CHECK (users_teacher_specialties IS NULL
           OR jsonb_typeof(users_teacher_specialties) = 'array'),
  CONSTRAINT ck_ut_certificates_type
    CHECK (users_teacher_certificates IS NULL
           OR jsonb_typeof(users_teacher_certificates) = 'array'),
  CONSTRAINT ck_ut_links_type
    CHECK (users_teacher_links IS NULL
           OR jsonb_typeof(users_teacher_links) = 'object')
);

-- =========================================
-- SEARCH COLUMN (GENERATED, tanpa trigger)
-- =========================================
ALTER TABLE user_teachers
  ADD COLUMN users_teacher_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(users_teacher_field,'')),     'A') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_short_bio,'')), 'B') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_education,'')), 'C') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_activity,'')),  'C') ||
    setweight(to_tsvector('simple', coalesce(users_teacher_greeting,'')),  'D')
  ) STORED;

-- =========================================
-- INDEXES (baru, bersih)
-- =========================================
-- Pencarian & lookup
CREATE INDEX idx_user_teachers_field_trgm
  ON user_teachers USING gin (users_teacher_field gin_trgm_ops)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX idx_user_teachers_field_lower
  ON user_teachers (lower(users_teacher_field))
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX idx_user_teachers_search
  ON user_teachers USING gin (users_teacher_search);

-- Listing cepat (baris hidup)
CREATE INDEX idx_user_teachers_active
  ON user_teachers (users_teacher_is_active)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX ix_user_teachers_active_verified_created
  ON user_teachers (users_teacher_is_active, users_teacher_is_verified, users_teacher_created_at DESC)
  WHERE users_teacher_deleted_at IS NULL;

-- JSONB (aktif bila difilter berdasarkan tag/isi)
CREATE INDEX gin_user_teachers_specialties
  ON user_teachers USING gin (users_teacher_specialties jsonb_path_ops)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX gin_user_teachers_certificates
  ON user_teachers USING gin (users_teacher_certificates jsonb_path_ops)
  WHERE users_teacher_deleted_at IS NULL;

CREATE INDEX gin_user_teachers_links
  ON user_teachers USING gin (users_teacher_links jsonb_path_ops)
  WHERE users_teacher_deleted_at IS NULL;

-- Arsip waktu (ringan)
CREATE INDEX brin_user_teachers_created_at
  ON user_teachers USING brin (users_teacher_created_at);

COMMIT;
