-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE/fuzzy)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- optional use with expression indexes


-- =========================================================
-- ENUMS (idempotent)
-- =========================================================

-- class_delivery_mode_enum
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('offline','online','hybrid');
  END IF;
END$$;

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
-- TABLE: school_teachers  (JSONB sections + csst)
-- =========================================================
CREATE TABLE IF NOT EXISTS school_teachers (
  school_teacher_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope/relasi
  school_teacher_school_id       UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
  school_teacher_user_teacher_id UUID NOT NULL REFERENCES user_teachers(user_teacher_id) ON DELETE CASCADE,

  -- Identitas/kepegawaian
  school_teacher_code       VARCHAR(50),
  school_teacher_slug       VARCHAR(50),
  school_teacher_employment teacher_employment_enum,
  school_teacher_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  school_teacher_joined_at  DATE,
  school_teacher_left_at    DATE,
  CONSTRAINT mtj_left_after_join_chk CHECK (
    school_teacher_left_at IS NULL
    OR school_teacher_joined_at IS NULL
    OR school_teacher_left_at >= school_teacher_joined_at
  ),

  -- Verifikasi
  school_teacher_is_verified BOOLEAN   NOT NULL DEFAULT FALSE,
  school_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  school_teacher_is_public  BOOLEAN NOT NULL DEFAULT TRUE,
  school_teacher_notes      TEXT,

  -- Snapshot user_teachers
  school_teacher_user_teacher_name_snapshot          VARCHAR(80),
  school_teacher_user_teacher_avatar_url_snapshot    VARCHAR(255),
  school_teacher_user_teacher_whatsapp_url_snapshot  VARCHAR(50),
  school_teacher_user_teacher_title_prefix_snapshot  VARCHAR(20),
  school_teacher_user_teacher_title_suffix_snapshot  VARCHAR(30),
  school_teacher_user_teacher_gender_snapshot VARCHAR(20),

  -- JSONB: daftar section yang diampu (homeroom/assistant/teacher)
  school_teacher_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_mtj_sections_is_array CHECK (jsonb_typeof(school_teacher_sections) = 'array'),

  -- JSONB: daftar CSST (Section×Subject×Teacher)
  school_teacher_csst JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_mtj_csst_is_array CHECK (jsonb_typeof(school_teacher_csst) = 'array'),

  -- Audit
  school_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_deleted_at TIMESTAMPTZ
);

-- Pair unik (tenant-safe join ops)
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_teachers_id_tenant
  ON school_teachers (school_teacher_id, school_teacher_school_id);

-- =======================
-- INDEXING & OPTIMIZATION (school_teachers)
-- =======================

-- Unik: 1 user_teacher per school (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_school_user_alive
  ON school_teachers (school_teacher_school_id, school_teacher_user_teacher_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Unik SLUG per school (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_slug_alive_ci
  ON school_teachers (school_teacher_school_id, lower(school_teacher_slug))
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_slug IS NOT NULL;

-- Unik CODE per school (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_code_alive_ci
  ON school_teachers (school_teacher_school_id, lower(school_teacher_code))
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_code IS NOT NULL;

-- Filter umum
CREATE INDEX IF NOT EXISTS ix_mtj_tenant_active_public_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_active, school_teacher_is_public, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_verified_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_verified, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_employment_created
  ON school_teachers (school_teacher_school_id, school_teacher_employment, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- Akses cepat
CREATE INDEX IF NOT EXISTS idx_mtj_user_teacher_alive
  ON school_teachers (school_teacher_user_teacher_id)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_school_alive
  ON school_teachers (school_teacher_school_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Notes search (ILIKE/fuzzy)
CREATE INDEX IF NOT EXISTS gin_mtj_notes_trgm_alive
  ON school_teachers USING GIN (lower(school_teacher_notes) gin_trgm_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- JSONB containment (sections & csst)
CREATE INDEX IF NOT EXISTS gin_mtj_sections
  ON school_teachers USING GIN (school_teacher_sections jsonb_path_ops)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_mtj_csst
  ON school_teachers USING GIN (school_teacher_csst jsonb_path_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- Partial index: punya section aktif
CREATE INDEX IF NOT EXISTS ix_mtj_has_active_section_per_tenant
  ON school_teachers (school_teacher_school_id)
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_sections @? '$ ? (@.is_active == true)';

-- Partial index: punya csst aktif
CREATE INDEX IF NOT EXISTS ix_mtj_has_active_csst_per_tenant
  ON school_teachers (school_teacher_school_id)
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_csst @? '$ ? (@.is_active == true)';

-- Functional index: aktif_count (tanpa kolom turunan)
CREATE INDEX IF NOT EXISTS ix_mtj_sections_active_count_expr
  ON school_teachers (
    (jsonb_array_length(jsonb_path_query_array(school_teacher_sections, '$ ? (@.is_active == true)')))
  )
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_csst_active_count_expr
  ON school_teachers (
    (jsonb_array_length(jsonb_path_query_array(school_teacher_csst, '$ ? (@.is_active == true)')))
  )
  WHERE school_teacher_deleted_at IS NULL;

-- Pencarian nama guru dari snapshot (ILIKE/fuzzy)
CREATE INDEX IF NOT EXISTS gin_mtj_name_snap_trgm_alive
  ON school_teachers USING GIN (lower(school_teacher_user_teacher_name_snapshot) gin_trgm_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- Listing cepat per school + terbaru (fallback)
CREATE INDEX IF NOT EXISTS ix_mtj_school_created_desc
  ON school_teachers (school_teacher_school_id, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- BRIN time
CREATE INDEX IF NOT EXISTS brin_mtj_joined_at
  ON school_teachers USING BRIN (school_teacher_joined_at);
CREATE INDEX IF NOT EXISTS brin_mtj_created_at
  ON school_teachers USING BRIN (school_teacher_created_at);


-- =========================================================
-- TABLE: school_students (JSONB sections + school snapshot)
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

  -- Snapshot user_profiles
  school_student_user_profile_name_snapshot                VARCHAR(80),
  school_student_user_profile_avatar_url_snapshot          VARCHAR(255),
  school_student_user_profile_whatsapp_url_snapshot        VARCHAR(50),
  school_student_user_profile_parent_name_snapshot         VARCHAR(80),
  school_student_user_profile_parent_whatsapp_url_snapshot VARCHAR(50),
  school_student_user_profile_gender_snapshot VARCHAR(20),

  -- JSONB SECTIONS (dipelihara backend)
  -- contoh item:
  -- {
  --   "class_section_id": "uuid",
  --   "is_active": true, "from": "YYYY-MM-DD", "to": null,
  --   "class_section_name": "Tahfidz A",
  --   "class_section_slug": "tahfidz-a",
  --   "class_section_image_url": "https://...",
  --   "class_section_image_object_key": "..."
  -- }
  school_student_class_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_ms_sections_is_array CHECK (jsonb_typeof(school_student_class_sections) = 'array'),

  -- Audit
  school_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_deleted_at TIMESTAMPTZ
);

-- Pair unik (tenant-safe join ops)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_id_school
  ON school_students (school_student_id, school_student_school_id);

-- =======================
-- INDEXING & OPTIMIZATION (school_students)
-- =======================

-- Unik SLUG per school (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_slug_alive_ci
  ON school_students (school_student_school_id, lower(school_student_slug))
  WHERE school_student_deleted_at IS NULL;

-- Unik: 1 profile aktif per school (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_profile_per_school_live
  ON school_students (school_student_school_id, school_student_user_profile_id)
  WHERE school_student_deleted_at IS NULL
    AND school_student_status = 'active';

-- Unik CODE per school (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_code_alive_ci
  ON school_students (school_student_school_id, lower(school_student_code))
  WHERE school_student_deleted_at IS NULL
    AND school_student_code IS NOT NULL;

-- Lookups umum per tenant
CREATE INDEX IF NOT EXISTS ix_ms_tenant_status_created
  ON school_students (school_student_school_id, school_student_status, school_student_created_at DESC)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_school_alive
  ON school_students (school_student_school_id)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_profile_alive
  ON school_students (school_student_user_profile_id)
  WHERE school_student_deleted_at IS NULL;

-- Notes search
CREATE INDEX IF NOT EXISTS gin_ms_note_trgm_alive
  ON school_students USING GIN (lower(school_student_note) gin_trgm_ops)
  WHERE school_student_deleted_at IS NULL;

-- JSONB containment (sections)
CREATE INDEX IF NOT EXISTS gin_ms_sections
  ON school_students USING GIN (school_student_class_sections jsonb_path_ops)
  WHERE school_student_deleted_at IS NULL;

-- Partial index: punya section aktif
CREATE INDEX IF NOT EXISTS ix_ms_has_active_section_per_tenant
  ON school_students (school_student_school_id)
  WHERE school_student_deleted_at IS NULL
    AND school_student_class_sections @? '$ ? (@.is_active == true)';

-- Functional index: aktif_count (tanpa kolom turunan)
CREATE INDEX IF NOT EXISTS ix_ms_sections_active_count_expr
  ON school_students (
    (jsonb_array_length(jsonb_path_query_array(school_student_class_sections, '$ ? (@.is_active == true)')))
  )
  WHERE school_student_deleted_at IS NULL;

-- Pencarian nama dari profile snapshot (opsional)
CREATE INDEX IF NOT EXISTS gin_ms_name_snap_trgm_alive
  ON school_students USING GIN (lower(school_student_user_profile_name_snapshot) gin_trgm_ops)
  WHERE school_student_deleted_at IS NULL;

-- BRIN time
CREATE INDEX IF NOT EXISTS brin_ms_created_at
  ON school_students USING BRIN (school_student_created_at);

COMMIT;
