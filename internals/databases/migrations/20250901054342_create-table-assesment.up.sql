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

  assessments_title       VARCHAR(180) NOT NULL,
  assessments_description TEXT,

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

