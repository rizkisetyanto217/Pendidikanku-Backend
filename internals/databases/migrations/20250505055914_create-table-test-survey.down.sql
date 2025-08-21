
-- =====================================================================
-- ==============================  DOWN  ================================
-- =====================================================================

BEGIN;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_user_test_exams_touch ON user_test_exams;
DROP TRIGGER IF EXISTS trg_test_exams_touch ON test_exams;
DROP TRIGGER IF EXISTS trg_user_surveys_validate ON user_surveys;
DROP TRIGGER IF EXISTS trg_user_surveys_touch ON user_surveys;
DROP TRIGGER IF EXISTS trg_survey_questions_touch ON survey_questions;

-- Drop indexes
-- user_test_exams
DROP INDEX IF EXISTS idx_user_test_exams_exam_top;
DROP INDEX IF EXISTS idx_user_test_exams_user_created;
DROP INDEX IF EXISTS idx_user_test_exams_exam_grade;

-- test_exams
DROP INDEX IF EXISTS idx_test_exams_name_trgm;
DROP INDEX IF EXISTS idx_test_exams_status_created;

-- user_surveys
DROP INDEX IF EXISTS idx_user_surveys_question_created;
DROP INDEX IF EXISTS idx_user_surveys_user_created;

-- survey_questions
DROP INDEX IF EXISTS idx_survey_questions_text_trgm;
DROP INDEX IF EXISTS idx_survey_questions_created;
DROP INDEX IF EXISTS idx_survey_questions_order;

-- Drop tables
DROP TABLE IF EXISTS user_test_exams;
DROP TABLE IF EXISTS test_exams;
DROP TABLE IF EXISTS user_surveys;
DROP TABLE IF EXISTS survey_questions;

-- Drop functions
DROP FUNCTION IF EXISTS fn_validate_user_survey_answer;
DROP FUNCTION IF EXISTS fn_touch_updated_at_generic;

COMMIT;
