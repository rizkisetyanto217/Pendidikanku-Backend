-- =========================================
-- DOWN Migration — DROP INDEXES & TABLES
-- =========================================
BEGIN;

-- =========================================
-- 4) QUIZ ATTEMPT ANSWERS
-- =========================================
DROP INDEX IF EXISTS brin_answers_answered_at;
DROP INDEX IF EXISTS idx_answers_question;
DROP TABLE IF EXISTS quiz_attempt_answers;

-- =========================================
-- 3) QUIZ ATTEMPTS
-- =========================================
DROP INDEX IF EXISTS brin_attempts_created_at;
DROP INDEX IF EXISTS idx_attempts_quiz_active;
DROP INDEX IF EXISTS idx_attempts_student_status;
DROP INDEX IF EXISTS idx_attempts_student;
DROP INDEX IF EXISTS idx_attempts_masjid_quiz;
DROP INDEX IF EXISTS idx_attempts_quiz_student_started_desc;
DROP INDEX IF EXISTS brin_attempts_started_at;
DROP INDEX IF EXISTS idx_attempts_status;
DROP INDEX IF EXISTS idx_attempts_quiz_student;
DROP TABLE IF EXISTS quiz_attempts;

-- =========================================
-- 2) QUIZ ITEMS
-- =========================================
DROP INDEX IF EXISTS idx_quiz_items_quiz_essay;
DROP INDEX IF EXISTS idx_quiz_items_quiz_question;
DROP INDEX IF EXISTS idx_quiz_items_type;
DROP INDEX IF EXISTS idx_quiz_items_question;
DROP INDEX IF EXISTS idx_quiz_items_quiz;
DROP INDEX IF EXISTS uq_essay_single_row_per_question;
DROP INDEX IF EXISTS uq_question_option_pair;
DROP INDEX IF EXISTS uq_single_correct_per_question;
-- (opsional) constraint akan ikut terhapus saat DROP TABLE:
-- ALTER TABLE IF EXISTS quiz_items DROP CONSTRAINT IF EXISTS ck_quiz_items_shape;
DROP TABLE IF EXISTS quiz_items;

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

-- =========================================
-- Extensions (umumnya jangan di-drop; uncomment jika yakin tidak dipakai)
-- =========================================
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
