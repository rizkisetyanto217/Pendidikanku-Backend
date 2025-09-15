-- =========================================
-- UP Migration — Assessments (3 tabel, final, tanpa prefix "academic")
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- 1) MASTER TYPES (tanpa ordering)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_types (
  assessment_types_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_types_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  assessment_types_key  VARCHAR(32)  NOT NULL,  -- unik per masjid (uas, uts, tugas, ...)
  assessment_types_name VARCHAR(120) NOT NULL,

  assessment_types_weight_percent NUMERIC(5,2) NOT NULL DEFAULT 0
    CHECK (assessment_types_weight_percent >= 0 AND assessment_types_weight_percent <= 100),

  assessment_types_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  assessment_types_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_types_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_types_deleted_at TIMESTAMPTZ
);

-- unik per masjid + key
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_masjid_key
  ON assessment_types(assessment_types_masjid_id, assessment_types_key);

-- listing aktif
CREATE INDEX IF NOT EXISTS idx_assessment_types_masjid_active
  ON assessment_types(assessment_types_masjid_id, assessment_types_is_active);



-- =========================================================
-- ASSESSMENTS — CLEAN RELATION (ONLY TO CSST)
-- =========================================================
BEGIN;

-- Pastikan unique index tenant-safe ada di CSST
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_masjid
ON class_section_subject_teachers (
  class_section_subject_teachers_id,
  class_section_subject_teachers_masjid_id
);

-- Table assessments
CREATE TABLE IF NOT EXISTS assessments (
  assessments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessments_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Hanya relasi ke CSST (tenant-safe)
  assessments_class_section_subject_teacher_id UUID NULL,

  assessments_type_id UUID
    REFERENCES assessment_types(assessment_types_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  assessments_slug VARCHAR(160),                    -- untuk URL

  assessments_title       VARCHAR(180) NOT NULL,
  assessments_description TEXT,

  -- Jadwal
  assessments_start_at TIMESTAMPTZ,
  assessments_due_at   TIMESTAMPTZ,
  assessments_published_at TIMESTAMPTZ,             -- jadwal rilis
  assessments_closed_at   TIMESTAMPTZ,              -- jadwal tutup

  assessments_duration_minutes INT,                 -- menit
  assessments_total_attempts_allowed INT NOT NULL DEFAULT 1,

  assessments_start_at TIMESTAMPTZ,
  assessments_due_at   TIMESTAMPTZ,

  assessments_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessments_max_score >= 0 AND assessments_max_score <= 100),

  assessments_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessments_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  assessments_created_by_teacher_id UUID,  -- FK ke masjid_teachers

  assessments_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_deleted_at TIMESTAMPTZ
);

-- FK ke CSST
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessments_csst'
  ) THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessments_csst
      FOREIGN KEY (assessments_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Composite tenant-safe FK (masjid_id harus match)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessments_csst_masjid_tenant_safe'
  ) THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessments_csst_masjid_tenant_safe
      FOREIGN KEY (
        assessments_class_section_subject_teacher_id,
        assessments_masjid_id
      )
      REFERENCES class_section_subject_teachers(
        class_section_subject_teachers_id,
        class_section_subject_teachers_masjid_id
      )
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- Indeks
-- =========================================================
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_created_at
  ON assessments(assessments_masjid_id, assessments_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_assessments_type_id
  ON assessments(assessments_type_id);

CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments(assessments_class_section_subject_teacher_id);

CREATE INDEX IF NOT EXISTS idx_assessments_created_by_teacher
  ON assessments(assessments_created_by_teacher_id);

CREATE INDEX IF NOT EXISTS brin_assessments_created_at
  ON assessments USING BRIN (assessments_created_at);

COMMIT;



-- =========================================================
-- 3) ASSESSMENT URLS (selaras dgn announcement_urls)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  assessment_url_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  assessment_url_assessment_id   UUID NOT NULL
    REFERENCES assessments(assessments_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'image','video','attachment','link', dst.)
  assessment_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  assessment_url_href            TEXT,        -- URL publik (boleh NULL jika murni object storage)
  assessment_url_object_key      TEXT,        -- object key aktif di storage
  assessment_url_object_key_old  TEXT,        -- object key lama (retensi in-place replace)
  assessment_url_mime            VARCHAR(80), -- opsional

  -- Tampilan
  assessment_url_label           VARCHAR(160),
  assessment_url_order           INT NOT NULL DEFAULT 0,
  assessment_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  assessment_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_deleted_at      TIMESTAMPTZ,          -- soft delete (versi-per-baris)
  assessment_url_delete_pending_until TIMESTAMPTZ      -- tenggat purge (baris aktif dgn *_old atau baris soft-deleted)
);

-- =========================================================
-- INDEXING / OPTIMIZATION (paritas dg announcement/submission urls)
-- =========================================================

-- Lookup per assessment (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_ass_urls_by_owner_live
  ON assessment_urls (
    assessment_url_assessment_id,
    assessment_url_kind,
    assessment_url_is_primary DESC,
    assessment_url_order,
    assessment_url_created_at
  )
  WHERE assessment_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_ass_urls_by_masjid_live
  ON assessment_urls (assessment_url_masjid_id)
  WHERE assessment_url_deleted_at IS NULL;

-- Satu primary per (assessment, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ass_urls_primary_per_kind_alive
  ON assessment_urls (assessment_url_assessment_id, assessment_url_kind)
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_is_primary = TRUE;

-- Anti-duplikat href per assessment (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ass_urls_assessment_href_alive
  ON assessment_urls (assessment_url_assessment_id, assessment_url_href)
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_href IS NOT NULL;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_ass_urls_purge_due
  ON assessment_urls (assessment_url_delete_pending_until)
  WHERE assessment_url_delete_pending_until IS NOT NULL
    AND (
      (assessment_url_deleted_at IS NULL  AND assessment_url_object_key_old IS NOT NULL) OR
      (assessment_url_deleted_at IS NOT NULL AND assessment_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_ass_urls_label_trgm_live
  ON assessment_urls USING GIN (assessment_url_label gin_trgm_ops)
  WHERE assessment_url_deleted_at IS NULL;
