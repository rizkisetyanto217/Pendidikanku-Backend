-- +migrate Down
-- =========================================
-- DOWN Migration — drop Student Quiz Attempts & Answers
-- =========================================
BEGIN;

DROP TABLE IF EXISTS student_quiz_attempts CASCADE;

COMMIT;
