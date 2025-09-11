-- =========================================
-- DOWN Migration — revert TABLES, FKs, TRIGGER, INDEXES
-- =========================================
BEGIN;

-- -------------------------
-- 4) USER QUIZ ATTEMPT ANSWERS
-- -------------------------
-- Drop trigger & function dulu (biar aman di DB yg strict)
DROP TRIGGER IF EXISTS trg_uqaa_fill_quiz_id ON user_quiz_attempt_answers;
DROP FUNCTION IF EXISTS uqaa_fill_quiz_id() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS brin_uqaa_answered_at;
DROP INDEX IF EXISTS idx_uqaa_quiz;
DROP INDEX IF EXISTS idx_uqaa_attempt;
DROP INDEX IF EXISTS idx_uqaa_question;

-- Drop table (ikut ngedrop FK komposit)
DROP TABLE IF EXISTS user_quiz_attempt_answers;

-- -------------------------
-- 3) USER QUIZ ATTEMPTS
-- -------------------------
-- Drop indexes
DROP INDEX IF EXISTS brin_uqa_created_at;
DROP INDEX IF EXISTS idx_uqa_quiz_active;
DROP INDEX IF EXISTS idx_uqa_student_status;
DROP INDEX IF EXISTS idx_uqa_student;
DROP INDEX IF EXISTS idx_uqa_masjid_quiz;
DROP INDEX IF EXISTS idx_uqa_quiz_student_started_desc;
DROP INDEX IF EXISTS brin_uqa_started_at;
DROP INDEX IF EXISTS idx_uqa_status;
DROP INDEX IF EXISTS idx_uqa_quiz_student;

-- (Unique constraint uq_uqa_id_quiz akan ikut hilang saat table di-drop)
DROP TABLE IF EXISTS user_quiz_attempts;

-- -------------------------
-- 2) QUIZ QUESTIONS
-- -------------------------
-- Drop indexes
DROP INDEX IF EXISTS gin_qq_tsv;
DROP INDEX IF EXISTS trgm_qq_text;
DROP INDEX IF EXISTS gin_qq_answers;
DROP INDEX IF EXISTS brin_qq_created_at;
DROP INDEX IF EXISTS idx_qq_masjid_created_desc;
DROP INDEX IF EXISTS idx_qq_masjid;
DROP INDEX IF EXISTS idx_qq_quiz;

-- (Constraints ck_qq_* & unique uq_qq_id_quiz ikut hilang saat table di-drop)
DROP TABLE IF EXISTS quiz_questions;

-- -------------------------
-- 1) QUIZZES
-- -------------------------
-- Drop indexes
DROP INDEX IF EXISTS idx_quizzes_masjid_created_desc;
DROP INDEX IF EXISTS idx_quizzes_masjid_assessment;
DROP INDEX IF EXISTS gin_quizzes_desc_trgm;
DROP INDEX IF EXISTS gin_quizzes_title_trgm;
DROP INDEX IF EXISTS brin_quizzes_created_at;
DROP INDEX IF EXISTS idx_quizzes_assessment;
DROP INDEX IF EXISTS idx_quizzes_masjid_published;

DROP TABLE IF EXISTS quizzes;

-- -------------------------
-- (Optional) drop extensions kalau memang dibuat khusus untuk modul ini
-- !!! HATI-HATI: hanya kalau yakin tidak dipakai modul lain !!!
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
