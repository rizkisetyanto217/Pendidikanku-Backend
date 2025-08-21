BEGIN;
-- =====================================================================
-- ============================  DOWN  =================================
-- =====================================================================

-- Drop triggers
DROP TRIGGER IF EXISTS trg_validate_user_questionnaire_answer ON user_questionnaire_answers;

-- Drop indexes (answers)
DROP INDEX IF EXISTS idx_user_questionnaire_event_ref;
DROP INDEX IF EXISTS idx_user_questionnaire_lecture_ref;
DROP INDEX IF EXISTS idx_user_questionnaire_user_created;
DROP INDEX IF EXISTS idx_user_questionnaire_question_created;
DROP INDEX IF EXISTS idx_user_questionnaire_ref_created;
DROP INDEX IF EXISTS idx_user_questionnaire_answer_trgm;

-- Drop indexes (questions)
DROP INDEX IF EXISTS idx_questions_type_created;
DROP INDEX IF EXISTS idx_questions_scope_lecture;
DROP INDEX IF EXISTS idx_questions_scope_event;
DROP INDEX IF EXISTS idx_questions_scope_general;
DROP INDEX IF EXISTS idx_questionnaire_question_text_trgm;

-- Drop tables
DROP TABLE IF EXISTS user_questionnaire_answers;
DROP TABLE IF EXISTS questionnaire_questions;

-- Drop function
DROP FUNCTION IF EXISTS fn_validate_user_questionnaire_answer;

COMMIT;