-- =========================================
-- UP Migration — TABLES + STRONG FKs + TRIGGER (no selected_option_id)
-- =========================================
BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- 1) QUIZZES
-- =========================================
CREATE TABLE IF NOT EXISTS quizzes (
  quizzes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quizzes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  quizzes_assessment_id UUID
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  quizzes_title VARCHAR(180) NOT NULL,
  quizzes_description TEXT,
  quizzes_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  quizzes_time_limit_sec INT,

  quizzes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quizzes_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quizzes_deleted_at TIMESTAMPTZ
);

-- Indexes (quizzes)
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_published
  ON quizzes (quizzes_masjid_id, quizzes_is_published)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_assessment
  ON quizzes (quizzes_assessment_id)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_quizzes_created_at
  ON quizzes USING BRIN (quizzes_created_at);

CREATE INDEX IF NOT EXISTS gin_quizzes_title_trgm
  ON quizzes USING GIN (quizzes_title gin_trgm_ops)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_quizzes_desc_trgm
  ON quizzes USING GIN (quizzes_description gin_trgm_ops)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_assessment
  ON quizzes (quizzes_masjid_id, quizzes_assessment_id)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_created_desc
  ON quizzes (quizzes_masjid_id, quizzes_created_at DESC)
  WHERE quizzes_deleted_at IS NULL;

-- =========================================
-- 2) QUIZ QUESTIONS
-- =========================================
CREATE TABLE IF NOT EXISTS quiz_questions (
  quiz_questions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  quiz_questions_quiz_id   UUID NOT NULL
    REFERENCES quizzes(quizzes_id) ON DELETE CASCADE,

  quiz_questions_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  quiz_questions_type VARCHAR(8) NOT NULL
    CHECK (quiz_questions_type IN ('single','essay')),

  quiz_questions_text   TEXT NOT NULL,
  quiz_questions_points NUMERIC(6,2) NOT NULL DEFAULT 1
    CHECK (quiz_questions_points >= 0),

  quiz_questions_answers JSONB,
  quiz_questions_correct CHAR(1)
    CHECK (quiz_questions_correct IN ('A','B','C','D')),

  quiz_questions_explanation TEXT,

  quiz_questions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_questions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_questions_deleted_at TIMESTAMPTZ
);

-- CHECK constraints (idempotent)
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_essay_shape;
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_single_answers_required;
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_single_answers_shape;

ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_essay_shape
  CHECK (
    quiz_questions_type <> 'essay'
    OR (quiz_questions_answers IS NULL AND quiz_questions_correct IS NULL)
  );

ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_single_answers_required
  CHECK (
    quiz_questions_type <> 'single'
    OR quiz_questions_answers IS NOT NULL
  );

ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_single_answers_shape
  CHECK (
    quiz_questions_type <> 'single'
    OR jsonb_typeof(quiz_questions_answers) IN ('object','array')
  );

-- UNIQUE untuk FK komposit (id, quiz_id)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_qq_id_quiz') THEN
    ALTER TABLE quiz_questions
      ADD CONSTRAINT uq_qq_id_quiz UNIQUE (quiz_questions_id, quiz_questions_quiz_id);
  END IF;
END $$;

-- Indexes (quiz_questions)
CREATE INDEX IF NOT EXISTS idx_qq_quiz
  ON quiz_questions (quiz_questions_quiz_id)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_masjid
  ON quiz_questions (quiz_questions_masjid_id)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_masjid_created_desc
  ON quiz_questions (quiz_questions_masjid_id, quiz_questions_created_at DESC)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_qq_created_at
  ON quiz_questions USING BRIN (quiz_questions_created_at);

CREATE INDEX IF NOT EXISTS gin_qq_answers
  ON quiz_questions USING GIN (quiz_questions_answers jsonb_path_ops)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS trgm_qq_text
  ON quiz_questions USING GIN ((LOWER(quiz_questions_text)) gin_trgm_ops)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_qq_tsv
  ON quiz_questions USING GIN (
    (
      setweight(to_tsvector('simple', COALESCE(quiz_questions_text, '')), 'A') ||
      setweight(to_tsvector('simple', COALESCE(quiz_questions_explanation, '')), 'B')
    )
  )
  WHERE quiz_questions_deleted_at IS NULL;
