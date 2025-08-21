-- ============================
-- FAQ ANSWERS (drop child dulu)
-- ============================
DROP INDEX IF EXISTS idx_fqa_text_trgm;
DROP INDEX IF EXISTS idx_fqa_search_fts;
DROP INDEX IF EXISTS idx_fqa_masjid_created;
DROP INDEX IF EXISTS idx_fqa_answered_by_created;
DROP INDEX IF EXISTS idx_fqa_question_created;

DROP TRIGGER IF EXISTS trg_fqa_unmark_on_delete ON faq_answers;
DROP FUNCTION IF EXISTS unmark_question_when_last_answer_deleted();

DROP TRIGGER IF EXISTS trg_fqa_mark_answered ON faq_answers;
DROP FUNCTION IF EXISTS mark_question_answered_on_insert();

DROP TRIGGER IF EXISTS trg_fqa_set_updated_at ON faq_answers;
DROP FUNCTION IF EXISTS set_faq_answers_updated_at();

DROP TABLE IF EXISTS faq_answers;

-- ============================
-- FAQ QUESTIONS
-- ============================
DROP INDEX IF EXISTS idx_fqq_text_trgm;
DROP INDEX IF EXISTS idx_fqq_search_fts;
DROP INDEX IF EXISTS idx_fqq_unanswered_created;
DROP INDEX IF EXISTS idx_fqq_user_created;
DROP INDEX IF EXISTS idx_fqq_lecture_answered_created;
DROP INDEX IF EXISTS idx_fqq_session_answered_created;

DROP TRIGGER IF EXISTS trg_fqq_set_updated_at ON faq_questions;
DROP FUNCTION IF EXISTS set_faq_questions_updated_at();

DROP TABLE IF EXISTS faq_questions;

-- Extensions tidak di-drop (bisa dipakai objek lain)
