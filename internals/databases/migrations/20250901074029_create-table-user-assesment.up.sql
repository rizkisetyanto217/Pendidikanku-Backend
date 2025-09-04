-- =========================================================
-- 4) SUBMISSIONS (pengumpulan tugas oleh user)
-- =========================================================
CREATE TABLE IF NOT EXISTS submissions (
  submissions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- keterkaitan tenant & entitas
  submissions_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  submissions_assessment_id UUID NOT NULL
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  submissions_user_id UUID NOT NULL
    REFERENCES users(id)  -- langsung ke tabel users
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
  submissions_graded_by_teacher_id UUID,  -- FK opsional ke masjid_teachers
  submissions_graded_at TIMESTAMPTZ,

  submissions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submissions_deleted_at TIMESTAMPTZ
);

-- (Opsional) tambahkan FK grader jika tabel tersedia :
-- ALTER TABLE submissions
--   ADD CONSTRAINT fk_submissions_graded_by_teacher
--   FOREIGN KEY (submissions_graded_by_teacher_id)
--   REFERENCES masjid_teachers(masjid_teachers_id)
--   ON UPDATE CASCADE ON DELETE SET NULL;

-- Unik 1 submission aktif per (assessment, user)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_assessment_user
  ON submissions(submissions_assessment_id, submissions_user_id)
  WHERE submissions_deleted_at IS NULL;

-- indeks bantu
CREATE INDEX IF NOT EXISTS idx_submissions_assessment
  ON submissions(submissions_assessment_id);

CREATE INDEX IF NOT EXISTS idx_submissions_user
  ON submissions(submissions_user_id);

CREATE INDEX IF NOT EXISTS idx_submissions_masjid
  ON submissions(submissions_masjid_id);

CREATE INDEX IF NOT EXISTS brin_submissions_created_at
  ON submissions USING BRIN (submissions_created_at);


-- =========================================================
-- 5) SUBMISSION URLS (lampiran kiriman user)
-- =========================================================
CREATE TABLE IF NOT EXISTS submission_urls (
  submission_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  submission_urls_submission_id UUID NOT NULL
    REFERENCES submissions(submissions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  submission_urls_label VARCHAR(120),
  submission_urls_href  TEXT NOT NULL,

  -- opsi "trash" seperti versi guru
  submission_urls_trash_url            TEXT,
  submission_urls_delete_pending_until TIMESTAMPTZ,

  submission_urls_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  submission_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_urls_deleted_at TIMESTAMPTZ
);

-- anti duplikat href per submission (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submission_urls_submission_href
  ON submission_urls(submission_urls_submission_id, submission_urls_href)
  WHERE submission_urls_deleted_at IS NULL;

-- time-scan
CREATE INDEX IF NOT EXISTS brin_submission_urls_created_at
  ON submission_urls USING BRIN (submission_urls_created_at);

