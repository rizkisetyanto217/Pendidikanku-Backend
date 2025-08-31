-- =========================================================
-- DOWN: Subjects, Class-Subjects, CSST
-- Bersih & idempotent (aman jika objek tidak ada)
-- =========================================================
BEGIN;

-- =========================
-- CLASS SECTION SUBJECT TEACHERS
-- =========================
-- Triggers & functions
DROP TRIGGER IF EXISTS trg_csst_validate_consistency ON class_section_subject_teachers;
DROP TRIGGER IF EXISTS set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;

DROP FUNCTION IF EXISTS fn_csst_validate_consistency() CASCADE;
DROP FUNCTION IF EXISTS trg_set_timestamp_class_sec_subj_teachers() CASCADE;

-- Indexes
DROP INDEX IF EXISTS uq_csst_active_by_cs;
DROP INDEX IF EXISTS idx_csst_by_cs_alive;
DROP INDEX IF EXISTS idx_csst_by_section_alive;
DROP INDEX IF EXISTS idx_csst_by_teacher_alive;
DROP INDEX IF EXISTS idx_csst_by_masjid_alive;

-- Tabel
DROP TABLE IF EXISTS class_section_subject_teachers CASCADE;

-- =========================
-- CLASS_SUBJECTS
-- =========================
-- Triggers & functions
DROP TRIGGER IF EXISTS set_timestamptz_class_subjects ON class_subjects;
DROP FUNCTION IF EXISTS trg_set_timestamptz_class_subjects() CASCADE;

-- Indexes & constraints khusus
DROP INDEX IF EXISTS uq_class_subjects_by_term;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='uq_class_subjects_id_masjid') THEN
    ALTER TABLE class_subjects DROP CONSTRAINT uq_class_subjects_id_masjid;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_class_masjid_pair') THEN
    ALTER TABLE class_subjects DROP CONSTRAINT fk_cs_class_masjid_pair;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_term') THEN
    ALTER TABLE class_subjects DROP CONSTRAINT fk_cs_term;
  END IF;
END$$;

-- Tabel
DROP TABLE IF EXISTS class_subjects CASCADE;

-- =========================
-- SUBJECTS
-- =========================
-- Triggers & functions
DROP TRIGGER IF EXISTS trg_subjects_touch_updated_at ON subjects;
DROP TRIGGER IF EXISTS trg_subjects_normalize ON subjects;

DROP FUNCTION IF EXISTS fn_subjects_touch_updated_at() CASCADE;
DROP FUNCTION IF EXISTS fn_subjects_normalize() CASCADE;

-- Indexes
DROP INDEX IF EXISTS uq_subjects_code_per_masjid;
DROP INDEX IF EXISTS uq_subjects_slug_per_masjid;
DROP INDEX IF EXISTS idx_subjects_active;
DROP INDEX IF EXISTS gin_subjects_name_trgm;
DROP INDEX IF EXISTS idx_subjects_masjid_alive;

-- (Optional) drop check constraints if needed (akan ikut terhapus saat DROP TABLE)
-- DO $$
-- BEGIN
--   IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_code_not_blank') THEN
--     ALTER TABLE subjects DROP CONSTRAINT chk_subjects_code_not_blank;
--   END IF;
--   IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_slug_not_blank') THEN
--     ALTER TABLE subjects DROP CONSTRAINT chk_subjects_slug_not_blank;
--   END IF;
-- END$$;

-- Tabel
DROP TABLE IF EXISTS subjects CASCADE;

COMMIT;
