-- +migrate Up
BEGIN;

-- =========================================
-- EXTENSIONS (idempotent)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;


-- =========================================
-- ENUMS (idempotent)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_enrollment_status') THEN
    CREATE TYPE class_enrollment_status AS ENUM (
      'initiated','pending_review','awaiting_payment',
      'accepted','waitlisted','rejected','canceled'
    );
  END IF;
END$$;

-- Pastikan academic_terms punya UNIQUE (id, school_id) untuk FK komposit
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_tenant') THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_tenant
      UNIQUE (academic_term_id, academic_term_school_id);
  END IF;
END$$;

-- =========================================
-- TABLE: student_class_enrollments (+ term snapshots)
-- =========================================
CREATE TABLE IF NOT EXISTS student_class_enrollments (
  student_class_enrollments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi identitas
  student_class_enrollments_school_id         UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE RESTRICT,

  student_class_enrollments_school_student_id UUID NOT NULL,
  student_class_enrollments_class_id          UUID NOT NULL,

  -- Tenant-safe FKs (komposit)
  CONSTRAINT fk_sce_student_same_school
    FOREIGN KEY (student_class_enrollments_school_student_id, student_class_enrollments_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_sce_class_same_school
    FOREIGN KEY (student_class_enrollments_class_id, student_class_enrollments_school_id)
    REFERENCES classes (class_id, class_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- status & biaya
  student_class_enrollments_status class_enrollment_status NOT NULL DEFAULT 'initiated',
  student_class_enrollments_total_due_idr NUMERIC(12,0) NOT NULL DEFAULT 0 CHECK (student_class_enrollments_total_due_idr >= 0),

  -- pembayaran (opsional)
  student_class_enrollments_payment_id UUID,
  student_class_enrollments_payment_snapshot JSONB,

  -- preferensi (opsional)
  student_class_enrollments_preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_sce_prefs_obj
    CHECK (jsonb_typeof(student_class_enrollments_preferences) = 'object'),

  -- ===== Snapshots (DIMINTA) =====
  -- Dari classes
  student_class_enrollments_class_name_snapshot VARCHAR(160),
  student_class_enrollments_class_slug_snapshot VARCHAR(160),

  -- ===== Denormalized TERM (diambil dari classes → class_term_*) =====
  student_class_enrollments_term_id                       UUID,
  student_class_enrollments_term_academic_year_snapshot   TEXT,
  student_class_enrollments_term_name_snapshot            TEXT,
  student_class_enrollments_term_slug_snapshot            TEXT,
  student_class_enrollments_term_angkatan_snapshot        INTEGER,

  -- FK komposit ke academic_terms (nullable; historis)
  CONSTRAINT fk_sce_term_same_school
    FOREIGN KEY (student_class_enrollments_term_id, student_class_enrollments_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- Dari school_students (snapshot identitas)
  student_class_enrollments_student_name_snapshot VARCHAR(80),
  student_class_enrollments_student_code_snapshot VARCHAR(50),
  student_class_enrollments_student_slug_snapshot VARCHAR(50),


  student_class_enrollments_class_section_id UUID,
  student_class_enrollments_class_section_name_snapshot VARCHAR(50),
  student_class_enrollments_class_section_slug_snapshot VARCHAR(50),

  -- jejak waktu (audit)
  student_class_enrollments_applied_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  student_class_enrollments_reviewed_at   TIMESTAMPTZ,
  student_class_enrollments_accepted_at   TIMESTAMPTZ,
  student_class_enrollments_waitlisted_at TIMESTAMPTZ,
  student_class_enrollments_rejected_at   TIMESTAMPTZ,
  student_class_enrollments_canceled_at   TIMESTAMPTZ,

  student_class_enrollments_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  student_class_enrollments_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  student_class_enrollments_deleted_at TIMESTAMPTZ,

  -- tenant-safe pair
  UNIQUE (student_class_enrollments_id, student_class_enrollments_school_id)
);



-- =========================================
-- FK komposit ke class_sections (tenant-safe)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_sce_class_section_same_school'
  ) THEN
    ALTER TABLE student_class_enrollments
      ADD CONSTRAINT fk_sce_class_section_same_school
      FOREIGN KEY (student_class_enrollments_class_section_id, student_class_enrollments_school_id)
      REFERENCES class_sections (class_section_id, class_section_school_id)
      ON UPDATE CASCADE
      ON DELETE SET NULL;
  END IF;
END$$;


-- ==========================
-- INDEXES
-- ==========================

-- Unik: satu siswa hanya boleh aktif di satu class (untuk status non-final tertentu)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sce_active_per_student_class
  ON student_class_enrollments (
    student_class_enrollments_school_student_id,
    student_class_enrollments_class_id
  )
  WHERE student_class_enrollments_deleted_at IS NULL
    AND student_class_enrollments_status IN ('initiated','pending_review','awaiting_payment','accepted','waitlisted');

-- Lookups umum
CREATE INDEX IF NOT EXISTS ix_sce_tenant_student_created
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_school_student_id,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sce_tenant_class_created
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_class_id,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sce_status_created_alive
  ON student_class_enrollments (
    student_class_enrollments_status,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_sce_prefs
  ON student_class_enrollments USING GIN (student_class_enrollments_preferences jsonb_path_ops);

-- Search cepat pada snapshot nama
CREATE INDEX IF NOT EXISTS gin_sce_class_name_snap_trgm_alive
  ON student_class_enrollments USING GIN (LOWER(student_class_enrollments_class_name_snapshot) gin_trgm_ops)
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_sce_student_name_snap_trgm_alive
  ON student_class_enrollments USING GIN (LOWER(student_class_enrollments_student_name_snapshot) gin_trgm_ops)
  WHERE student_class_enrollments_deleted_at IS NULL;

-- Filter per tenant & term (denormalized)
CREATE INDEX IF NOT EXISTS idx_sce_tenant_term_alive
  ON student_class_enrollments (student_class_enrollments_school_id, student_class_enrollments_term_id)
  WHERE student_class_enrollments_deleted_at IS NULL;

-- Fuzzy search (opsional) di nama term & year
CREATE INDEX IF NOT EXISTS gin_sce_term_name_trgm_alive
  ON student_class_enrollments USING GIN (LOWER(student_class_enrollments_term_name_snapshot) gin_trgm_ops)
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_sce_term_year_trgm_alive
  ON student_class_enrollments USING GIN (LOWER(student_class_enrollments_term_academic_year_snapshot) gin_trgm_ops)
  WHERE student_class_enrollments_deleted_at IS NULL;



-- =========================================
-- INDEXES untuk kolom baru
-- =========================================

-- 1) Lookup per tenant + section (buat filter "section ini isinya siapa saja")
CREATE INDEX IF NOT EXISTS idx_sce_tenant_section_alive
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_class_section_id
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

-- 2) Optional: cepat cari berdasarkan NAMA section (trgm, fuzzy search)
CREATE INDEX IF NOT EXISTS gin_sce_class_section_name_snap_trgm_alive
  ON student_class_enrollments USING GIN (
    LOWER(student_class_enrollments_class_section_name_snapshot) gin_trgm_ops
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

-- 3) Optional: index slug snapshot per tenant (buat join / lookup exact)
CREATE INDEX IF NOT EXISTS idx_sce_tenant_section_slug_snap_alive
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_class_section_slug_snapshot
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

COMMIT;
