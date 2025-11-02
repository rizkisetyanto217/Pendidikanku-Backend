-- =========================================================
-- MIGRATION DOWN: lecture_exams & user_lecture_exams
-- =========================================================

-- Hapus triggers
DROP TRIGGER IF EXISTS trg_user_lexams_touch ON user_lecture_exams;
DROP TRIGGER IF EXISTS trg_lexams_touch ON lecture_exams;

-- Hapus index user_lecture_exams
DROP INDEX IF EXISTS idx_ulexams_username_trgm;
DROP INDEX IF EXISTS idx_ulexams_school_created_desc;
DROP INDEX IF EXISTS idx_ulexams_user_created_desc;
DROP INDEX IF EXISTS idx_ulexams_exam_user_created_desc;

-- Hapus table anak dulu
DROP TABLE IF EXISTS user_lecture_exams;

-- Hapus index lecture_exams
DROP INDEX IF EXISTS idx_lexams_desc_trgm;
DROP INDEX IF EXISTS idx_lexams_title_trgm;
DROP INDEX IF EXISTS idx_lexams_tsv_gin;
DROP INDEX IF EXISTS idx_lexams_school_created_desc;
DROP INDEX IF EXISTS idx_lexams_lecture_created_desc;
DROP INDEX IF EXISTS ux_lecture_exams_title_per_lecture_ci;

-- Hapus table induk
DROP TABLE IF EXISTS lecture_exams;

-- Hapus trigger functions
DROP FUNCTION IF EXISTS fn_touch_updated_at_user_lexams();
DROP FUNCTION IF EXISTS fn_touch_updated_at_lexams();

-- (Ekstensi pg_trgm/pgcrypto tidak di-drop karena bisa dipakai objek lain)
