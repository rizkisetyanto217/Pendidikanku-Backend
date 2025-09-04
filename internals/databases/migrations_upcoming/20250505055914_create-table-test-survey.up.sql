-- =====================================================================
-- Migration: survey_questions, user_surveys, test_exams, user_test_exams
-- DB: PostgreSQL
-- =====================================================================

BEGIN;

-- -------------------------------------------------
-- Extensions (idempotent)
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- (Opsional) Validasi jawaban user pada user_surveys
-- - Jika survey_question_answer (opsi) TIDAK NULL, maka jawaban user harus salah satu elemen opsi tsb
CREATE OR REPLACE FUNCTION fn_validate_user_survey_answer()
RETURNS TRIGGER AS $$
DECLARE
  allowed TEXT[];
BEGIN
  SELECT survey_question_answer
  INTO allowed
  FROM survey_questions
  WHERE survey_question_id = NEW.user_survey_question_id;

  -- allowed NULL artinya open-ended (bebas)
  IF allowed IS NOT NULL AND cardinality(allowed) > 0 THEN
    IF NOT (NEW.user_survey_answer = ANY(allowed)) THEN
      RAISE EXCEPTION 'Answer "%" is not in allowed options: %', NEW.user_survey_answer, allowed;
    END IF;
  END IF;

  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =====================================================================
-- ==============================  UP  ==================================
-- =====================================================================

-- =============================
-- 📋 survey_questions
-- =============================
CREATE TABLE IF NOT EXISTS survey_questions (
  survey_question_id           SERIAL PRIMARY KEY,
  survey_question_text         TEXT NOT NULL,
  survey_question_answer       TEXT[] DEFAULT NULL,        -- NULL = open-ended
  survey_question_order_index  INT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Urutan unik opsional (aktifkan jika perlu urutan unik secara global)
  -- CONSTRAINT uq_survey_questions_order UNIQUE (survey_question_order_index)
  -- Atau, jika per "survey"/section nanti, tambahkan kolom scope lalu unique (scope, order)
  CONSTRAINT chk_order_index_nonneg CHECK (survey_question_order_index >= 0)
);

-- Indexing (survey_questions)
CREATE INDEX IF NOT EXISTS idx_survey_questions_order
  ON survey_questions(survey_question_order_index);

CREATE INDEX IF NOT EXISTS idx_survey_questions_created
  ON survey_questions(created_at DESC);

-- Search teks pertanyaan (ILIKE/%%) - cepat dengan trigram
CREATE INDEX IF NOT EXISTS idx_survey_questions_text_trgm
  ON survey_questions USING GIN (survey_question_text gin_trgm_ops);


-- =============================
-- 🧾 user_surveys (jawaban user)
-- =============================
CREATE TABLE IF NOT EXISTS user_surveys (
  user_survey_id            SERIAL PRIMARY KEY,
  user_survey_user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_survey_question_id   INT  NOT NULL REFERENCES survey_questions(survey_question_id) ON DELETE CASCADE,
  user_survey_answer        TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Satu user satu jawaban per question (hindari duplikasi)
  CONSTRAINT uq_user_surveys_user_question UNIQUE (user_survey_user_id, user_survey_question_id)
);

-- Trigger: touch + validasi jawaban
DROP TRIGGER IF EXISTS trg_user_surveys_touch ON user_surveys;
CREATE TRIGGER trg_user_surveys_touch
BEFORE UPDATE ON user_surveys
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at_generic();

DROP TRIGGER IF EXISTS trg_user_surveys_validate ON user_surveys;
CREATE TRIGGER trg_user_surveys_validate
BEFORE INSERT OR UPDATE ON user_surveys
FOR EACH ROW
EXECUTE FUNCTION fn_validate_user_survey_answer();

-- Indexing (user_surveys)
-- Aktivitas user + waktu (profil user, histori)
CREATE INDEX IF NOT EXISTS idx_user_surveys_user_created
  ON user_surveys(user_survey_user_id, created_at DESC);

-- Rekap per question + waktu
CREATE INDEX IF NOT EXISTS idx_user_surveys_question_created
  ON user_surveys(user_survey_question_id, created_at DESC);


-- =============================
-- 🧪 test_exams
-- =============================
CREATE TABLE IF NOT EXISTS test_exams (
  test_exam_id     SERIAL PRIMARY KEY,
  test_exam_name   VARCHAR(50) NOT NULL,
  test_exam_status VARCHAR(10) NOT NULL DEFAULT 'pending'
    CHECK (test_exam_status IN ('active', 'pending', 'archived')),

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger: touch updated_at
DROP TRIGGER IF EXISTS trg_test_exams_touch ON test_exams;
CREATE TRIGGER trg_test_exams_touch
BEFORE UPDATE ON test_exams
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Indexing (test_exams)
-- Status-first listing (dashboard: filter by status, urut terbaru)
CREATE INDEX IF NOT EXISTS idx_test_exams_status_created
  ON test_exams(test_exam_status, created_at DESC);

-- Search name
CREATE INDEX IF NOT EXISTS idx_test_exams_name_trgm
  ON test_exams USING GIN (test_exam_name gin_trgm_ops);


-- =============================
-- 👤 user_test_exams (hasil)
-- =============================
CREATE TABLE IF NOT EXISTS user_test_exams (
  user_test_exam_id             SERIAL PRIMARY KEY,
  user_test_exam_user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_test_exam_test_exam_id   INT  NOT NULL REFERENCES test_exams(test_exam_id) ON DELETE CASCADE,

  user_test_exam_percentage_grade INTEGER NOT NULL DEFAULT 0 CHECK (user_test_exam_percentage_grade BETWEEN 0 AND 100),
  user_test_exam_time_duration     INTEGER NOT NULL DEFAULT 0 CHECK (user_test_exam_time_duration >= 0),

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Satu hasil per (user, exam). Jika ingin menyimpan banyak attempt, tambahkan kolom attempt_no.
  CONSTRAINT uq_user_test_exams_user_exam UNIQUE (user_test_exam_user_id, user_test_exam_test_exam_id)
);


-- Indexing (user_test_exams)
-- Rekap cepat per exam
CREATE INDEX IF NOT EXISTS idx_user_test_exams_exam_grade
  ON user_test_exams(user_test_exam_test_exam_id, user_test_exam_percentage_grade DESC);

-- Aktivitas user (riwayat)
CREATE INDEX IF NOT EXISTS idx_user_test_exams_user_created
  ON user_test_exams(user_test_exam_user_id, created_at DESC);

-- Leaderboard cepat (top skor) per exam
CREATE INDEX IF NOT EXISTS idx_user_test_exams_exam_top
  ON user_test_exams(user_test_exam_test_exam_id, user_test_exam_percentage_grade DESC, user_test_exam_time_duration ASC);


COMMIT;