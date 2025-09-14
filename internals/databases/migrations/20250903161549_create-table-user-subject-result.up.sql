-- =========================================
-- UP Migration — user_subject_summary (simple, backend-driven)
-- =========================================
BEGIN;

CREATE TABLE IF NOT EXISTS user_subject_summary (
  -- PK
  user_subject_summary_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant
  user_subject_summary_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- siswa (wajib)
  user_subject_summary_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- konteks mapel (wajib)
  user_subject_summary_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- konteks pengampu spesifik (opsional)
  user_subject_summary_csst_id UUID
    REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- (opsional) term/semester jika dipakai di sistem
  user_subject_summary_term_id UUID,

  -- (opsional) penunjuk assessment final/uas (AUDIT SAJA; validasi tipe di-backend)
  user_subject_summary_final_assessment_id UUID
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- hasil akhir rapor (0..100) — dihitung di-backend
  user_subject_summary_final_score NUMERIC(5,2)
    CHECK (user_subject_summary_final_score IS NULL OR (user_subject_summary_final_score BETWEEN 0 AND 100)),

  -- ambang lulus — ditentukan di-backend (default 70)
  user_subject_summary_pass_threshold NUMERIC(5,2) NOT NULL DEFAULT 70
    CHECK (user_subject_summary_pass_threshold BETWEEN 0 AND 100),

  -- status lulus — dihitung di-backend
  user_subject_summary_passed BOOLEAN NOT NULL DEFAULT FALSE,


  user_subject_summary_breakdown JSONB,

  -- metrik progres (opsional)
  user_subject_summary_total_assessments        INTEGER,
  user_subject_summary_total_completed_attempts INTEGER,
  user_subject_summary_last_assessed_at         TIMESTAMPTZ,

  -- penandaan sertifikat sudah diterbitkan (opsional)
  user_subject_summary_certificate_generated BOOLEAN NOT NULL DEFAULT FALSE,

  -- catatan bebas
  user_subject_summary_note TEXT,


  user_subject_summary_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_subject_summary_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_subject_summary_deleted_at TIMESTAMPTZ
);

-- Satu baris aktif per (siswa × subject × term) — soft-delete aware
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_subject_summary_unique_alive
  ON user_subject_summary(
    user_subject_summary_masjid_student_id,
    user_subject_summary_class_subjects_id,
    COALESCE(user_subject_summary_term_id::text, '')
  )
  WHERE user_subject_summary_deleted_at IS NULL;

-- Indeks bantu umum
CREATE INDEX IF NOT EXISTS idx_user_subject_summary_cs_alive
  ON user_subject_summary(user_subject_summary_class_subjects_id)
  WHERE user_subject_summary_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_subject_summary_csst_alive
  ON user_subject_summary(user_subject_summary_csst_id)
  WHERE user_subject_summary_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_subject_summary_masjid_grade
  ON user_subject_summary(user_subject_summary_masjid_id, user_subject_summary_final_score);

-- Time-scan (untuk laporan/arsip besar)
CREATE INDEX IF NOT EXISTS brin_user_subject_summary_created_at
  ON user_subject_summary USING BRIN (user_subject_summary_created_at);

COMMIT;
