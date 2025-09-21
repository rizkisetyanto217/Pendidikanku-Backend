-- +migrate Down
BEGIN;

-- =========================
-- 2) QUIZ QUESTIONS (child)
-- =========================

-- Drop indexes (if exist)
DROP INDEX IF EXISTS idx_qq_quiz;
DROP INDEX IF EXISTS idx_qq_masjid;
DROP INDEX IF EXISTS idx_qq_masjid_created_desc;
DROP INDEX IF EXISTS brin_qq_created_at;
DROP INDEX IF EXISTS gin_qq_answers;
DROP INDEX IF EXISTS trgm_qq_text;
DROP INDEX IF EXISTS gin_qq_tsv;

-- Drop constraints (idempotent)
ALTER TABLE IF EXISTS quiz_questions
  DROP CONSTRAINT IF EXISTS ck_qq_essay_shape,
  DROP CONSTRAINT IF EXISTS ck_qq_single_answers_required,
  DROP CONSTRAINT IF EXISTS ck_qq_single_answers_shape,
  DROP CONSTRAINT IF EXISTS uq_qq_id_quiz;

-- Drop table
DROP TABLE IF EXISTS quiz_questions;

-- =========================
-- 1) QUIZZES (parent)
-- =========================

-- Drop indexes (if exist)
DROP INDEX IF EXISTS uq_quizzes_slug_per_tenant_alive;
DROP INDEX IF EXISTS gin_quizzes_slug_trgm_alive;
DROP INDEX IF EXISTS uq_quizzes_id_tenant;
DROP INDEX IF EXISTS idx_quizzes_masjid_published;
DROP INDEX IF EXISTS idx_quizzes_assessment;
DROP INDEX IF EXISTS brin_quizzes_created_at;
DROP INDEX IF EXISTS gin_quizzes_title_trgm;
DROP INDEX IF EXISTS gin_quizzes_desc_trgm;
DROP INDEX IF EXISTS idx_quizzes_masjid_assessment;
DROP INDEX IF EXISTS idx_quizzes_masjid_created_desc;

-- Drop table
DROP TABLE IF EXISTS quizzes;

COMMIT;
