-- +migrate Up
-- =========================================
-- UP Migration — Assessments (3 tabel, final)
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- 1) ASSESSMENT_TYPES (master types)
--    - tabel plural
--    - kolom singular (prefix assessment_type_)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_types (
  assessment_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_type_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  assessment_type_key  VARCHAR(32)  NOT NULL,  -- unik per school (uas, uts, tugas, ...)
  assessment_type_name VARCHAR(120) NOT NULL,

  assessment_type_weight_percent NUMERIC(5,2) NOT NULL DEFAULT 0
    CHECK (assessment_type_weight_percent >= 0 AND assessment_type_weight_percent <= 100),

  assessment_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  assessment_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_deleted_at TIMESTAMPTZ
);

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_id_tenant
  ON assessment_types (assessment_type_id, assessment_type_school_id);

-- Unik per school + key (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_key_per_school_alive
  ON assessment_types (assessment_type_school_id, LOWER(assessment_type_key))
  WHERE assessment_type_deleted_at IS NULL;

-- Listing aktif / filter umum
CREATE INDEX IF NOT EXISTS idx_assessment_types_school_active
  ON assessment_types (assessment_type_school_id, assessment_type_is_active)
  WHERE assessment_type_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_assessment_types_created_at
  ON assessment_types USING BRIN (assessment_type_created_at);



-- =========================================================
-- 0) TENANT-SAFE GUARD PADA CSST (unik pair id+tenant)
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_school
ON class_section_subject_teachers (
  class_section_subject_teacher_id,
  class_section_subject_teacher_school_id
);

-- =========================================================
-- 2) ASSESSMENTS — clean relation (ONLY TO CSST) + saklar session
-- =========================================================
-- =========================================================
-- ENUM: assessment_kind_enum (bentuk assessment)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'assessment_kind_enum') THEN
    CREATE TYPE assessment_kind_enum AS ENUM (
      'quiz',
      'assignment_upload',
      'offline',
      'survey'
    );
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS assessments (
  assessment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Hanya relasi ke CSST (tenant tidak dipaksa komposit di DB)
  assessment_class_section_subject_teacher_id UUID NULL,

  -- Tipe penilaian (kategori akademik, tenant-safe di-backend)
  assessment_type_id UUID,

  assessment_slug VARCHAR(160),

  assessment_title       VARCHAR(180) NOT NULL,
  assessment_description TEXT,

  -- Jadwal by date
  assessment_start_at     TIMESTAMPTZ,
  assessment_due_at       TIMESTAMPTZ,
  assessment_published_at TIMESTAMPTZ,
  assessment_closed_at    TIMESTAMPTZ,

  -- Pengaturan
  assessment_kind assessment_kind_enum NOT NULL DEFAULT 'quiz', -- bentuk: quiz / assignment_upload / offline / survey
  assessment_duration_minutes       INT,
  assessment_total_attempts_allowed INT NOT NULL DEFAULT 1,
  assessment_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessment_max_score >= 0 AND assessment_max_score <= 100),

  assessment_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  -- Audit
  assessment_created_by_teacher_id UUID,

  -- Snapshot
  assessment_csst_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  assessment_announce_session_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  assessment_collect_session_snapshot  JSONB NOT NULL DEFAULT '{}'::jsonb,

  -- Saklar masa depan (per session)
  assessment_submission_mode TEXT NOT NULL DEFAULT 'date'
    CHECK (assessment_submission_mode IN ('date','session')),
  assessment_announce_session_id UUID,  -- opsional
  assessment_collect_session_id  UUID,  -- opsional

  -- Timestamps
  assessment_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- 3) FK KE CSST (single-col saja)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_csst') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_csst
      FOREIGN KEY (assessment_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teacher_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- 4) FK KE assessment_types (single-col saja)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_type') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_type
      FOREIGN KEY (assessment_type_id)
      REFERENCES assessment_types(assessment_type_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- index bantu untuk join cepat ke assessment_types
CREATE INDEX IF NOT EXISTS idx_assessments_type
  ON assessments (assessment_type_id)
  WHERE assessment_deleted_at IS NULL;

-- =========================================================
-- 5) FK OPSIONAL KE class_attendance_sessions
-- =========================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_name = 'class_attendance_sessions')
     AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_announce_session') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_announce_session
      FOREIGN KEY (assessment_announce_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_name = 'class_attendance_sessions')
     AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_collect_session') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_collect_session
      FOREIGN KEY (assessment_collect_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- 6) INDEXES assessments
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_id_tenant
  ON assessments (assessment_id, assessment_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_slug_per_tenant_alive
  ON assessments (assessment_school_id, LOWER(assessment_slug))
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_assessments_slug_trgm_alive
  ON assessments USING GIN (LOWER(assessment_slug) gin_trgm_ops)
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_school_created_at
  ON assessments (assessment_school_id, assessment_created_at DESC)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments (assessment_class_section_subject_teacher_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_created_by_teacher
  ON assessments (assessment_created_by_teacher_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_submission_mode_alive
  ON assessments (assessment_school_id, assessment_submission_mode)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_announce_session_alive
  ON assessments (assessment_announce_session_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_collect_session_alive
  ON assessments (assessment_collect_session_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_kind_alive
  ON assessments (assessment_school_id, assessment_kind)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_assessments_created_at
  ON assessments USING BRIN (assessment_created_at);


-- =========================================================
-- 7) ASSESSMENT_URLS (selaras dgn announcement_urls)
--    - tabel plural
--    - kolom singular (prefix assessment_url_)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  assessment_url_school_id       UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,
  assessment_url_assessment_id   UUID NOT NULL
    REFERENCES assessments(assessment_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'image','video','attachment','link', dst.)
  assessment_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link (skema dua-slot + retensi)
  assessment_url                    TEXT,
  assessment_url_object_key         TEXT,
  assessment_url_old                TEXT,
  assessment_url_object_key_old     TEXT,
  assessment_url_delete_pending_until TIMESTAMPTZ,

  -- Tampilan
  assessment_url_label           VARCHAR(160),
  assessment_url_order           INT NOT NULL DEFAULT 0,
  assessment_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  assessment_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_deleted_at      TIMESTAMPTZ
);

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_id_tenant
  ON assessment_urls (assessment_url_id, assessment_url_school_id);

-- Lookup per assessment (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_owner_live
  ON assessment_urls (
    assessment_url_assessment_id,
    assessment_url_kind,
    assessment_url_is_primary DESC,
    assessment_url_order,
    assessment_url_created_at
  )
  WHERE assessment_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_school_live
  ON assessment_urls (assessment_url_school_id)
  WHERE assessment_url_deleted_at IS NULL;

-- Satu primary per (assessment, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_primary_per_kind_alive
  ON assessment_urls (assessment_url_assessment_id, assessment_url_kind)
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_is_primary = TRUE;

-- Anti-duplikat URL per assessment (live only; case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_assessment_url_alive
  ON assessment_urls (assessment_url_assessment_id, LOWER(assessment_url))
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url IS NOT NULL;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_purge_due
  ON assessment_urls (assessment_url_delete_pending_until)
  WHERE assessment_url_delete_pending_until IS NOT NULL
    AND (
      (assessment_url_deleted_at IS NULL  AND assessment_url_object_key_old IS NOT NULL) OR
      (assessment_url_deleted_at IS NOT NULL AND assessment_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_assessment_urls_label_trgm_live
  ON assessment_urls USING GIN (assessment_url_label gin_trgm_ops)
  WHERE assessment_url_deleted_at IS NULL;

COMMIT;
