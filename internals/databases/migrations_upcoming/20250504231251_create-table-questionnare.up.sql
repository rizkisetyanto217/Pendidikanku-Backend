-- =====================================================================
-- Migration: questionnaire_questions & user_questionnaire_answers
-- DB: PostgreSQL
-- =====================================================================

BEGIN;

-- -------------------------------------------------
-- Extensions (idempotent)
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- -------------------------------------------------
-- Functions: Validations
-- -------------------------------------------------

-- ✅ Validasi konsistensi scope & options pada QUESTIONS via CHECK sudah dibuat di DDL,
--   namun untuk ANSWERS butuh trigger untuk:
--   1) memastikan reference_id exist sesuai type (lecture/event)
--   2) jika question type = choice(3), maka answer ∈ options

CREATE OR REPLACE FUNCTION fn_validate_user_questionnaire_answer()
RETURNS TRIGGER AS $$
DECLARE
  q_type INT;
  q_options TEXT[];
  ref_exists BOOLEAN;
BEGIN
  -- Ambil tipe & opsi dari pertanyaan
  SELECT questionnaire_question_type, questionnaire_question_options
  INTO q_type, q_options
  FROM questionnaire_questions
  WHERE questionnaire_question_id = NEW.user_questionnaire_question_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Question not found for id=%', NEW.user_questionnaire_question_id
      USING ERRCODE = 'foreign_key_violation';
  END IF;

  -- 1) Validasi reference existence sesuai type
  IF NEW.user_questionnaire_type = 1 THEN
    -- lecture
    SELECT EXISTS(
      SELECT 1 FROM lecture_sessions
      WHERE lecture_session_id = NEW.user_questionnaire_reference_id
    ) INTO ref_exists;
    IF NOT ref_exists THEN
      RAISE EXCEPTION 'Reference % not found in lecture_sessions for type=1', NEW.user_questionnaire_reference_id;
    END IF;
  ELSIF NEW.user_questionnaire_type = 2 THEN
    -- event
    SELECT EXISTS(
      SELECT 1 FROM events
      WHERE event_id = NEW.user_questionnaire_reference_id
    ) INTO ref_exists;
    IF NOT ref_exists THEN
      RAISE EXCEPTION 'Reference % not found in events for type=2', NEW.user_questionnaire_reference_id;
    END IF;
  ELSE
    RAISE EXCEPTION 'Invalid user_questionnaire_type=% (expected 1 or 2)', NEW.user_questionnaire_type;
  END IF;

  -- 2) Jika pertanyaan adalah pilihan (type=3), jawabannya harus ada di options
  IF q_type = 3 THEN
    IF q_options IS NULL OR cardinality(q_options) = 0 THEN
      RAISE EXCEPTION 'Question options are empty for a choice type question';
    END IF;

    IF NOT (NEW.user_questionnaire_answer = ANY(q_options)) THEN
      RAISE EXCEPTION 'Answer "%" is not one of the allowed options: %',
        NEW.user_questionnaire_answer, q_options;
    END IF;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================================
-- ============================  UP  ====================================
-- =====================================================================

-- =============================
-- 📋 Tabel Pertanyaan Kuisioner
-- =============================
CREATE TABLE IF NOT EXISTS questionnaire_questions (
  questionnaire_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  questionnaire_question_text TEXT NOT NULL,

  -- 1=rating, 2=text, 3=choice
  questionnaire_question_type INT NOT NULL CHECK (questionnaire_question_type IN (1, 2, 3)),

  -- wajib terisi hanya jika type=3 (choice), selain itu harus NULL
  questionnaire_question_options TEXT[],

  -- Scope referensi
  questionnaire_question_event_id UUID REFERENCES events(event_id) ON DELETE CASCADE,
  questionnaire_question_lecture_session_id UUID REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,

  -- 1=general, 2=event, 3=lecture
  questionnaire_question_scope INT NOT NULL DEFAULT 1 CHECK (questionnaire_question_scope IN (1, 2, 3)),

  questionnaire_question_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- ✅ CHECK konsistensi SCOPE ↔️ referensi
  CONSTRAINT chk_question_scope_refs CHECK (
    (questionnaire_question_scope = 1 AND questionnaire_question_event_id IS NULL AND questionnaire_question_lecture_session_id IS NULL) OR
    (questionnaire_question_scope = 2 AND questionnaire_question_event_id IS NOT NULL AND questionnaire_question_lecture_session_id IS NULL) OR
    (questionnaire_question_scope = 3 AND questionnaire_question_event_id IS NULL AND questionnaire_question_lecture_session_id IS NOT NULL)
  ),

  -- ✅ CHECK konsistensi opsi untuk type=3
  CONSTRAINT chk_question_options_by_type CHECK (
    (questionnaire_question_type = 3 AND questionnaire_question_options IS NOT NULL AND cardinality(questionnaire_question_options) > 0)
    OR
    (questionnaire_question_type <> 3 AND questionnaire_question_options IS NULL)
  )
);

-- Indexing (questions)
-- Search text cepat (ILIKE/%%) pada pertanyaan aktif
CREATE INDEX IF NOT EXISTS idx_questionnaire_question_text_trgm
  ON questionnaire_questions USING GIN (questionnaire_question_text gin_trgm_ops);

-- Scope-based partial indexes
CREATE INDEX IF NOT EXISTS idx_questions_scope_general
  ON questionnaire_questions(questionnaire_question_created_at DESC)
  WHERE questionnaire_question_scope = 1;

CREATE INDEX IF NOT EXISTS idx_questions_scope_event
  ON questionnaire_questions(questionnaire_question_event_id, questionnaire_question_created_at DESC)
  WHERE questionnaire_question_scope = 2;

CREATE INDEX IF NOT EXISTS idx_questions_scope_lecture
  ON questionnaire_questions(questionnaire_question_lecture_session_id, questionnaire_question_created_at DESC)
  WHERE questionnaire_question_scope = 3;

-- Tipe pertanyaan + waktu (moderasi/filter cepat)
CREATE INDEX IF NOT EXISTS idx_questions_type_created
  ON questionnaire_questions(questionnaire_question_type, questionnaire_question_created_at DESC);


-- =============================
-- 🧾 Tabel Jawaban Kuisioner User
-- =============================
CREATE TABLE IF NOT EXISTS user_questionnaire_answers (
  user_questionnaire_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_questionnaire_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- 1=lecture, 2=event
  user_questionnaire_type INT NOT NULL CHECK (user_questionnaire_type IN (1, 2)),

  -- ID dari lecture_session atau event (divalidasi trigger)
  user_questionnaire_reference_id UUID NOT NULL,

  user_questionnaire_question_id UUID NOT NULL REFERENCES questionnaire_questions(questionnaire_question_id) ON DELETE CASCADE,

  user_questionnaire_answer TEXT NOT NULL,
  user_questionnaire_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- ✅ Prevent duplicate answer per (user, reference, question)
  CONSTRAINT uq_user_answer UNIQUE (user_questionnaire_user_id, user_questionnaire_reference_id, user_questionnaire_question_id)
);

-- Triggers (answers)
DROP TRIGGER IF EXISTS trg_validate_user_questionnaire_answer ON user_questionnaire_answers;
CREATE TRIGGER trg_validate_user_questionnaire_answer
BEFORE INSERT OR UPDATE ON user_questionnaire_answers
FOR EACH ROW
EXECUTE FUNCTION fn_validate_user_questionnaire_answer();

-- Indexing (answers)
-- Query umum: by reference + waktu
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_ref_created
  ON user_questionnaire_answers(user_questionnaire_reference_id, user_questionnaire_created_at DESC);

-- Rekap cepat per pertanyaan
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_question_created
  ON user_questionnaire_answers(user_questionnaire_question_id, user_questionnaire_created_at DESC);

-- Aktivitas user
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_user_created
  ON user_questionnaire_answers(user_questionnaire_user_id, user_questionnaire_created_at DESC);

-- Partial index by type (lecture/event) untuk report per kanal
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_lecture_ref
  ON user_questionnaire_answers(user_questionnaire_reference_id)
  WHERE user_questionnaire_type = 1;

CREATE INDEX IF NOT EXISTS idx_user_questionnaire_event_ref
  ON user_questionnaire_answers(user_questionnaire_reference_id)
  WHERE user_questionnaire_type = 2;

-- Search jawaban teks (opsional; aktifkan bila perlu)
-- CREATE INDEX IF NOT EXISTS idx_user_questionnaire_answer_trgm
--   ON user_questionnaire_answers USING GIN (user_questionnaire_answer gin_trgm_ops);


COMMIT;