
-- =========================================================
-- ================          DOWN            ===============
-- =========================================================

-- Drop triggers
DROP TRIGGER IF EXISTS trg_ls_user_questions_touch ON lecture_sessions_user_questions;
DROP TRIGGER IF EXISTS trg_ls_questions_touch ON lecture_sessions_questions;

-- Drop functions
DROP FUNCTION IF EXISTS fn_touch_updated_at_ls_user_questions();
DROP FUNCTION IF EXISTS fn_touch_updated_at_ls_questions();

-- Drop indexes (user questions)
DROP INDEX IF EXISTS idx_ls_user_questions_qid_answer;
DROP INDEX IF EXISTS idx_ls_user_questions_qid_is_correct;
DROP INDEX IF EXISTS idx_ls_user_questions_created_at;
DROP INDEX IF EXISTS idx_ls_user_questions_masjid_id;
DROP INDEX IF EXISTS idx_ls_user_questions_question_id;

-- Drop table (user questions)
DROP TABLE IF EXISTS lecture_sessions_user_questions;

-- Drop indexes (questions)
DROP INDEX IF EXISTS idx_ls_questions_trgm;
DROP INDEX IF EXISTS idx_ls_questions_tsv_gin;
DROP INDEX IF EXISTS idx_ls_questions_answers_gin;
DROP INDEX IF EXISTS idx_ls_questions_masjid_created_desc;
DROP INDEX IF EXISTS idx_ls_questions_masjid_id;
DROP INDEX IF EXISTS idx_ls_questions_created_at;
DROP INDEX IF EXISTS idx_ls_questions_exam_id;
DROP INDEX IF EXISTS idx_ls_questions_quiz_id;

-- Drop table (questions)
DROP TABLE IF EXISTS lecture_sessions_questions;