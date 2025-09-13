-- =========================================
-- DOWN Migration — Assessments (3 tabel, tanpa prefix "academic")
-- =========================================
BEGIN;


-- =========================================
-- A) ASSESSMENTS
-- =========================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_assessments_updated_at') THEN
    DROP TRIGGER trg_touch_assessments_updated_at ON assessments;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_touch_assessments_updated_at() RESTRICT;

-- indexes
DROP INDEX IF EXISTS idx_assessments_masjid_created_at;
DROP INDEX IF EXISTS idx_assessments_type_id;
DROP INDEX IF EXISTS idx_assessments_csst;
DROP INDEX IF EXISTS idx_assessments_section;
DROP INDEX IF EXISTS idx_assessments_subject;
DROP INDEX IF EXISTS idx_assessments_created_by_teacher;
DROP INDEX IF EXISTS brin_assessments_created_at;

-- table
DROP TABLE IF EXISTS assessments;

-- =========================================
-- B) ASSESSMENT TYPES (master)
-- =========================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_assessment_types_updated_at') THEN
    DROP TRIGGER trg_touch_assessment_types_updated_at ON assessment_types;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_touch_assessment_types_updated_at() RESTRICT;

-- indexes
DROP INDEX IF EXISTS uq_assessment_types_masjid_key;
DROP INDEX IF EXISTS idx_assessment_types_masjid_active;

-- table
DROP TABLE IF EXISTS assessment_types;

COMMIT;
