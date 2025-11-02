
-- =========================================================
-- =================== MIGRATION DOWN ======================
-- =========================================================
BEGIN;

-- Drop indexes (answers → questions)
DROP INDEX IF EXISTS idx_fqa_text_trgm;
DROP INDEX IF EXISTS idx_fqa_search_fts;
DROP INDEX IF EXISTS idx_fqa_school_created;
DROP INDEX IF EXISTS idx_fqa_answered_by_created;
DROP INDEX IF EXISTS idx_fqa_question_created;

DROP INDEX IF EXISTS idx_fqq_text_trgm;
DROP INDEX IF EXISTS idx_fqq_search_fts;
DROP INDEX IF EXISTS idx_fqq_unanswered_created;
DROP INDEX IF EXISTS idx_fqq_user_created;
DROP INDEX IF EXISTS idx_fqq_lecture_answered_created;
DROP INDEX IF EXISTS idx_fqq_session_answered_created;

-- Drop triggers (answers → questions)
DROP TRIGGER IF EXISTS trg_fqa_mark_answered ON faq_answers;
DROP TRIGGER IF EXISTS trg_fqa_unmark_on_delete ON faq_answers;
DROP TRIGGER IF EXISTS trg_fqa_softdel_restore ON faq_answers;
DROP TRIGGER IF EXISTS trg_fqa_touch ON faq_answers;

DROP TRIGGER IF EXISTS trg_fqq_softdel_restore ON faq_questions;
DROP TRIGGER IF EXISTS trg_fqq_touch ON faq_questions;

-- Drop functions (answers helpers dulu, lalu questions)
DROP FUNCTION IF EXISTS fn_on_question_softdelete_affect_answers() CASCADE;
DROP FUNCTION IF EXISTS fn_reevaluate_on_answer_softdelete_restore() CASCADE;
DROP FUNCTION IF EXISTS fn_unmark_question_when_last_answer_deleted() CASCADE;
DROP FUNCTION IF EXISTS fn_mark_question_answered_on_insert() CASCADE;
DROP FUNCTION IF EXISTS fn_reevaluate_question_answered(UUID) CASCADE;
DROP FUNCTION IF EXISTS fn_touch_updated_at_fqa() CASCADE;

DROP FUNCTION IF EXISTS fn_touch_updated_at_fqq() CASCADE;

-- Drop tables (answers dulu, lalu questions)
DROP TABLE IF EXISTS faq_answers;
DROP TABLE IF EXISTS faq_questions;

COMMIT;