BEGIN;

-- =========================================================
-- A) CLASS SECTION SUBJECT TEACHERS (CSST) — CHILD
-- =========================================================

-- 1) Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sec_subj_teachers_validate_tenant') THEN
    DROP TRIGGER trg_class_sec_subj_teachers_validate_tenant ON class_section_subject_teachers;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'set_timestamp_class_sec_subj_teachers') THEN
    DROP TRIGGER set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;
  END IF;
END$$;

-- 2) Constraints (FK)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_csst_section_masjid') THEN
    ALTER TABLE class_section_subject_teachers DROP CONSTRAINT fk_csst_section_masjid;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_csst_teacher_membership') THEN
    ALTER TABLE class_section_subject_teachers DROP CONSTRAINT fk_csst_teacher_membership;
  END IF;
END$$;

-- 3) Indexes
DROP INDEX IF EXISTS uq_csst_active_unique;
DROP INDEX IF EXISTS idx_csst_teacher_alive;
DROP INDEX IF EXISTS idx_csst_masjid_alive;
DROP INDEX IF EXISTS idx_csst_section_subject_active_alive;

-- 4) Table
DROP TABLE IF EXISTS class_section_subject_teachers;

-- 5) Functions
DROP FUNCTION IF EXISTS fn_class_sec_subj_teachers_validate_tenant();
DROP FUNCTION IF EXISTS trg_set_timestamp_class_sec_subj_teachers();


-- =========================================================
-- B) CLASS SUBJECTS — CHILD (mengacu ke subjects/classes/academic_terms)
-- =========================================================

-- 1) Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cs_term_tenant_check') THEN
    DROP TRIGGER trg_cs_term_tenant_check ON class_subjects;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'set_timestamp_class_subjects') THEN
    DROP TRIGGER set_timestamp_class_subjects ON class_subjects;
  END IF;
END$$;

-- 2) Constraints (FK)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_class_masjid_pair') THEN
    ALTER TABLE class_subjects DROP CONSTRAINT fk_cs_class_masjid_pair;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_term') THEN
    ALTER TABLE class_subjects DROP CONSTRAINT fk_cs_term;
  END IF;
END$$;

-- 3) Indexes
DROP INDEX IF EXISTS uq_class_subjects_by_term;

DROP INDEX IF EXISTS idx_cs_masjid_class_term_active;
DROP INDEX IF EXISTS idx_cs_masjid_subject_term_active;

DROP INDEX IF EXISTS idx_cs_term_alive;
DROP INDEX IF EXISTS idx_cs_masjid_active;
DROP INDEX IF EXISTS idx_cs_class_order;
DROP INDEX IF EXISTS idx_cs_masjid_alive;

DROP INDEX IF EXISTS gin_cs_desc_trgm;

-- 4) Table
DROP TABLE IF EXISTS class_subjects;

-- 5) Functions
DROP FUNCTION IF EXISTS fn_cs_term_tenant_check();
DROP FUNCTION IF EXISTS trg_set_timestamp_class_subjects();


-- =========================================================
-- C) SUBJECTS — PARENT
--    (Jika tabel ini sudah ada sebelum migration ini, hati-hati:
--     baris DROP TABLE di bawah akan menghapusnya. Sesuaikan bila perlu.)
-- =========================================================

-- 1) Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_touch_updated_at') THEN
    DROP TRIGGER trg_subjects_touch_updated_at ON subjects;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_normalize') THEN
    DROP TRIGGER trg_subjects_normalize ON subjects;
  END IF;
END$$;

-- 2) Constraints (CHECK)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_code_not_blank' AND conrelid='subjects'::regclass) THEN
    ALTER TABLE subjects DROP CONSTRAINT chk_subjects_code_not_blank;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_slug_not_blank' AND conrelid='subjects'::regclass) THEN
    ALTER TABLE subjects DROP CONSTRAINT chk_subjects_slug_not_blank;
  END IF;
END$$;

-- 3) Indexes
DROP INDEX IF EXISTS uq_subjects_code_per_masjid;
DROP INDEX IF EXISTS uq_subjects_slug_per_masjid;

DROP INDEX IF EXISTS idx_subjects_active;
DROP INDEX IF EXISTS gin_subjects_name_trgm;
DROP INDEX IF EXISTS idx_subjects_masjid_alive;
DROP INDEX IF EXISTS idx_subjects_code_ci_alive;
DROP INDEX IF EXISTS idx_subjects_slug_ci_alive;

-- 4) Table (drop seluruh tabel)
DROP TABLE IF EXISTS subjects;

-- 5) Functions
DROP FUNCTION IF EXISTS fn_subjects_touch_updated_at();
DROP FUNCTION IF EXISTS fn_subjects_normalize();

COMMIT;
