-- +migrate Down
-- =========================================
-- DOWN Migration — drop Student Quiz Attempts & Answers
-- =========================================
BEGIN;

-- Drop in reverse dependency order
DROP TABLE IF EXISTS student_quiz_attempt_answers CASCADE;
DROP TABLE IF EXISTS student_quiz_attempts CASCADE;

COMMIT;
