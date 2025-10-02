BEGIN;

-- safe to repeat
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- TABEL: user_class_sections (gabungan enrolment + section)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  user_class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- identitas siswa (tenant-aware)
  user_class_section_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE RESTRICT,

  -- section (kelas paralel) & tenant
  user_class_section_section_id UUID NOT NULL,
  user_class_section_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  -- lifecycle enrolment
  user_class_section_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_class_section_status IN ('active','inactive','completed')),

  -- outcome (hasil akhir)
  user_class_section_result TEXT
    CHECK (user_class_section_result IN ('passed','failed')),

  -- snapshot biaya saat masuk section (agar tetap konsisten walau aturan berubah)
  user_class_section_fee_snapshot JSONB,

  -- jejak waktu enrolment
  user_class_section_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  user_class_section_unassigned_at DATE,
  user_class_section_completed_at  TIMESTAMPTZ,

  -- audit
  user_class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_ucsec_dates CHECK (
    user_class_section_unassigned_at IS NULL
    OR user_class_section_unassigned_at >= user_class_section_assigned_at
  ),
  CONSTRAINT chk_ucsec_result_only_when_completed CHECK (
    (user_class_section_status = 'completed' AND user_class_section_result IS NOT NULL)
    OR (user_class_section_status <> 'completed' AND user_class_section_result IS NULL)
  ),
  CONSTRAINT chk_ucsec_completed_at_when_completed CHECK (
    user_class_section_status <> 'completed'
    OR user_class_section_completed_at IS NOT NULL
  ),

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_ucsec__section_masjid_pair
    FOREIGN KEY (user_class_section_section_id, user_class_section_masjid_id)
    REFERENCES class_sections (class_section_id, class_section_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================================================
-- INDEXES
-- =========================================================

-- unik: satu siswa hanya boleh aktif di satu section per masjid
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsec_active_per_student
  ON user_class_sections (user_class_section_masjid_student_id, user_class_section_masjid_id)
  WHERE user_class_section_deleted_at IS NULL
    AND user_class_section_status = 'active';

-- lookups umum
CREATE INDEX IF NOT EXISTS ix_ucsec_tenant_student_created
  ON user_class_sections (user_class_section_masjid_id, user_class_section_masjid_student_id, user_class_section_created_at DESC)
  WHERE user_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ucsec_tenant_status_created
  ON user_class_sections (user_class_section_masjid_id, user_class_section_status, user_class_section_created_at DESC)
  WHERE user_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsec_section_alive
  ON user_class_sections (user_class_section_section_id)
  WHERE user_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsec_student_alive
  ON user_class_sections (user_class_section_masjid_student_id)
  WHERE user_class_section_deleted_at IS NULL;

-- BRIN untuk range waktu
CREATE INDEX IF NOT EXISTS brin_ucsec_created_at
  ON user_class_sections USING BRIN (user_class_section_created_at);

CREATE INDEX IF NOT EXISTS brin_ucsec_updated_at
  ON user_class_sections USING BRIN (user_class_section_updated_at);

COMMIT;
