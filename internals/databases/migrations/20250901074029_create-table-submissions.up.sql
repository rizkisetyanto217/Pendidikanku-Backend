-- =========================================================
-- 4) SUBMISSIONS (pengumpulan tugas oleh siswa) — FINAL
-- =========================================================
CREATE TABLE IF NOT EXISTS submissions (
  submission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- keterkaitan tenant & entitas
  submission_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  submission_assessment_id UUID NOT NULL
    REFERENCES assessments(assessment_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- pengumpul: relasi ke masjid_students (BUKAN users langsung)
  submission_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- isi & status pengumpulan
  submission_text TEXT,
  submission_status VARCHAR(24) NOT NULL DEFAULT 'submitted'
    CHECK (submission_status IN ('draft','submitted','resubmitted','graded','returned')),

  submission_submitted_at TIMESTAMPTZ,
  submission_is_late      BOOLEAN,

  -- penilaian
  submission_score    NUMERIC(5,2) CHECK (submission_score >= 0 AND submission_score <= 100),
  submission_feedback TEXT,

  -- pengoreksi: relasi ke masjid_teachers
  submission_graded_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  submission_graded_at TIMESTAMPTZ,

  submission_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_deleted_at TIMESTAMPTZ
);

-- ============== INDEXES (submissions) ==============

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_id_tenant
  ON submissions (submission_id, submission_masjid_id);

-- Unik: 1 submission aktif per (assessment, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_assessment_student_alive
  ON submissions (submission_assessment_id, submission_student_id)
  WHERE submission_deleted_at IS NULL;

-- (opsional, lebih ketat & tenant-safe; aktifkan bila perlu)
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_tenant_assessment_student_alive
--   ON submissions (submission_masjid_id, submission_assessment_id, submission_student_id)
--   WHERE submission_deleted_at IS NULL;

-- Jalur query umum
CREATE INDEX IF NOT EXISTS idx_submissions_assessment_alive
  ON submissions (submission_assessment_id)
  WHERE submission_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_student_alive
  ON submissions (submission_student_id)
  WHERE submission_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_masjid_alive
  ON submissions (submission_masjid_id)
  WHERE submission_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_status_alive
  ON submissions (submission_status)
  WHERE submission_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_graded_by_teacher_alive
  ON submissions (submission_graded_by_teacher_id)
  WHERE submission_deleted_at IS NULL;

-- Time-based
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at_alive
  ON submissions (submission_submitted_at)
  WHERE submission_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_submissions_created_at
  ON submissions USING BRIN (submission_created_at);

-- (opsional) cari feedback cepat
-- CREATE INDEX IF NOT EXISTS gin_submissions_feedback_trgm_alive
--   ON submissions USING GIN (submission_feedback gin_trgm_ops)
--   WHERE submission_deleted_at IS NULL;



-- =========================================================
-- 5) SUBMISSION_URLS (lampiran kiriman user)
-- =========================================================
CREATE TABLE IF NOT EXISTS submission_urls (
  submission_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  submission_url_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  submission_url_submission_id   UUID NOT NULL
    REFERENCES submissions(submission_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (selaras: 'image','video','attachment','link', dst.)
  submission_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  -- storage (2-slot + retensi)
  submission_url                  TEXT,        -- aktif
  submission_url_object_key           TEXT,
  submission_url_old              TEXT,        -- kandidat delete
  submission_url_object_key_old       TEXT,
  submission_url_delete_pending_until TIMESTAMPTZ, -- jadwal hard delete old


  -- Tampilan
  submission_url_label           VARCHAR(160),
  submission_url_order           INT NOT NULL DEFAULT 0,
  submission_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Pengumpul: relasi ke masjid_students (BUKAN users langsung)
  submission_url_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Pengumpul: relasi ke masjid_teachers (BUKAN users langsung)
  submission_url_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Audit & retensi
  submission_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_url_deleted_at      TIMESTAMPTZ,          -- soft delete (versi-per-baris)
);

-- ============== INDEXES (submission_urls) ==============

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submission_urls_id_tenant
  ON submission_urls (submission_url_id, submission_url_masjid_id);

-- Lookup per submission (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_submission_urls_by_owner_live
  ON submission_urls (
    submission_url_submission_id,
    submission_url_kind,
    submission_url_is_primary DESC,
    submission_url_order,
    submission_url_created_at
  )
  WHERE submission_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_submission_urls_by_masjid_live
  ON submission_urls (submission_url_masjid_id)
  WHERE submission_url_deleted_at IS NULL;

-- Satu primary per (submission, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submission_urls_primary_per_kind_alive
  ON submission_urls (submission_url_submission_id, submission_url_kind)
  WHERE submission_url_deleted_at IS NULL
    AND submission_url_is_primary = TRUE;

-- Anti-duplikat href per submission (live only; case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submission_urls_submission_href_alive
  ON submission_urls (submission_url_submission_id, LOWER(submission_url_href))
  WHERE submission_url_deleted_at IS NULL
    AND submission_url_href IS NOT NULL;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_submission_urls_purge_due
  ON submission_urls (submission_url_delete_pending_until)
  WHERE submission_url_delete_pending_until IS NOT NULL
    AND (
      (submission_url_deleted_at IS NULL  AND submission_url_object_key_old IS NOT NULL) OR
      (submission_url_deleted_at IS NOT NULL AND submission_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_submission_urls_label_trgm_live
  ON submission_urls USING GIN (submission_url_label gin_trgm_ops)
  WHERE submission_url_deleted_at IS NULL;

-- Time-scan
CREATE INDEX IF NOT EXISTS brin_submission_urls_created_at
  ON submission_urls USING BRIN (submission_url_created_at);