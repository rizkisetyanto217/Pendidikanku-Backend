-- =========================================
-- UP Migration — Assessments (3 tabel, final)
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- 1) ASSESSMENT_TYPES (master types)
--    - tabel plural
--    - kolom singular (prefix assessment_type_)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_types (
  assessment_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  assessment_type_key  VARCHAR(32)  NOT NULL,  -- unik per masjid (uas, uts, tugas, ...)
  assessment_type_name VARCHAR(120) NOT NULL,

  assessment_type_weight_percent NUMERIC(5,2) NOT NULL DEFAULT 0
    CHECK (assessment_type_weight_percent >= 0 AND assessment_type_weight_percent <= 100),

  assessment_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  assessment_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_deleted_at TIMESTAMPTZ
);

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_id_tenant
  ON assessment_types (assessment_type_id, assessment_type_masjid_id);

-- Unik per masjid + key (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_key_per_masjid_alive
  ON assessment_types (assessment_type_masjid_id, LOWER(assessment_type_key))
  WHERE assessment_type_deleted_at IS NULL;

-- Listing aktif / filter umum
CREATE INDEX IF NOT EXISTS idx_assessment_types_masjid_active
  ON assessment_types (assessment_type_masjid_id, assessment_type_is_active)
  WHERE assessment_type_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_assessment_types_created_at
  ON assessment_types USING BRIN (assessment_type_created_at);



-- =========================================================
-- 2) ASSESSMENTS — clean relation (ONLY TO CSST)
--    - tabel plural
--    - kolom singular (prefix assessment_)
-- =========================================================
-- Pastikan unique index tenant-safe ada di CSST (aman kalau sudah ada)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_masjid
ON class_section_subject_teachers (
  class_section_subject_teacher_id,
  class_section_subject_teacher_masjid_id
);

CREATE TABLE IF NOT EXISTS assessments (
  assessment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Hanya relasi ke CSST (tenant-safe)
  assessment_class_section_subject_teacher_id UUID NULL,

  assessment_type_id UUID
    REFERENCES assessment_types(assessment_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  assessment_slug VARCHAR(160),                    -- untuk URL

  assessment_title       VARCHAR(180) NOT NULL,
  assessment_description TEXT,

  -- Jadwal
  assessment_start_at      TIMESTAMPTZ,
  assessment_due_at        TIMESTAMPTZ,
  assessment_published_at  TIMESTAMPTZ,            -- jadwal rilis
  assessment_closed_at     TIMESTAMPTZ,            -- jadwal tutup

  assessment_duration_minutes        INT,          -- menit
  assessment_total_attempts_allowed  INT NOT NULL DEFAULT 1,

  assessment_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessment_max_score >= 0 AND assessment_max_score <= 100),

  assessment_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  assessment_created_by_teacher_id UUID,          -- FK ke masjid_teachers (opsional)

  assessment_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_deleted_at TIMESTAMPTZ
);

-- FK ke CSST (single col)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_csst') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_csst
      FOREIGN KEY (assessment_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teacher_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Composite tenant-safe FK (assessment_masjid_id harus match CSST.masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_csst_masjid_tenant_safe') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_csst_masjid_tenant_safe
      FOREIGN KEY (
        assessment_class_section_subject_teacher_id,
        assessment_masjid_id
      )
      REFERENCES class_section_subject_teachers(
        class_section_subject_teacher_id,
        class_section_subject_teacher_masjid_id
      )
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_id_tenant
  ON assessments (assessment_id, assessment_masjid_id);

-- Unik slug per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_slug_per_tenant_alive
  ON assessments (assessment_masjid_id, LOWER(assessment_slug))
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

-- Pencarian slug cepat (alive only)
CREATE INDEX IF NOT EXISTS gin_assessments_slug_trgm_alive
  ON assessments USING GIN (LOWER(assessment_slug) gin_trgm_ops)
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

-- Indeks akses umum
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_created_at
  ON assessments (assessment_masjid_id, assessment_created_at DESC)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_type_id
  ON assessments (assessment_type_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments (assessment_class_section_subject_teacher_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_created_by_teacher
  ON assessments (assessment_created_by_teacher_id)
  WHERE assessment_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_assessments_created_at
  ON assessments USING BRIN (assessment_created_at);



-- =========================================================
-- 3) ASSESSMENT_URLS (selaras dgn announcement_urls)
--    - tabel plural
--    - kolom singular (prefix assessment_url_)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  assessment_url_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  assessment_url_assessment_id   UUID NOT NULL
    REFERENCES assessments(assessment_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'image','video','attachment','link', dst.)
  assessment_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  -- storage (2-slot + retensi)
  assessment_url                  TEXT,        -- aktif
  assessment_url_object_key           TEXT,
  assessment_url_old              TEXT,        -- kandidat delete
  assessment_url_object_key_old       TEXT,
  assessment_url_delete_pending_until TIMESTAMPTZ, -- jadwal hard delete old

  -- Tampilan
  assessment_url_label           VARCHAR(160),
  assessment_url_order           INT NOT NULL DEFAULT 0,
  assessment_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  assessment_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_deleted_at      TIMESTAMPTZ,          -- soft delete (versi-per-baris)
);

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_id_tenant
  ON assessment_urls (assessment_url_id, assessment_url_masjid_id);

-- Lookup per assessment (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_owner_live
  ON assessment_urls (
    assessment_url_assessment_id,
    assessment_url_kind,
    assessment_url_is_primary DESC,
    assessment_url_order,
    assessment_url_created_at
  )
  WHERE assessment_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_masjid_live
  ON assessment_urls (assessment_url_masjid_id)
  WHERE assessment_url_deleted_at IS NULL;

-- Satu primary per (assessment, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_primary_per_kind_alive
  ON assessment_urls (assessment_url_assessment_id, assessment_url_kind)
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_is_primary = TRUE;

-- Anti-duplikat href per assessment (live only; case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_assessment_href_alive
  ON assessment_urls (assessment_url_assessment_id, LOWER(assessment_url_href))
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_href IS NOT NULL;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_purge_due
  ON assessment_urls (assessment_url_delete_pending_until)
  WHERE assessment_url_delete_pending_until IS NOT NULL
    AND (
      (assessment_url_deleted_at IS NULL  AND assessment_url_object_key_old IS NOT NULL) OR
      (assessment_url_deleted_at IS NOT NULL AND assessment_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_assessment_urls_label_trgm_live
  ON assessment_urls USING GIN (assessment_url_label gin_trgm_ops)
  WHERE assessment_url_deleted_at IS NULL;

COMMIT;