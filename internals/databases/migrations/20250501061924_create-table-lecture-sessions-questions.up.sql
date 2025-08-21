-- =========================================================
-- ================          UP              ===============
-- =========================================================

-- Extensions (UUID gen + trigram search)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------
-- Touch updated_at helpers
-- ---------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_touch_updated_at_ls_questions()
RETURNS TRIGGER AS $$
BEGIN
  NEW.lecture_sessions_question_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_updated_at_ls_user_questions()
RETURNS TRIGGER AS $$
BEGIN
  NEW.lecture_sessions_user_question_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

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

  -- Masjid wajib
  lecture_sessions_question_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  lecture_sessions_question_updated_at TIMESTAMP,

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

-- Indexes: FK / waktu / masjid
CREATE INDEX IF NOT EXISTS idx_ls_questions_quiz_id
  ON lecture_sessions_questions(lecture_sessions_question_quiz_id);

CREATE INDEX IF NOT EXISTS idx_ls_questions_exam_id
  ON lecture_sessions_questions(lecture_question_exam_id);

CREATE INDEX IF NOT EXISTS idx_ls_questions_created_at
  ON lecture_sessions_questions(lecture_sessions_question_created_at);

CREATE INDEX IF NOT EXISTS idx_ls_questions_masjid_id
  ON lecture_sessions_questions(lecture_sessions_question_masjid_id);

-- Komposit: masjid + created_at (sorting terbaru per masjid)
CREATE INDEX IF NOT EXISTS idx_ls_questions_masjid_created_desc
  ON lecture_sessions_questions(lecture_sessions_question_masjid_id, lecture_sessions_question_created_at DESC);

-- GIN untuk JSONB answers (existence / containment)
CREATE INDEX IF NOT EXISTS idx_ls_questions_answers_gin
  ON lecture_sessions_questions USING GIN (lecture_sessions_question_answers jsonb_path_ops);

-- FTS GIN index
CREATE INDEX IF NOT EXISTS idx_ls_questions_tsv_gin
  ON lecture_sessions_questions USING GIN (lecture_sessions_question_search_tsv);

-- Trigram untuk ILIKE '%...%' di kolom pertanyaan (opsional tapi praktis)
CREATE INDEX IF NOT EXISTS idx_ls_questions_trgm
  ON lecture_sessions_questions USING GIN (LOWER(lecture_sessions_question) gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_ls_questions_touch ON lecture_sessions_questions;
CREATE TRIGGER trg_ls_questions_touch
BEFORE UPDATE ON lecture_sessions_questions
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_ls_questions();

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

  -- Masjid wajib
  lecture_sessions_user_question_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_user_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  lecture_sessions_user_question_updated_at TIMESTAMP
);

-- Indexes: FK / masjid / waktu
CREATE INDEX IF NOT EXISTS idx_ls_user_questions_question_id
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id);

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_masjid_id
  ON lecture_sessions_user_questions(lecture_sessions_user_question_masjid_id);

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_created_at
  ON lecture_sessions_user_questions(lecture_sessions_user_question_created_at);

-- Komposit untuk agregasi & analitik jawaban per soal
CREATE INDEX IF NOT EXISTS idx_ls_user_questions_qid_is_correct
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id,
                                    lecture_sessions_user_question_is_correct);

CREATE INDEX IF NOT EXISTS idx_ls_user_questions_qid_answer
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id,
                                    lecture_sessions_user_question_answer);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_ls_user_questions_touch ON lecture_sessions_user_questions;
CREATE TRIGGER trg_ls_user_questions_touch
BEFORE UPDATE ON lecture_sessions_user_questions
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_ls_user_questions();