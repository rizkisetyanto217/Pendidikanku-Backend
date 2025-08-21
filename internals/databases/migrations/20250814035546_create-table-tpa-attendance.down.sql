-- =========================================================
-- DROP TABLES (urutan aman)
-- =========================================================

-- 1) Drop user attendance per session (paling bawah)
DROP TABLE IF EXISTS user_class_attendance_sessions CASCADE;

-- 2) Drop class attendance sessions
DROP TABLE IF EXISTS class_attendance_sessions CASCADE;

-- 3) Drop CSST (Class Section Subject Teachers)
-- 1) Hapus trigger validasi tenant & fungsinya
DROP TRIGGER  IF EXISTS trg_class_sec_subj_teachers_validate_tenant ON class_section_subject_teachers;
DROP FUNCTION IF EXISTS fn_class_sec_subj_teachers_validate_tenant();

-- 2) Hapus trigger updated_at & fungsinya
DROP TRIGGER  IF EXISTS set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;
DROP FUNCTION IF EXISTS trg_set_timestamp_class_sec_subj_teachers();

-- 3) Hapus index
DROP INDEX IF EXISTS idx_csst_section_subject_active_alive;
DROP INDEX IF EXISTS idx_csst_masjid_alive;
DROP INDEX IF EXISTS idx_csst_teacher_alive;
DROP INDEX IF EXISTS uq_csst_active_unique;

-- 4) Hapus constraint komposit
ALTER TABLE class_section_subject_teachers
  DROP CONSTRAINT IF EXISTS fk_csst_teacher_membership;

ALTER TABLE class_section_subject_teachers
  DROP CONSTRAINT IF EXISTS fk_csst_section_masjid;

-- 5) Terakhir: drop tabel
DROP TABLE IF EXISTS class_section_subject_teachers;


-- 4) Drop class_subjects
DROP TABLE IF EXISTS class_subjects CASCADE;

-- 5) Drop subjects
DROP TABLE IF EXISTS subjects CASCADE;
