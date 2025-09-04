-- =========================================================
-- ================ BLOK 2: FAQ_ANSWERS ====================
-- =========================================================

-- ---------------------------------------------------------
-- Tables (faq_answers)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS faq_answers (
  faq_answer_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  faq_answer_question_id  UUID NOT NULL REFERENCES faq_questions(faq_question_id) ON DELETE CASCADE,
  faq_answer_answered_by  UUID REFERENCES users(id) ON DELETE SET NULL,
  faq_answer_text         TEXT NOT NULL,
  faq_answer_masjid_id    UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  faq_answer_created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  faq_answer_updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  faq_answer_deleted_at   TIMESTAMPTZ NULL,
  faq_answer_search tsvector GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(faq_answer_text,''))
  ) STORED
);


-- helper: re-evaluate is_answered (butuh faq_answers sudah ada)
CREATE OR REPLACE FUNCTION fn_reevaluate_question_answered(p_question_id UUID)
RETURNS VOID AS $$
DECLARE
  v_has_alive_answer BOOLEAN;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM faq_answers
     WHERE faq_answer_question_id = p_question_id
       AND faq_answer_deleted_at IS NULL
  ) INTO v_has_alive_answer;

  UPDATE faq_questions
     SET faq_question_is_answered = COALESCE(v_has_alive_answer, FALSE),
         faq_question_updated_at  = CURRENT_TIMESTAMP
   WHERE faq_question_id = p_question_id;
END$$ LANGUAGE plpgsql;

-- insert jawaban → tandai answered
CREATE OR REPLACE FUNCTION fn_mark_question_answered_on_insert()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.faq_answer_deleted_at IS NULL THEN
    UPDATE faq_questions
       SET faq_question_is_answered = TRUE,
           faq_question_updated_at  = CURRENT_TIMESTAMP
     WHERE faq_question_id = NEW.faq_answer_question_id
       AND faq_question_deleted_at IS NULL;
  END IF;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- hard delete jawaban → re-eval status pertanyaan
CREATE OR REPLACE FUNCTION fn_unmark_question_when_last_answer_deleted()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM fn_reevaluate_question_answered(OLD.faq_answer_question_id);
  RETURN OLD;
END$$ LANGUAGE plpgsql;

-- soft delete / restore jawaban → re-eval
CREATE OR REPLACE FUNCTION fn_reevaluate_on_answer_softdelete_restore()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM fn_reevaluate_question_answered(NEW.faq_answer_question_id);
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- ⚠️ sekarang aman bikin fungsi soft-delete/restore untuk faq_questions
-- karena helper di atas sudah ada & faq_answers sudah exist
CREATE OR REPLACE FUNCTION fn_on_question_softdelete_affect_answers()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.faq_question_deleted_at IS NOT NULL THEN
    -- soft delete pertanyaan → anggap belum terjawab
    UPDATE faq_questions
       SET faq_question_is_answered = FALSE,
           faq_question_updated_at  = CURRENT_TIMESTAMP
     WHERE faq_question_id = NEW.faq_question_id;
  ELSE
    -- restore pertanyaan → evaluasi ulang berdasar jawaban hidup
    PERFORM fn_reevaluate_question_answered(NEW.faq_question_id);
  END IF;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------
-- Triggers (faq_answers + melengkapi faq_questions)
-- ---------------------------------------------------------
-- answers: updated_at
DROP TRIGGER IF EXISTS trg_fqa_touch ON faq_answers;
CREATE TRIGGER trg_fqa_touch
BEFORE UPDATE ON faq_answers
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_fqa();

-- answers: insert → answered
DROP TRIGGER IF EXISTS trg_fqa_mark_answered ON faq_answers;
CREATE TRIGGER trg_fqa_mark_answered
AFTER INSERT ON faq_answers
FOR EACH ROW EXECUTE FUNCTION fn_mark_question_answered_on_insert();

-- answers: hard delete → re-eval
DROP TRIGGER IF EXISTS trg_fqa_unmark_on_delete ON faq_answers;
CREATE TRIGGER trg_fqa_unmark_on_delete
AFTER DELETE ON faq_answers
FOR EACH ROW EXECUTE FUNCTION fn_unmark_question_when_last_answer_deleted();

-- answers: soft delete / restore → re-eval
DROP TRIGGER IF EXISTS trg_fqa_softdel_restore ON faq_answers;
CREATE TRIGGER trg_fqa_softdel_restore
AFTER UPDATE OF faq_answer_deleted_at ON faq_answers
FOR EACH ROW EXECUTE FUNCTION fn_reevaluate_on_answer_softdelete_restore();

-- questions: soft delete / restore → reset / re-eval (dibuat di blok 2 krn dependensi)
DROP TRIGGER IF EXISTS trg_fqq_softdel_restore ON faq_questions;
CREATE TRIGGER trg_fqq_softdel_restore
AFTER UPDATE OF faq_question_deleted_at ON faq_questions
FOR EACH ROW EXECUTE FUNCTION fn_on_question_softdelete_affect_answers();

-- ---------------------------------------------------------
-- Indexing & Optimize (faq_answers)
-- ---------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_fqa_question_created
  ON faq_answers (faq_answer_question_id, faq_answer_created_at DESC)
  WHERE faq_answer_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fqa_answered_by_created
  ON faq_answers (faq_answer_answered_by, faq_answer_created_at DESC)
  WHERE faq_answer_answered_by IS NOT NULL AND faq_answer_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fqa_masjid_created
  ON faq_answers (faq_answer_masjid_id, faq_answer_created_at DESC)
  WHERE faq_answer_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fqa_search_fts
  ON faq_answers USING GIN (faq_answer_search);

CREATE INDEX IF NOT EXISTS idx_fqa_text_trgm
  ON faq_answers USING GIN (faq_answer_text gin_trgm_ops)
  WHERE faq_answer_deleted_at IS NULL;

