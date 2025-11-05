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

-- =========================================
-- TABLE: student_class_enrollments
-- =========================================
CREATE TABLE IF NOT EXISTS student_class_enrollments (
  student_class_enrollments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi identitas
  student_class_enrollments_school_id         UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE RESTRICT,

  student_class_enrollments_school_student_id UUID NOT NULL,
  student_class_enrollments_class_id          UUID NOT NULL,

  -- Tenant-safe FKs (komposit)
  CONSTRAINT fk_student_class_enrollments_student_same_school
    FOREIGN KEY (student_class_enrollments_school_student_id, student_class_enrollments_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_student_class_enrollments_class_same_school
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
  CONSTRAINT ck_student_class_enrollments_prefs_obj
    CHECK (jsonb_typeof(student_class_enrollments_preferences) = 'object'),

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
-- INDEXES: student_class_enrollments
-- =========================================

-- Enrollment "aktif" per (student, class)
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_class_enrollments_active_per_student_class
  ON student_class_enrollments (
    student_class_enrollments_school_student_id,
    student_class_enrollments_class_id
  )
  WHERE student_class_enrollments_deleted_at IS NULL
    AND student_class_enrollments_status IN ('initiated','pending_review','awaiting_payment','accepted','waitlisted');

-- Lookups umum
CREATE INDEX IF NOT EXISTS ix_student_class_enrollments_tenant_student_created
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_school_student_id,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_student_class_enrollments_tenant_class_created
  ON student_class_enrollments (
    student_class_enrollments_school_id,
    student_class_enrollments_class_id,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_student_class_enrollments_status_created
  ON student_class_enrollments (
    student_class_enrollments_status,
    student_class_enrollments_created_at DESC
  )
  WHERE student_class_enrollments_deleted_at IS NULL;

-- JSONB prefs
CREATE INDEX IF NOT EXISTS gin_student_class_enrollments_prefs
  ON student_class_enrollments USING GIN (student_class_enrollments_preferences jsonb_path_ops);

COMMIT;
