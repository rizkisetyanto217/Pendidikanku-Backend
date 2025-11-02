-- =========================================================
-- ================          UP              ===============
-- =========================================================

-- Extensions (UUID gen + trigram search)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------
-- Table: lecture_sessions_questions
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_sessions_questions (
  lecture_sessions_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_question TEXT NOT NULL,
  lecture_sessions_question_answers JSONB NOT NULL,
  lecture_sessions_question_correct CHAR(1) NOT NULL
    CHECK (lecture_sessions_question_correct IN ('A','B','C','D')),
  lecture_sessions_question_explanation TEXT,
  lecture_sessions_question_quiz_id UUID
    REFERENCES lecture_sessions_quiz(lecture_sessions_quiz_id) ON DELETE SET NULL,
  lecture_question_exam_id UUID
    REFERENCES lecture_exams(lecture_exam_id) ON DELETE SET NULL,

  -- School wajib
  lecture_sessions_question_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  lecture_sessions_question_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_question_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_question_deleted_at TIMESTAMPTZ NULL,  -- ✅ soft delete

  -- Validasi dasar struktur answers
  CONSTRAINT lecture_sessions_question_answers_type
    CHECK (jsonb_typeof(lecture_sessions_question_answers) IN ('object','array')),

  -- Jika answers object, kunci jawaban benar harus ada
  CONSTRAINT lecture_sessions_question_correct_key_exists
    CHECK (
      jsonb_typeof(lecture_sessions_question_answers) <> 'object'
      OR (lecture_sessions_question_answers ? lecture_sessions_question_correct)
    )
);

-- Full-text search vector (pertanyaan + penjelasan)
ALTER TABLE lecture_sessions_questions
  ADD COLUMN IF NOT EXISTS lecture_sessions_question_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(lecture_sessions_question, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(lecture_sessions_question_explanation, '')), 'B')
  ) STORED;

-- Indexes: hanya baris hidup (deleted_at IS NULL)
CREATE INDEX IF NOT EXISTS idx_ls_questions_quiz_id
  ON lecture_sessions_questions(lecture_sessions_question_quiz_id)
  WHERE lecture_sessions_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_questions_exam_id
  ON lecture_sessions_questions(lecture_question_exam_id)
  WHERE lecture_sessions_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_questions_created_at
  ON lecture_sessions_questions(lecture_sessions_question_created_at)
  WHERE lecture_sessions_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_questions_school_id
  ON lecture_sessions_questions(lecture_sessions_question_school_id)
  WHERE lecture_sessions_question_deleted_at IS NULL;

-- Komposit: school + created_at (sorting terbaru per school)
CREATE INDEX IF NOT EXISTS idx_ls_questions_school_created_desc
  ON lecture_sessions_questions(lecture_sessions_question_school_id, lecture_sessions_question_created_at DESC)
  WHERE lecture_sessions_question_deleted_at IS NULL;

-- GIN untuk JSONB answers
CREATE INDEX IF NOT EXISTS idx_ls_questions_answers_gin
  ON lecture_sessions_questions USING GIN (lecture_sessions_question_answers jsonb_path_ops)
  WHERE lecture_sessions_question_deleted_at IS NULL;

-- FTS GIN index
CREATE INDEX IF NOT EXISTS idx_ls_questions_tsv_gin
  ON lecture_sessions_questions USING GIN (lecture_sessions_question_search_tsv)
  WHERE lecture_sessions_question_deleted_at IS NULL;

-- Trigram
CREATE INDEX IF NOT EXISTS idx_ls_questions_trgm
  ON lecture_sessions_questions USING GIN (LOWER(lecture_sessions_question) gin_trgm_ops)
  WHERE lecture_sessions_question_deleted_at IS NULL;

-- ---------------------------------------------------------
-- Table: lecture_sessions_user_questions
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_sessions_user_questions (
  lecture_sessions_user_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_user_question_answer CHAR(1) NOT NULL
    CHECK (lecture_sessions_user_question_answer IN ('A','B','C','D')),
  lecture_sessions_user_question_is_correct BOOLEAN NOT NULL,
  lecture_sessions_user_question_question_id UUID NOT NULL
    REFERENCES lecture_sessions_questions(lecture_sessions_question_id) ON DELETE CASCADE,

  -- School wajib
  lecture_sessions_user_question_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  lecture_sessions_user_question_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_user_question_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_user_question_deleted_at TIMESTAMPTZ NULL   -- ✅ soft delete
);

-- Indexes: hanya baris hidup
CREATE INDEX IF NOT EXISTS idx_ls_user_questions_question_id
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id)
  WHERE lecture_sessions_user_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_school_id
  ON lecture_sessions_user_questions(lecture_sessions_user_question_school_id)
  WHERE lecture_sessions_user_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_created_at
  ON lecture_sessions_user_questions(lecture_sessions_user_question_created_at)
  WHERE lecture_sessions_user_question_deleted_at IS NULL;

-- Komposit untuk analitik
CREATE INDEX IF NOT EXISTS idx_ls_user_questions_qid_is_correct
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id,
                                    lecture_sessions_user_question_is_correct)
  WHERE lecture_sessions_user_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_qid_answer
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id,
                                    lecture_sessions_user_question_answer)
  WHERE lecture_sessions_user_question_deleted_at IS NULL;