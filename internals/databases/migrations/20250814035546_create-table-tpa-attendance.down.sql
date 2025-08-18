-- =========================================================
-- DROP TABLES (urutan aman)
-- =========================================================

-- 1) Drop user attendance per session (paling bawah)
DROP TABLE IF EXISTS user_class_attendance_sessions CASCADE;

-- 2) Drop class attendance sessions
DROP TABLE IF EXISTS class_attendance_sessions CASCADE;

-- 3) Drop CSST (Class Section Subject Teachers)
DROP TABLE IF EXISTS class_section_subject_teachers CASCADE;

-- 4) Drop class_subjects
DROP TABLE IF EXISTS class_subjects CASCADE;

-- 5) Drop subjects
DROP TABLE IF EXISTS subjects CASCADE;
