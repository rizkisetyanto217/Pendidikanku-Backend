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



-- =========================================================
-- 5) SUBMISSION URLS (lampiran kiriman user) — selaras dgn announcement_urls
-- =========================================================
CREATE TABLE IF NOT EXISTS submission_urls (
  submission_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  submission_url_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  submission_url_submission_id   UUID NOT NULL
    REFERENCES submissions(submissions_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (selaras: 'image','video','attachment','link', dst.)
  submission_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  submission_url_href            TEXT,        -- URL publik (bisa null jika pakai object storage saja)
  submission_url_object_key      TEXT,        -- object key aktif di storage
  submission_url_object_key_old  TEXT,        -- object key lama (retensi in-place replace)
  submission_url_mime            VARCHAR(80), -- opsional

  -- Tampilan
  submission_url_label           VARCHAR(160),
  submission_url_order           INT NOT NULL DEFAULT 0,
  submission_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  submission_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_url_deleted_at      TIMESTAMPTZ,          -- soft delete (versi-per-baris)
  submission_url_delete_pending_until TIMESTAMPTZ      -- tenggat purge (baris aktif dgn *_old atau baris soft-deleted)
);

-- =========================================================
-- INDEXING / OPTIMIZATION (paritas dg announcement_urls)
-- =========================================================

-- Lookup per submission (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_sub_urls_by_owner_live
  ON submission_urls (
    submission_url_submission_id,
    submission_url_kind,
    submission_url_is_primary DESC,
    submission_url_order,
    submission_url_created_at
  )
  WHERE submission_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_sub_urls_by_masjid_live
  ON submission_urls (submission_url_masjid_id)
  WHERE submission_url_deleted_at IS NULL;

-- Satu primary per (submission, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sub_urls_primary_per_kind_alive
  ON submission_urls (submission_url_submission_id, submission_url_kind)
  WHERE submission_url_deleted_at IS NULL
    AND submission_url_is_primary = TRUE;

-- Anti-duplikat href per submission (live only) — opsional, berguna utk link eksternal
CREATE UNIQUE INDEX IF NOT EXISTS uq_sub_urls_submission_href_alive
  ON submission_urls (submission_url_submission_id, submission_url_href)
  WHERE submission_url_deleted_at IS NULL
    AND submission_url_href IS NOT NULL;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_sub_urls_purge_due
  ON submission_urls (submission_url_delete_pending_until)
  WHERE submission_url_delete_pending_until IS NOT NULL
    AND (
      (submission_url_deleted_at IS NULL  AND submission_url_object_key_old IS NOT NULL) OR
      (submission_url_deleted_at IS NOT NULL AND submission_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_sub_urls_label_trgm_live
  ON submission_urls USING GIN (submission_url_label gin_trgm_ops)
  WHERE submission_url_deleted_at IS NULL;
