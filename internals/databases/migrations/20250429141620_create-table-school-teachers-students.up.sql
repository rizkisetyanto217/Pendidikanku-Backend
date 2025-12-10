-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;



-- =========================================================
-- ENUMS (idempotent)
-- =========================================================

-- teacher_employment_enum
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'teacher_employment_enum') THEN
    CREATE TYPE teacher_employment_enum AS ENUM (
      'tetap','kontrak','paruh_waktu','magang','honorer','relawan','tamu'
    );
  END IF;
END$$;



-- =========================================================
-- TABLE: school_teachers (CLEAN VERSION)
-- =========================================================
CREATE TABLE IF NOT EXISTS school_teachers (
  school_teacher_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope / tenant
  school_teacher_school_id       UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
  school_teacher_user_teacher_id UUID NOT NULL REFERENCES user_teachers(user_teacher_id) ON DELETE CASCADE,

  -- Identitas guru
  school_teacher_code        VARCHAR(50),
  school_teacher_slug        VARCHAR(50),
  school_teacher_employment  teacher_employment_enum,
  school_teacher_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  school_teacher_joined_at   DATE,
  school_teacher_left_at     DATE,
  CONSTRAINT ck_teacher_left_after_join CHECK (
    school_teacher_left_at IS NULL
    OR school_teacher_joined_at IS NULL
    OR school_teacher_left_at >= school_teacher_joined_at
  ),

  -- Verifikasi
  school_teacher_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  school_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  school_teacher_is_public   BOOLEAN NOT NULL DEFAULT TRUE,
  school_teacher_notes       TEXT,

  -- Snapshot ke user_teachers
  school_teacher_user_teacher_full_name_cache     VARCHAR(80),
  school_teacher_user_teacher_avatar_url_cache    VARCHAR(255),
  school_teacher_user_teacher_whatsapp_url_cache  VARCHAR(50),
  school_teacher_user_teacher_title_prefix_cache  VARCHAR(20),
  school_teacher_user_teacher_title_suffix_cache  VARCHAR(30),
  school_teacher_user_teacher_gender_cache        VARCHAR(20),

  -- Audit
  school_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_deleted_at TIMESTAMPTZ
);



-- =========================================================
-- INDEXES
-- =========================================================

-- Pair unik tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_teachers_id_tenant
  ON school_teachers (school_teacher_id, school_teacher_school_id);

-- Unik: 1 user_teacher per school (alive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_teacher_user_per_school_alive
  ON school_teachers (school_teacher_school_id, school_teacher_user_teacher_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Unik SLUG per school (CI; alive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_teacher_slug_alive_ci
  ON school_teachers (school_teacher_school_id, lower(school_teacher_slug))
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_slug IS NOT NULL;

-- Unik CODE per school (CI; alive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_teacher_code_alive_ci
  ON school_teachers (school_teacher_school_id, lower(school_teacher_code))
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_code IS NOT NULL;

-- Listing umum
CREATE INDEX IF NOT EXISTS ix_teacher_tenant_active_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_active, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- Verified listing
CREATE INDEX IF NOT EXISTS ix_teacher_verified_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_verified, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- Employment filter
CREATE INDEX IF NOT EXISTS ix_teacher_employment_created
  ON school_teachers (school_teacher_school_id, school_teacher_employment, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- Quick lookup
CREATE INDEX IF NOT EXISTS idx_teacher_user_alive
  ON school_teachers (school_teacher_user_teacher_id)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_teacher_school_alive
  ON school_teachers (school_teacher_school_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Notes search
CREATE INDEX IF NOT EXISTS gin_teacher_notes_trgm
  ON school_teachers USING GIN (lower(school_teacher_notes) gin_trgm_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- Name search via snapshot
CREATE INDEX IF NOT EXISTS gin_teacher_name_trgm
  ON school_teachers USING GIN (lower(school_teacher_user_teacher_full_name_cache) gin_trgm_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- BRIN time
CREATE INDEX IF NOT EXISTS brin_teacher_joined_at
  ON school_teachers USING BRIN (school_teacher_joined_at);

CREATE INDEX IF NOT EXISTS brin_teacher_created_at
  ON school_teachers USING BRIN (school_teacher_created_at);



-- =========================================================
-- TABLE: school_students (CLEAN VERSION)
-- =========================================================
CREATE TABLE IF NOT EXISTS school_students (
  school_student_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  school_student_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  school_student_user_profile_id UUID NOT NULL
    REFERENCES user_profiles(user_profile_id) ON DELETE CASCADE,

  school_student_slug VARCHAR(50) NOT NULL,
  school_student_code VARCHAR(50),

  school_student_status TEXT NOT NULL DEFAULT 'active'
    CHECK (school_student_status IN ('active','inactive','alumni')),

  -- Operasional
  school_student_joined_at TIMESTAMPTZ,
  school_student_left_at   TIMESTAMPTZ,
  school_student_needs_class_sections BOOLEAN NOT NULL DEFAULT FALSE,

  -- Catatan
  school_student_note TEXT,

  -- Snapshots user_profiles
  school_student_user_profile_name_cache                VARCHAR(80),
  school_student_user_profile_avatar_url_cache          VARCHAR(255),
  school_student_user_profile_whatsapp_url_cache        VARCHAR(50),
  school_student_user_profile_parent_name_cache         VARCHAR(80),
  school_student_user_profile_parent_whatsapp_url_cache VARCHAR(50),
  school_student_user_profile_gender_cache              VARCHAR(20),

  -- Audit
  school_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_deleted_at TIMESTAMPTZ
);


-- =========================================================
-- UNIQUE INDEXES (tenant safe)
-- =========================================================

CREATE UNIQUE INDEX IF NOT EXISTS uq_student_id_tenant
  ON school_students (school_student_id, school_student_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS ux_student_slug_alive_ci
  ON school_students (school_student_school_id, lower(school_student_slug))
  WHERE school_student_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_student_profile_per_school_alive
  ON school_students (school_student_school_id, school_student_user_profile_id)
  WHERE school_student_deleted_at IS NULL
    AND school_student_status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS ux_student_code_alive_ci
  ON school_students (school_student_school_id, lower(school_student_code))
  WHERE school_student_deleted_at IS NULL
    AND school_student_code IS NOT NULL;


-- =========================================================
-- GENERAL FILTERS
-- =========================================================

CREATE INDEX IF NOT EXISTS ix_student_tenant_status_created
  ON school_students (school_student_school_id, school_student_status, school_student_created_at DESC)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_school_alive
  ON school_students (school_student_school_id)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_profile_alive
  ON school_students (school_student_user_profile_id)
  WHERE school_student_deleted_at IS NULL;


-- =========================================================
-- NOTE SEARCH (TRGM)
-- =========================================================

CREATE INDEX IF NOT EXISTS gin_student_note_trgm
  ON school_students USING GIN (lower(school_student_note) gin_trgm_ops)
  WHERE school_student_deleted_at IS NULL;


-- =========================================================
-- NAME SEARCH TRGM
-- =========================================================

CREATE INDEX IF NOT EXISTS gin_student_name_snap_trgm
  ON school_students USING GIN (lower(school_student_user_profile_name_cache) gin_trgm_ops)
  WHERE school_student_deleted_at IS NULL;


-- =========================================================
-- BRIN TIME
-- =========================================================

CREATE INDEX IF NOT EXISTS brin_student_created_at
  ON school_students USING BRIN (school_student_created_at);


COMMIT;
