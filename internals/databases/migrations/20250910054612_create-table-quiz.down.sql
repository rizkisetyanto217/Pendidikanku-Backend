-- =========================================
-- DOWN Migration — DROP INDEXES & TABLES
-- =========================================
BEGIN;

-- =========================================
-- 4) USER QUIZ ATTEMPT ANSWERS
-- =========================================
DROP INDEX IF EXISTS brin_user_answers_answered_at;
DROP INDEX IF EXISTS idx_user_answers_question;
DROP TABLE IF EXISTS user_quiz_attempt_answers;

-- =========================================
-- 3) USER QUIZ ATTEMPTS
-- =========================================
DROP INDEX IF EXISTS brin_uqa_created_at;
DROP INDEX IF EXISTS idx_uqa_quiz_active;
DROP INDEX IF EXISTS idx_uqa_student_status;
DROP INDEX IF EXISTS idx_uqa_student;
DROP INDEX IF EXISTS idx_uqa_masjid_quiz;
DROP INDEX IF EXISTS idx_uqa_quiz_student_started_desc;
DROP INDEX IF EXISTS brin_uqa_started_at;
DROP INDEX IF EXISTS idx_uqa_status;
DROP INDEX IF EXISTS idx_uqa_quiz_student;
DROP TABLE IF EXISTS user_quiz_attempts;

-- =========================================
-- 2) QUIZ QUESTIONS
-- =========================================
DROP INDEX IF EXISTS gin_qq_tsv;
DROP INDEX IF EXISTS trgm_qq_text;
DROP INDEX IF EXISTS gin_qq_answers;
DROP INDEX IF EXISTS brin_qq_created_at;
DROP INDEX IF EXISTS idx_qq_masjid_created_desc;
DROP INDEX IF EXISTS idx_qq_masjid;
DROP INDEX IF EXISTS idx_qq_quiz;
DROP TABLE IF EXISTS quiz_questions;

-- =========================================
-- 1) QUIZZES
-- =========================================
DROP INDEX IF EXISTS idx_quizzes_masjid_created_desc;
DROP INDEX IF EXISTS idx_quizzes_masjid_assessment;
DROP INDEX IF EXISTS gin_quizzes_desc_trgm;
DROP INDEX IF EXISTS gin_quizzes_title_trgm;
DROP INDEX IF EXISTS brin_quizzes_created_at;
DROP INDEX IF EXISTS idx_quizzes_assessment;
DROP INDEX IF EXISTS idx_quizzes_masjid_published;
DROP TABLE IF EXISTS quizzes;

-- (Extensions biasanya dibiarkan)
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
