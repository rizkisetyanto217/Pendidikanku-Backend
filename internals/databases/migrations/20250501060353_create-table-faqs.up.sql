-- UUID & trigram (untuk ILIKE/fuzzy)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================
-- FAQ QUESTIONS
-- ============================
CREATE TABLE IF NOT EXISTS faq_questions (
  faq_question_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  faq_question_user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  faq_question_text              TEXT NOT NULL,
  faq_question_lecture_id        UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  faq_question_lecture_session_id UUID REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,

  faq_question_is_answered       BOOLEAN NOT NULL DEFAULT FALSE,

  faq_question_created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  faq_question_updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  -- Kolom FTS untuk search judul/isi pertanyaan
  faq_question_search tsvector GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(faq_question_text,''))
  ) STORED
);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION set_faq_questions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.faq_question_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_fqq_set_updated_at ON faq_questions;
CREATE TRIGGER trg_fqq_set_updated_at
BEFORE UPDATE ON faq_questions
FOR EACH ROW EXECUTE FUNCTION set_faq_questions_updated_at();

-- Indexing
-- Filter per session + status + terbaru
CREATE INDEX IF NOT EXISTS idx_fqq_session_answered_created
  ON faq_questions (faq_question_lecture_session_id, faq_question_is_answered, faq_question_created_at DESC);

-- Filter per lecture + status + terbaru
CREATE INDEX IF NOT EXISTS idx_fqq_lecture_answered_created
  ON faq_questions (faq_question_lecture_id, faq_question_is_answered, faq_question_created_at DESC);

-- Riwayat pertanyaan user (terbaru)
CREATE INDEX IF NOT EXISTS idx_fqq_user_created
  ON faq_questions (faq_question_user_id, faq_question_created_at DESC);

-- Hitung/list cepat yang belum dijawab
CREATE INDEX IF NOT EXISTS idx_fqq_unanswered_created
  ON faq_questions (faq_question_created_at)
  WHERE faq_question_is_answered = FALSE;

-- FTS & fuzzy search
CREATE INDEX IF NOT EXISTS idx_fqq_search_fts
  ON faq_questions USING GIN (faq_question_search);

CREATE INDEX IF NOT EXISTS idx_fqq_text_trgm
  ON faq_questions USING GIN (faq_question_text gin_trgm_ops);


-- ============================
-- FAQ ANSWERS
-- ============================
CREATE TABLE IF NOT EXISTS faq_answers (
  faq_answer_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  faq_answer_question_id  UUID NOT NULL REFERENCES faq_questions(faq_question_id) ON DELETE CASCADE,

  -- ON DELETE SET NULL ⇒ kolom harus nullable (jangan NOT NULL)
  faq_answer_answered_by  UUID REFERENCES users(id) ON DELETE SET NULL,

  faq_answer_text         TEXT NOT NULL,

  -- Masjid (untuk filter/analitik)
  faq_answer_masjid_id    UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  faq_answer_created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  faq_answer_updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  -- FTS untuk isi jawaban
  faq_answer_search tsvector GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(faq_answer_text,''))
  ) STORED
);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION set_faq_answers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.faq_answer_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_fqa_set_updated_at ON faq_answers;
CREATE TRIGGER trg_fqa_set_updated_at
BEFORE UPDATE ON faq_answers
FOR EACH ROW EXECUTE FUNCTION set_faq_answers_updated_at();

-- Sinkronisasi status pertanyaan ketika ada jawaban
-- Set is_answered = true saat ada jawaban baru
CREATE OR REPLACE FUNCTION mark_question_answered_on_insert()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE faq_questions
     SET faq_question_is_answered = TRUE,
         faq_question_updated_at = CURRENT_TIMESTAMP
   WHERE faq_question_id = NEW.faq_answer_question_id;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_fqa_mark_answered ON faq_answers;
CREATE TRIGGER trg_fqa_mark_answered
AFTER INSERT ON faq_answers
FOR EACH ROW EXECUTE FUNCTION mark_question_answered_on_insert();

-- Set is_answered = false jika semua jawaban dihapus
CREATE OR REPLACE FUNCTION unmark_question_when_last_answer_deleted()
RETURNS TRIGGER AS $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM faq_answers WHERE faq_answer_question_id = OLD.faq_answer_question_id
  ) THEN
    UPDATE faq_questions
       SET faq_question_is_answered = FALSE,
           faq_question_updated_at = CURRENT_TIMESTAMP
     WHERE faq_question_id = OLD.faq_answer_question_id;
  END IF;
  RETURN OLD;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_fqa_unmark_on_delete ON faq_answers;
CREATE TRIGGER trg_fqa_unmark_on_delete
AFTER DELETE ON faq_answers
FOR EACH ROW EXECUTE FUNCTION unmark_question_when_last_answer_deleted();

-- Indexing
-- Jawaban per pertanyaan (terbaru)
CREATE INDEX IF NOT EXISTS idx_fqa_question_created
  ON faq_answers (faq_answer_question_id, faq_answer_created_at DESC);

-- Jawaban per penjawab (terbaru), abaikan NULL
CREATE INDEX IF NOT EXISTS idx_fqa_answered_by_created
  ON faq_answers (faq_answer_answered_by, faq_answer_created_at DESC)
  WHERE faq_answer_answered_by IS NOT NULL;

-- Jawaban per masjid (terbaru)
CREATE INDEX IF NOT EXISTS idx_fqa_masjid_created
  ON faq_answers (faq_answer_masjid_id, faq_answer_created_at DESC);

-- FTS & fuzzy search
CREATE INDEX IF NOT EXISTS idx_fqa_search_fts
  ON faq_answers USING GIN (faq_answer_search);

CREATE INDEX IF NOT EXISTS idx_fqa_text_trgm
  ON faq_answers USING GIN (faq_answer_text gin_trgm_ops);
