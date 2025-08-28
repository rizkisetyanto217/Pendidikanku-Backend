-- USER LECTURE SESSIONS
DROP INDEX IF EXISTS idx_uls_lecture_for_grade;
DROP INDEX IF EXISTS idx_uls_user_masjid_created_desc;
DROP INDEX IF EXISTS idx_uls_by_session;

DROP TRIGGER IF EXISTS trg_user_lecture_sessions_set_updated_at ON user_lecture_sessions;
DROP FUNCTION IF EXISTS set_user_lecture_sessions_updated_at();

DROP TABLE IF EXISTS user_lecture_sessions;

-- LECTURE SESSIONS
DROP INDEX IF EXISTS idx_ls_start_time;
DROP INDEX IF EXISTS idx_ls_slug_trgm;
DROP INDEX IF EXISTS idx_ls_title_trgm;
DROP INDEX IF EXISTS idx_ls_search_fts;
DROP INDEX IF EXISTS idx_ls_masjid_active_created_desc;
DROP INDEX IF EXISTS idx_ls_lecture_start;
DROP INDEX IF EXISTS idx_ls_teacher_start;
DROP INDEX IF EXISTS idx_ls_masjid_active_start;
DROP INDEX IF EXISTS uq_ls_slug_ci;

DROP TRIGGER IF EXISTS trg_lecture_sessions_set_updated_at ON lecture_sessions;
DROP FUNCTION IF EXISTS set_lecture_sessions_updated_at();

DROP TABLE IF EXISTS lecture_sessions;

-- Extensions dibiarkan (mungkin dipakai objek lain)
