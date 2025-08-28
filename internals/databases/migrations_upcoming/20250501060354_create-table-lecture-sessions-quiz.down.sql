-- =========================================================
-- MIGRATION DOWN: lecture_sessions_quiz & user_lecture_sessions_quiz
-- =========================================================

-- Hapus triggers
DROP TRIGGER IF EXISTS trg_user_lsquiz_touch ON user_lecture_sessions_quiz;
DROP TRIGGER IF EXISTS trg_lsquiz_touch ON lecture_sessions_quiz;

-- Hapus index user_lecture_sessions_quiz
DROP INDEX IF EXISTS idx_ulsq_masjid_created_desc;
DROP INDEX IF EXISTS idx_ulsq_user_created_desc;
DROP INDEX IF EXISTS idx_ulsq_session_user;
DROP INDEX IF EXISTS idx_ulsq_quser_created_desc;

-- Drop child table dulu
DROP TABLE IF EXISTS user_lecture_sessions_quiz;

-- Hapus index lecture_sessions_quiz
DROP INDEX IF EXISTS idx_lsquiz_desc_trgm;
DROP INDEX IF EXISTS idx_lsquiz_title_trgm;
DROP INDEX IF EXISTS idx_lsquiz_tsv_gin;
DROP INDEX IF EXISTS idx_lsquiz_masjid_created_desc;
DROP INDEX IF EXISTS idx_lsquiz_session_created_desc;
DROP INDEX IF EXISTS ux_lsquiz_per_session_title_ci;

-- Drop parent table
DROP TABLE IF EXISTS lecture_sessions_quiz;

-- Hapus trigger functions
DROP FUNCTION IF EXISTS fn_touch_updated_at_user_lsquiz();
DROP FUNCTION IF EXISTS fn_touch_updated_at_lsquiz();

-- (Ekstensi pg_trgm/pgcrypto tidak di-drop karena mungkin dipakai objek lain)
