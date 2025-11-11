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
  student_class_section_school_student_id UUID NOT NULL
    REFERENCES school_students(school_student_id) ON DELETE RESTRICT,
  student_class_section_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE RESTRICT,

  -- section (kelas paralel) & tenant
  student_class_section_section_id UUID NOT NULL,
  -- slug adalah teks; gunakan panjang aman
  student_class_section_section_slug_snapshot VARCHAR(160) NOT NULL,

  -- lifecycle enrolment
  student_class_section_status TEXT NOT NULL DEFAULT 'active'
    CHECK (student_class_section_status IN ('active','inactive','completed')),

  -- outcome (hasil akhir)
  student_class_section_result TEXT
    CHECK (student_class_section_result IN ('passed','failed')),

  -- ==========================
  -- 📘 NILAI AKHIR (grades)
  -- ==========================
  -- Skor 0..100 dengan 2 desimal (opsional)
  student_class_section_final_score NUMERIC(5,2)
    CHECK (
      student_class_section_final_score IS NULL
      OR (student_class_section_final_score >= 0 AND student_class_section_final_score <= 100)
    ),
  -- Huruf nilai (A, A-, B+, ... sampai E/F) maksimal 3 char
  student_class_section_final_grade_letter VARCHAR(3),
  -- Konversi poin (0..4) untuk keperluan IPK
  student_class_section_final_grade_point NUMERIC(3,2)
    CHECK (
      student_class_section_final_grade_point IS NULL
      OR (student_class_section_final_grade_point >= 0 AND student_class_section_final_grade_point <= 4)
    ),
  -- Peringkat di dalam section (opsional, >0)
  student_class_section_final_rank INT
    CHECK (student_class_section_final_rank IS NULL OR student_class_section_final_rank > 0),
  -- Catatan penilaian
  student_class_section_final_remarks TEXT,
  -- (opsional) siapa yang menilai + kapan
  student_class_section_graded_by_teacher_id UUID,
  student_class_section_graded_at TIMESTAMPTZ,

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

  -- Guards (tanggal & result)
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

  -- Guards (nilai hanya ketika completed; minimal satu metrik terisi)
  CONSTRAINT chk_scsec_grades_only_when_completed CHECK (
    CASE
      WHEN student_class_section_status = 'completed' THEN TRUE
      ELSE
        student_class_section_final_score IS NULL AND
        student_class_section_final_grade_letter IS NULL AND
        student_class_section_final_grade_point IS NULL AND
        student_class_section_final_rank IS NULL AND
        student_class_section_final_remarks IS NULL AND
        student_class_section_graded_at IS NULL
    END
  ),
  CONSTRAINT chk_scsec_at_least_one_final_measure_when_completed CHECK (
    student_class_section_status <> 'completed'
    OR (
      student_class_section_final_score IS NOT NULL
      OR student_class_section_final_grade_letter IS NOT NULL
      OR student_class_section_final_grade_point IS NOT NULL
    )
  ),

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_scsec__section_school_pair
    FOREIGN KEY (student_class_section_section_id, student_class_section_school_id)
    REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================================================
-- INDEXES
-- =========================================================

-- unik: satu siswa hanya boleh aktif di satu section per school
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsec_active_per_student
  ON student_class_sections (student_class_section_school_student_id, student_class_section_school_id)
  WHERE student_class_section_deleted_at IS NULL
    AND student_class_section_status = 'active';

-- lookups umum
CREATE INDEX IF NOT EXISTS ix_scsec_tenant_student_created
  ON student_class_sections (student_class_section_school_id, student_class_section_school_student_id, student_class_section_created_at DESC)
  WHERE student_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_scsec_tenant_status_created
  ON student_class_sections (student_class_section_school_id, student_class_section_status, student_class_section_created_at DESC)
  WHERE student_class_section_deleted_at IS NULL;

-- JOIN cepat ke class_sections (komposit)
CREATE INDEX IF NOT EXISTS idx_scsec_section_school_alive
  ON student_class_sections (student_class_section_section_id, student_class_section_school_id)
  WHERE student_class_section_deleted_at IS NULL;

-- lookup spesifik
CREATE INDEX IF NOT EXISTS idx_scsec_section_alive
  ON student_class_sections (student_class_section_section_id)
  WHERE student_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsec_student_alive
  ON student_class_sections (student_class_section_school_student_id)
  WHERE student_class_section_deleted_at IS NULL;

-- BRIN untuk range waktu
CREATE INDEX IF NOT EXISTS brin_scsec_created_at
  ON student_class_sections USING BRIN (student_class_section_created_at);

CREATE INDEX IF NOT EXISTS brin_scsec_updated_at
  ON student_class_sections USING BRIN (student_class_section_updated_at);

-- 🔎 Index untuk laporan nilai
CREATE INDEX IF NOT EXISTS ix_scsec_tenant_gradedat
  ON student_class_sections (student_class_section_school_id, student_class_section_graded_at DESC)
  WHERE student_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_scsec_completed_with_grade
  ON student_class_sections (student_class_section_school_id, student_class_section_status, student_class_section_final_score)
  WHERE student_class_section_deleted_at IS NULL
    AND student_class_section_status = 'completed';

COMMIT;
