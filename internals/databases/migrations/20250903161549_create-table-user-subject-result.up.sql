-- -- =========================================
-- -- UP Migration — user_class_subjects (dengan rekap absensi)
-- -- =========================================
-- BEGIN;

-- CREATE TABLE IF NOT EXISTS user_class_subjects (
--   -- PK
--   user_class_subject_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

--   -- tenant
--   user_class_subject_masjid_id UUID NOT NULL
--     REFERENCES masjids(masjid_id) ON DELETE CASCADE,

--   -- siswa (wajib)
--   user_class_subject_masjid_student_id UUID NOT NULL
--     REFERENCES masjid_students(masjid_student_id)
--     ON UPDATE CASCADE ON DELETE CASCADE,

--   -- konteks mapel (wajib)
--   user_class_subject_class_subject_id UUID NOT NULL
--     REFERENCES class_subjects(class_subjects_id)
--     ON UPDATE CASCADE ON DELETE RESTRICT,

--   -- (opsional) penunjuk assessment final/uas (AUDIT SAJA; validasi tipe di-backend)
--   user_class_subject_final_assessment_id UUID
--     REFERENCES assessments(assessments_id)
--     ON UPDATE CASCADE ON DELETE SET NULL,

--   -- hasil akhir rapor (0..100) — dihitung di-backend
--   user_class_subject_final_score NUMERIC(5,2)
--     CHECK (user_class_subject_final_score IS NULL OR (user_class_subject_final_score BETWEEN 0 AND 100)),

--   -- ambang lulus — ditentukan di-backend (default 70)
--   user_class_subject_pass_threshold NUMERIC(5,2) NOT NULL DEFAULT 70
--     CHECK (user_class_subject_pass_threshold BETWEEN 0 AND 100),

--   -- status lulus — dihitung di-backend
--   user_class_subject_passed BOOLEAN NOT NULL DEFAULT FALSE,

--   -- breakdown skor (opsional, by assessment type/weight)
--   user_class_subject_breakdown JSONB,

--   -- ==============================
--   -- Rekap ABSENSI per mapel
--   -- ==============================
--   user_class_subject_total_meetings   INTEGER NOT NULL DEFAULT 0 CHECK (user_class_subject_total_meetings >= 0),
--   user_class_subject_attend_present   INTEGER NOT NULL DEFAULT 0 CHECK (user_class_subject_attend_present >= 0),
--   user_class_subject_attend_sick      INTEGER NOT NULL DEFAULT 0 CHECK (user_class_subject_attend_sick >= 0),
--   user_class_subject_attend_permit    INTEGER NOT NULL DEFAULT 0 CHECK (user_class_subject_attend_permit >= 0),
--   user_class_subject_attend_unexcused INTEGER NOT NULL DEFAULT 0 CHECK (user_class_subject_attend_unexcused >= 0),

--   CONSTRAINT chk_user_class_subject_attendance_sum
--     CHECK (
--       (user_class_subject_attend_present
--        + user_class_subject_attend_sick
--        + user_class_subject_attend_permit
--        + user_class_subject_attend_unexcused)
--       <= user_class_subject_total_meetings
--     ),

--   -- metrik progres (opsional)
--   user_class_subject_total_assessments        INTEGER,
--   user_class_subject_total_completed_attempts INTEGER,
--   user_class_subject_last_assessed_at         TIMESTAMPTZ,

--   -- penandaan sertifikat sudah diterbitkan (opsional)
--   user_class_subject_certificate_generated BOOLEAN NOT NULL DEFAULT FALSE,

--   -- catatan bebas
--   user_class_subject_note TEXT,

--   user_class_subject_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--   user_class_subject_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--   user_class_subject_deleted_at TIMESTAMPTZ
-- );

-- -- Satu baris aktif per (siswa × subject) — soft-delete aware
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_subject_unique_alive
--   ON user_class_subjects(
--     user_class_subject_masjid_student_id,
--     user_class_subject_class_subject_id
--   )
--   WHERE user_class_subject_deleted_at IS NULL;

-- -- Indeks bantu umum
-- CREATE INDEX IF NOT EXISTS idx_user_class_subject_cs_alive
--   ON user_class_subjects(user_class_subject_class_subject_id)
--   WHERE user_class_subject_deleted_at IS NULL;

-- CREATE INDEX IF NOT EXISTS idx_user_class_subject_masjid_grade
--   ON user_class_subjects(user_class_subject_masjid_id, user_class_subject_final_score);

-- -- (Opsional) Index untuk kueri laporan absensi cepat per mapel
-- CREATE INDEX IF NOT EXISTS idx_user_class_subject_masjid_subject_attendance
--   ON user_class_subjects(
--     user_class_subject_masjid_id,
--     user_class_subject_class_subject_id,
--     user_class_subject_total_meetings
--   )
--   WHERE user_class_subject_deleted_at IS NULL;

-- -- Time-scan (untuk laporan/arsip besar)
-- CREATE INDEX IF NOT EXISTS brin_user_class_subject_created_at
--   ON user_class_subjects USING BRIN (user_class_subject_created_at);

-- COMMIT;
