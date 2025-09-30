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
-- TABLE: masjid_teachers  (JSONB sections + csst + masjid snapshot)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teacher_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope/relasi
  masjid_teacher_masjid_id       UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teacher_user_teacher_id UUID NOT NULL REFERENCES user_teachers(user_teacher_id) ON DELETE CASCADE,

  -- Identitas/kepegawaian
  masjid_teacher_code       VARCHAR(50),
  masjid_teacher_slug       VARCHAR(50),
  masjid_teacher_employment teacher_employment_enum,
  masjid_teacher_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  masjid_teacher_joined_at  DATE,
  masjid_teacher_left_at    DATE,
  CONSTRAINT mtj_left_after_join_chk CHECK (
    masjid_teacher_left_at IS NULL
    OR masjid_teacher_joined_at IS NULL
    OR masjid_teacher_left_at >= masjid_teacher_joined_at
  ),

  -- Verifikasi
  masjid_teacher_is_verified BOOLEAN   NOT NULL DEFAULT FALSE,
  masjid_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  masjid_teacher_is_public  BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_teacher_notes      TEXT,

  -- Snapshot user_teachers
  masjid_teacher_user_teacher_name_snapshot          VARCHAR(80),
  masjid_teacher_user_teacher_avatar_url_snapshot    VARCHAR(255),
  masjid_teacher_user_teacher_whatsapp_url_snapshot  VARCHAR(50),
  masjid_teacher_user_teacher_title_prefix_snapshot  VARCHAR(20),
  masjid_teacher_user_teacher_title_suffix_snapshot  VARCHAR(30),

  -- MASJID SNAPSHOT (untuk render cepat /me)
  masjid_teacher_masjid_name_snapshot      VARCHAR(100),
  masjid_teacher_masjid_slug_snapshot      VARCHAR(100),
  masjid_teacher_masjid_logo_url_snapshot  TEXT,

  -- JSONB: daftar section yang diampu (homeroom/assistant/teacher) — data diisi backend
  -- contoh item:
  -- {
  --   "class_section_id": "uuid",
  --   "role": "homeroom|teacher|assistant",
  --   "is_active": true,
  --   "from": "YYYY-MM-DD", "to": "YYYY-MM-DD",
  --   "class_section_name": "Tahfidz A",
  --   "class_section_slug": "tahfidz-a",
  --   "class_section_image_url": "https://...",
  --   "class_section_image_object_key": "..."
  -- }
  masjid_teacher_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_mtj_sections_is_array CHECK (jsonb_typeof(masjid_teacher_sections) = 'array'),

  -- JSONB: daftar CSST (Section×Subject×Teacher) — grup guru mapel
  -- contoh item minimal:
  -- {
  --   "csst_id": "uuid",
  --   "is_active": true,
  --   "from": "YYYY-MM-DD", "to": null,
  --   "subject_name": "Fiqih",
  --   "subject_slug": "fiqih",
  --   "class_section_id": "uuid",
  --   "class_section_name": "Tahfidz A",
  --   "class_section_slug": "tahfidz-a"
  -- }
  masjid_teacher_csst JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_mtj_csst_is_array CHECK (jsonb_typeof(masjid_teacher_csst) = 'array'),

  -- Audit
  masjid_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_deleted_at TIMESTAMPTZ
);

-- Pair unik (tenant-safe join ops)
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_teachers_id_tenant
  ON masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id);

-- =======================
-- INDEXING & OPTIMIZATION (masjid_teachers)
-- =======================

-- Unik: 1 user_teacher per masjid (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_masjid_user_alive
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_teacher_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Unik SLUG per masjid (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_slug_alive_ci
  ON masjid_teachers (masjid_teacher_masjid_id, lower(masjid_teacher_slug))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_slug IS NOT NULL;

-- Unik CODE per masjid (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_code_alive_ci
  ON masjid_teachers (masjid_teacher_masjid_id, lower(masjid_teacher_code))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_code IS NOT NULL;

-- Filter umum
CREATE INDEX IF NOT EXISTS ix_mtj_tenant_active_public_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_active, masjid_teacher_is_public, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_verified_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_verified, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_employment_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_employment, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Akses cepat
CREATE INDEX IF NOT EXISTS idx_mtj_user_teacher_alive
  ON masjid_teachers (masjid_teacher_user_teacher_id)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_masjid_alive
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Notes search (ILIKE/fuzzy)
CREATE INDEX IF NOT EXISTS gin_mtj_notes_trgm_alive
  ON masjid_teachers USING GIN (lower(masjid_teacher_notes) gin_trgm_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

-- JSONB containment (sections & csst)
CREATE INDEX IF NOT EXISTS gin_mtj_sections
  ON masjid_teachers USING GIN (masjid_teacher_sections jsonb_path_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_mtj_csst
  ON masjid_teachers USING GIN (masjid_teacher_csst jsonb_path_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Partial index: punya section aktif
CREATE INDEX IF NOT EXISTS ix_mtj_has_active_section_per_tenant
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_sections @? '$ ? (@.is_active == true)';

-- Partial index: punya csst aktif
CREATE INDEX IF NOT EXISTS ix_mtj_has_active_csst_per_tenant
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_csst @? '$ ? (@.is_active == true)';

-- Functional index: aktif_count (tanpa kolom turunan)
CREATE INDEX IF NOT EXISTS ix_mtj_sections_active_count_expr
  ON masjid_teachers (
    (jsonb_array_length(jsonb_path_query_array(masjid_teacher_sections, '$ ? (@.is_active == true)')))
  )
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_csst_active_count_expr
  ON masjid_teachers (
    (jsonb_array_length(jsonb_path_query_array(masjid_teacher_csst, '$ ? (@.is_active == true)')))
  )
  WHERE masjid_teacher_deleted_at IS NULL;

-- Pencarian nama guru dari snapshot (ILIKE/fuzzy)
CREATE INDEX IF NOT EXISTS gin_mtj_name_snap_trgm_alive
  ON masjid_teachers USING GIN (lower(masjid_teacher_user_teacher_name_snapshot) gin_trgm_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Filter by nama/slug masjid dari snapshot (untuk /me)
CREATE INDEX IF NOT EXISTS ix_mtj_masjid_name_snap_ci_alive
  ON masjid_teachers (lower(masjid_teacher_masjid_name_snapshot))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_masjid_name_snapshot IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_masjid_slug_snap_ci_alive
  ON masjid_teachers (lower(masjid_teacher_masjid_slug_snapshot))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_masjid_slug_snapshot IS NOT NULL;

-- Listing cepat per masjid + terbaru (fallback)
CREATE INDEX IF NOT EXISTS ix_mtj_masjid_created_desc
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

-- BRIN time
CREATE INDEX IF NOT EXISTS brin_mtj_joined_at
  ON masjid_teachers USING BRIN (masjid_teacher_joined_at);
CREATE INDEX IF NOT EXISTS brin_mtj_created_at
  ON masjid_teachers USING BRIN (masjid_teacher_created_at);



-- =========================================================
-- TABLE: masjid_students (JSONB sections + masjid snapshot)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_students (
  masjid_student_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_student_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  masjid_student_user_profile_id UUID NOT NULL
    REFERENCES users_profile(users_profile_id) ON DELETE CASCADE,

  masjid_student_slug VARCHAR(50) NOT NULL,
  masjid_student_code VARCHAR(50),

  masjid_student_status TEXT NOT NULL DEFAULT 'active'
    CHECK (masjid_student_status IN ('active','inactive','alumni')),

  -- Operasional
  masjid_student_joined_at TIMESTAMPTZ,
  masjid_student_left_at   TIMESTAMPTZ,

  -- Catatan
  masjid_student_note TEXT,

  -- Snapshot users_profile
  masjid_student_user_profile_name_snapshot                VARCHAR(80),
  masjid_student_user_profile_avatar_url_snapshot          VARCHAR(255),
  masjid_student_user_profile_whatsapp_url_snapshot        VARCHAR(50),
  masjid_student_user_profile_parent_name_snapshot         VARCHAR(80),
  masjid_student_user_profile_parent_whatsapp_url_snapshot VARCHAR(50),

  -- MASJID SNAPSHOT (untuk render cepat /me)
  masjid_student_masjid_name_snapshot      VARCHAR(100),
  masjid_student_masjid_slug_snapshot      VARCHAR(100),
  masjid_student_masjid_logo_url_snapshot  TEXT,

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
  masjid_student_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_ms_sections_is_array CHECK (jsonb_typeof(masjid_student_sections) = 'array'),

  -- Audit
  masjid_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_deleted_at TIMESTAMPTZ
);

-- Pair unik (tenant-safe join ops)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_id_masjid
  ON masjid_students (masjid_student_id, masjid_student_masjid_id);

-- =======================
-- INDEXING & OPTIMIZATION (masjid_students)
-- =======================

-- Unik SLUG per masjid (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_slug_alive_ci
  ON masjid_students (masjid_student_masjid_id, lower(masjid_student_slug))
  WHERE masjid_student_deleted_at IS NULL;

-- Unik: 1 profile aktif per masjid (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_profile_per_masjid_live
  ON masjid_students (masjid_student_masjid_id, masjid_student_user_profile_id)
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_status = 'active';

-- Unik CODE per masjid (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_code_alive_ci
  ON masjid_students (masjid_student_masjid_id, lower(masjid_student_code))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_code IS NOT NULL;

-- Lookups umum per tenant
CREATE INDEX IF NOT EXISTS ix_ms_tenant_status_created
  ON masjid_students (masjid_student_masjid_id, masjid_student_status, masjid_student_created_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_masjid_alive
  ON masjid_students (masjid_student_masjid_id)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_profile_alive
  ON masjid_students (masjid_student_user_profile_id)
  WHERE masjid_student_deleted_at IS NULL;

-- Notes search
CREATE INDEX IF NOT EXISTS gin_ms_note_trgm_alive
  ON masjid_students USING GIN (lower(masjid_student_note) gin_trgm_ops)
  WHERE masjid_student_deleted_at IS NULL;

-- JSONB containment (sections)
CREATE INDEX IF NOT EXISTS gin_ms_sections
  ON masjid_students USING GIN (masjid_student_sections jsonb_path_ops)
  WHERE masjid_student_deleted_at IS NULL;

-- Partial index: punya section aktif
CREATE INDEX IF NOT EXISTS ix_ms_has_active_section_per_tenant
  ON masjid_students (masjid_student_masjid_id)
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_sections @? '$ ? (@.is_active == true)';

-- Functional index: aktif_count (tanpa kolom turunan)
CREATE INDEX IF NOT EXISTS ix_ms_sections_active_count_expr
  ON masjid_students (
    (jsonb_array_length(jsonb_path_query_array(masjid_student_sections, '$ ? (@.is_active == true)')))
  )
  WHERE masjid_student_deleted_at IS NULL;

-- Pencarian nama dari profile snapshot (opsional)
CREATE INDEX IF NOT EXISTS gin_ms_name_snap_trgm_alive
  ON masjid_students USING GIN (lower(masjid_student_user_profile_name_snapshot) gin_trgm_ops)
  WHERE masjid_student_deleted_at IS NULL;

-- Filter by masjid snapshot
CREATE INDEX IF NOT EXISTS ix_ms_masjid_name_snap_ci_alive
  ON masjid_students (lower(masjid_student_masjid_name_snapshot))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_masjid_name_snapshot IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_ms_masjid_slug_snap_ci_alive
  ON masjid_students (lower(masjid_student_masjid_slug_snapshot))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_masjid_slug_snapshot IS NOT NULL;

-- BRIN time
CREATE INDEX IF NOT EXISTS brin_ms_created_at
  ON masjid_students USING BRIN (masjid_student_created_at);


