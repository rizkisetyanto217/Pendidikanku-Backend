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
-- 2) ASSESSMENTS (tanpa weight_override) — FINAL
-- =========================================================
CREATE TABLE IF NOT EXISTS assessments (
  assessments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessments_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- opsional relasi kelas/mapel/pengampu
  assessments_class_section_id UUID,
  assessments_class_subjects_id UUID,
  assessments_class_section_subject_teacher_id UUID, -- (full name csst)

  -- tipe (FK ke master)
  assessments_type_id UUID
    REFERENCES assessment_types(assessment_types_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  assessments_title       VARCHAR(180) NOT NULL,
  assessments_description TEXT,

  assessments_start_at TIMESTAMPTZ,
  assessments_due_at   TIMESTAMPTZ,

  assessments_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessments_max_score >= 0 AND assessments_max_score <= 100),

  assessments_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessments_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  -- creator (guru global; beda dengan CSST pengampu spesifik)
  assessments_created_by_teacher_id UUID,  -- -> FK ke masjid_teachers

  assessments_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_deleted_at TIMESTAMPTZ
);

-- indeks
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_created_at
  ON assessments(assessments_masjid_id, assessments_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_assessments_type_id
  ON assessments(assessments_type_id);

CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments(assessments_class_section_subject_teacher_id);

CREATE INDEX IF NOT EXISTS idx_assessments_section
  ON assessments(assessments_class_section_id);

CREATE INDEX IF NOT EXISTS idx_assessments_subject
  ON assessments(assessments_class_subjects_id);

CREATE INDEX IF NOT EXISTS idx_assessments_created_by_teacher
  ON assessments(assessments_created_by_teacher_id);

CREATE INDEX IF NOT EXISTS brin_assessments_created_at
  ON assessments USING BRIN (assessments_created_at);




-- =========================================================
-- 3) ASSESSMENT URLS (tanpa mime/size/checksum/audience)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_urls_assessment_id UUID NOT NULL
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  assessment_urls_label VARCHAR(120),
  assessment_urls_href  TEXT NOT NULL,

  -- tambahan kolom baru
  assessment_urls_trash_url           TEXT,
  assessment_urls_delete_pending_until TIMESTAMPTZ,

  assessment_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_urls_deleted_at TIMESTAMPTZ,

  -- publish flags (disederhanakan)
  assessment_urls_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_urls_is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_urls_published_at TIMESTAMPTZ,
  assessment_urls_expires_at   TIMESTAMPTZ,
  assessment_urls_public_slug  VARCHAR(64),
  assessment_urls_public_token VARCHAR(64)
);

-- anti duplikat file per assessment (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_assessment_href
  ON assessment_urls(assessment_urls_assessment_id, assessment_urls_href)
  WHERE assessment_urls_deleted_at IS NULL;

-- filter publikasi cepat
CREATE INDEX IF NOT EXISTS idx_assessment_urls_publish_flags
  ON assessment_urls(assessment_urls_is_published, assessment_urls_is_active)
  WHERE assessment_urls_deleted_at IS NULL;

-- time-scan
CREATE INDEX IF NOT EXISTS brin_assessment_urls_created_at
  ON assessment_urls USING BRIN (assessment_urls_created_at);


COMMIT;
