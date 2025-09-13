-- =========================================================
-- 4) SUBMISSIONS (pengumpulan tugas oleh user)
-- =========================================================
-- =========================================================
-- 4) SUBMISSIONS (pengumpulan tugas oleh siswa) — FINAL
-- =========================================================
CREATE TABLE IF NOT EXISTS submissions (
  submissions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- keterkaitan tenant & entitas
  submissions_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  submissions_assessment_id UUID NOT NULL
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- pengumpul: relasi ke masjid_students (BUKAN users langsung)
  submissions_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- isi & status pengumpulan
  submissions_text TEXT,
  submissions_status VARCHAR(24) NOT NULL DEFAULT 'submitted'
    CHECK (submissions_status IN ('draft','submitted','resubmitted','graded','returned')),

  submissions_submitted_at TIMESTAMPTZ,
  submissions_is_late      BOOLEAN,

  -- penilaian
  submissions_score    NUMERIC(5,2) CHECK (submissions_score >= 0 AND submissions_score <= 100),
  submissions_feedback TEXT,

  -- pengoreksi: relasi ke masjid_teachers
  submissions_graded_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  submissions_graded_at TIMESTAMPTZ,

  submissions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_deleted_at TIMESTAMPTZ
);

-- Unik: 1 submission aktif per (assessment, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_assessment_student_alive
  ON submissions(submissions_assessment_id, submissions_student_id)
  WHERE submissions_deleted_at IS NULL;

-- Indeks bantu
CREATE INDEX IF NOT EXISTS idx_submissions_assessment
  ON submissions(submissions_assessment_id);

CREATE INDEX IF NOT EXISTS idx_submissions_student
  ON submissions(submissions_student_id);

CREATE INDEX IF NOT EXISTS idx_submissions_masjid
  ON submissions(submissions_masjid_id);

CREATE INDEX IF NOT EXISTS idx_submissions_status_alive
  ON submissions(submissions_status)
  WHERE submissions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_graded_by_teacher
  ON submissions(submissions_graded_by_teacher_id);

-- Time-based
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at
  ON submissions(submissions_submitted_at);

CREATE INDEX IF NOT EXISTS brin_submissions_created_at
  ON submissions USING BRIN (submissions_created_at);
