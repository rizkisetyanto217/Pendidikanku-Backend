-- =========================================================
-- 4) SUBMISSIONS (pengumpulan tugas oleh siswa) — FINAL
-- =========================================================
CREATE TABLE IF NOT EXISTS submissions (
  submissions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi inti
  submissions_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  submissions_assessment_id UUID NOT NULL
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- pengumpul: relasi ke masjid_students (BUKAN users langsung)
  submissions_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Attempting model (multi attempt support)
  submissions_attempt_no INT NOT NULL DEFAULT 1,           -- attempt ke-n
  submissions_is_final   BOOLEAN NOT NULL DEFAULT TRUE,    -- attempt dipakai untuk nilai

  -- Isi jawaban
  submissions_text TEXT,                                   -- jawaban teks
  submissions_answer_url TEXT,                             -- file/link jawaban

  submissions_answer_type VARCHAR(16) DEFAULT 'text'       -- jenis jawaban
    CHECK (submissions_answer_type IN ('text','file','link')),

  -- Status & waktu
  submissions_status VARCHAR(24) NOT NULL DEFAULT 'submitted'
    CHECK (submissions_status IN ('draft','submitted','resubmitted','graded','returned')),
  submissions_started_at   TIMESTAMPTZ,                    -- mulai mengerjakan
  submissions_submitted_at TIMESTAMPTZ,
  submissions_late_reason TEXT,

  -- Penilaian
  submissions_score_raw   NUMERIC(5,2)
    CHECK (submissions_score_raw IS NULL OR (submissions_score_raw >= 0 AND submissions_score_raw <= 100)),
  submissions_penalty_percent NUMERIC(5,2)
    CHECK (submissions_penalty_percent IS NULL OR (submissions_penalty_percent >= 0 AND submissions_penalty_percent <= 100)),
  submissions_score_final NUMERIC(5,2)
    CHECK (submissions_score_final IS NULL OR (submissions_score_final >= 0 AND submissions_score_final <= 100)),

  -- Snapshot bobot/KKM (untuk konsistensi bila master berubah)
  submissions_weight_snapshot_percent NUMERIC(5,2)
    CHECK (submissions_weight_snapshot_percent IS NULL OR (submissions_weight_snapshot_percent >= 0 AND submissions_weight_snapshot_percent <= 100)),
  submissions_min_pass_score_snapshot NUMERIC(5,2)
    CHECK (submissions_min_pass_score_snapshot IS NULL OR (submissions_min_pass_score_snapshot >= 0 AND submissions_min_pass_score_snapshot <= 100)),

  -- Feedback
  submissions_feedback TEXT,
  submissions_feedback_visibility VARCHAR(12) DEFAULT 'private'
    CHECK (submissions_feedback_visibility IN ('private','public','return_only')),
  submissions_returned_at TIMESTAMPTZ,

  -- Pengoreksi
  submissions_graded_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  submissions_graded_at TIMESTAMPTZ,
  submissions_is_auto_graded BOOLEAN DEFAULT FALSE,        -- nilai otomatis/manual
  submissions_internal_notes TEXT,                         -- catatan internal guru

  -- Audit
  submissions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_deleted_at TIMESTAMPTZ
);

-- ===================== CONSTRAINTS & INDEXES =====================

-- Unik: 1 row aktif per (assessment, student, attempt_no)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_assessment_student_attempt_alive
  ON submissions(submissions_assessment_id, submissions_student_id, submissions_attempt_no)
  WHERE submissions_deleted_at IS NULL;

-- Unik: hanya 1 attempt final aktif per (assessment, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_final_choice_alive
  ON submissions(submissions_assessment_id, submissions_student_id)
  WHERE submissions_is_final = TRUE AND submissions_deleted_at IS NULL;

-- Index akses cepat per assessment / student / masjid
CREATE INDEX IF NOT EXISTS idx_submissions_assessment
  ON submissions(submissions_assessment_id)
  WHERE submissions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_student
  ON submissions(submissions_student_id)
  WHERE submissions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_masjid
  ON submissions(submissions_masjid_id)
  WHERE submissions_deleted_at IS NULL;

-- Status & grading
CREATE INDEX IF NOT EXISTS idx_submissions_status_alive
  ON submissions(submissions_status)
  WHERE submissions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_graded_by_teacher
  ON submissions(submissions_graded_by_teacher_id)
  WHERE submissions_deleted_at IS NULL;

-- Attempt ordering (ambil attempt terbaru cepat)
CREATE INDEX IF NOT EXISTS idx_submissions_assessment_student_attempt_desc
  ON submissions(submissions_assessment_id, submissions_student_id, submissions_attempt_no DESC)
  WHERE submissions_deleted_at IS NULL;

-- Time-based
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at_alive
  ON submissions(submissions_submitted_at)
  WHERE submissions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_submissions_created_at
  ON submissions USING BRIN (submissions_created_at);

-- Filter “butuh dinilai” cepat
CREATE INDEX IF NOT EXISTS idx_submissions_to_grade
  ON submissions(submissions_assessment_id, submissions_status, submissions_is_final)
  WHERE submissions_deleted_at IS NULL
    AND submissions_status IN ('submitted','resubmitted')
    AND submissions_is_final = TRUE;

-- Tambahan composite untuk dashboard by masjid + status
CREATE INDEX IF NOT EXISTS idx_submissions_masjid_status
  ON submissions(submissions_masjid_id, submissions_status)
  WHERE submissions_deleted_at IS NULL;
