BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram (opsional)

-- =========================================
-- TABLE: user_teachers (fresh)
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

  -- Metadata fleksibel
  user_teacher_specialties      JSONB,
  user_teacher_certificates     JSONB,
  user_teacher_whatsapp_url    VARCHAR(20),

    -- Avatar (single file, 2-slot + retensi 30 hari)
  user_teacher_avatar_url                   TEXT,
  user_teacher_avatar_object_key            TEXT,
  user_teacher_avatar_url_old               TEXT,
  user_teacher_avatar_object_key_old        TEXT,
  user_teacher_avatar_delete_pending_until  TIMESTAMPTZ,

  user_teacher_title_prefix       VARCHAR(60),
  user_teacher_title_suffix       VARCHAR(60),

  -- Status
  user_teacher_is_verified      BOOLEAN     NOT NULL DEFAULT FALSE,
  user_teacher_is_active        BOOLEAN     NOT NULL DEFAULT TRUE,

  -- Audit
  user_teacher_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_teacher_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_teacher_deleted_at       TIMESTAMPTZ,

  -- Satu profil per user
  CONSTRAINT uq_user_teachers_user UNIQUE (user_teacher_user_id),

  -- Guards sederhana
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
-- SEARCH COLUMN (GENERATED; tanpa trigger)
-- =========================================
ALTER TABLE user_teachers
  ADD COLUMN IF NOT EXISTS user_teacher_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_teacher_field,'')),            'A') ||
    setweight(to_tsvector('simple', coalesce(user_teacher_short_bio,'')),        'B') ||
    setweight(to_tsvector('simple', coalesce(user_teacher_education,'')),        'C') ||
    setweight(to_tsvector('simple', coalesce(user_teacher_activity,'')),         'C') ||
    setweight(to_tsvector('simple', coalesce(user_teacher_greeting,'')),         'D') ||
    -- ikutkan snapshot nama agar bisa dicari bebas
    setweight(to_tsvector('simple', coalesce(user_teacher_full_name_snapshot,'')),'A') ||
    setweight(to_tsvector('simple', coalesce(user_teacher_user_name_snapshot,'')),'A')
  ) STORED;

-- =========================================
-- INDEXES
-- =========================================

-- Pencarian & lookup
CREATE INDEX IF NOT EXISTS idx_user_teachers_field_trgm
  ON user_teachers USING gin (user_teacher_field gin_trgm_ops)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_teachers_field_lower
  ON user_teachers (lower(user_teacher_field))
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_teachers_search
  ON user_teachers USING gin (user_teacher_search);

-- Snapshot name lookup (disarankan)
CREATE INDEX IF NOT EXISTS idx_ut_full_name_snap_lower
  ON user_teachers (lower(user_teacher_full_name_snapshot))
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ut_user_name_snap_lower
  ON user_teachers (lower(user_teacher_user_name_snapshot))
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ut_full_name_snap_trgm
  ON user_teachers USING gin (user_teacher_full_name_snapshot gin_trgm_ops)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ut_user_name_snap_trgm
  ON user_teachers USING gin (user_teacher_user_name_snapshot gin_trgm_ops)
  WHERE user_teacher_deleted_at IS NULL;

-- Listing cepat (baris hidup)
CREATE INDEX IF NOT EXISTS idx_user_teachers_active
  ON user_teachers (user_teacher_is_active)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_teachers_active_verified_created
  ON user_teachers (user_teacher_is_active, user_teacher_is_verified, user_teacher_created_at DESC)
  WHERE user_teacher_deleted_at IS NULL;

-- JSONB (aktif bila difilter berdasarkan tag/isi)
CREATE INDEX IF NOT EXISTS gin_user_teachers_specialties
  ON user_teachers USING gin (user_teacher_specialties jsonb_path_ops)
  WHERE user_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_user_teachers_certificates
  ON user_teachers USING gin (user_teacher_certificates jsonb_path_ops)
  WHERE user_teacher_deleted_at IS NULL;

-- Arsip waktu (ringan)
CREATE INDEX IF NOT EXISTS brin_user_teachers_created_at
  ON user_teachers USING brin (user_teacher_created_at);

COMMIT;
