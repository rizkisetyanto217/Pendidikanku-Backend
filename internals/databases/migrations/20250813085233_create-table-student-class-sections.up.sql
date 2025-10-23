-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE & fuzzy)

-- =========================================================
-- TABEL: student_class_sections (gabungan enrolment + section)
-- =========================================================
CREATE TABLE IF NOT EXISTS student_class_sections (
  student_class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- identitas siswa (tenant-aware)
  student_class_section_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE RESTRICT,

  -- section (kelas paralel) & tenant
  student_class_section_section_id UUID NOT NULL,
  student_class_section_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  -- lifecycle enrolment
  student_class_section_status TEXT NOT NULL DEFAULT 'active'
    CHECK (student_class_section_status IN ('active','inactive','completed')),

  -- outcome (hasil akhir)
  student_class_section_result TEXT
    CHECK (student_class_section_result IN ('passed','failed')),

  -- snapshot biaya saat masuk section (agar tetap konsisten walau aturan berubah)
  student_class_section_fee_snapshot JSONB,

  -- Snapshot users_profile (per siswa saat enrol ke section)
  student_class_section_user_profile_name_snapshot                VARCHAR(80),
  student_class_section_user_profile_avatar_url_snapshot          VARCHAR(255),
  student_class_section_user_profile_whatsapp_url_snapshot        VARCHAR(50),
  student_class_section_user_profile_parent_name_snapshot         VARCHAR(80),
  student_class_section_user_profile_parent_whatsapp_url_snapshot VARCHAR(50),

  -- jejak waktu enrolment
  student_class_section_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  student_class_section_unassigned_at DATE,
  student_class_section_completed_at  TIMESTAMPTZ,

  -- audit
  student_class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_section_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_scsec_dates CHECK (
    student_class_section_unassigned_at IS NULL
    OR student_class_section_unassigned_at >= student_class_section_assigned_at
  ),
  CONSTRAINT chk_scsec_result_only_when_completed CHECK (
    (student_class_section_status = 'completed' AND student_class_section_result IS NOT NULL)
    OR (student_class_section_status <> 'completed' AND student_class_section_result IS NULL)
  ),
  CONSTRAINT chk_scsec_completed_at_when_completed CHECK (
    student_class_section_status <> 'completed'
    OR student_class_section_completed_at IS NOT NULL
  ),

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_scsec__section_masjid_pair
    FOREIGN KEY (student_class_section_section_id, student_class_section_masjid_id)
    REFERENCES class_sections (class_section_id, class_section_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================================================
-- INDEXES
-- =========================================================

-- unik: satu siswa hanya boleh aktif di satu section per masjid
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsec_active_per_student
  ON student_class_sections (student_class_section_masjid_student_id, student_class_section_masjid_id)
  WHERE student_class_section_deleted_at IS NULL
    AND student_class_section_status = 'active';

-- lookups umum
CREATE INDEX IF NOT EXISTS ix_scsec_tenant_student_created
  ON student_class_sections (student_class_section_masjid_id, student_class_section_masjid_student_id, student_class_section_created_at DESC)
  WHERE student_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_scsec_tenant_status_created
  ON student_class_sections (student_class_section_masjid_id, student_class_section_status, student_class_section_created_at DESC)
  WHERE student_class_section_deleted_at IS NULL;

-- JOIN cepat ke class_sections (komposit)
CREATE INDEX IF NOT EXISTS idx_scsec_section_masjid_alive
  ON student_class_sections (student_class_section_section_id, student_class_section_masjid_id)
  WHERE student_class_section_deleted_at IS NULL;

-- lookup spesifik
CREATE INDEX IF NOT EXISTS idx_scsec_section_alive
  ON student_class_sections (student_class_section_section_id)
  WHERE student_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsec_student_alive
  ON student_class_sections (student_class_section_masjid_student_id)
  WHERE student_class_section_deleted_at IS NULL;

-- BRIN untuk range waktu
CREATE INDEX IF NOT EXISTS brin_scsec_created_at
  ON student_class_sections USING BRIN (student_class_section_created_at);

CREATE INDEX IF NOT EXISTS brin_scsec_updated_at
  ON student_class_sections USING BRIN (student_class_section_updated_at);

COMMIT;
