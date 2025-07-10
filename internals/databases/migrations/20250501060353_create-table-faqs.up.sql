CREATE TABLE IF NOT EXISTS faq_questions (
  faq_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  faq_question_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  faq_question_text TEXT NOT NULL,
  faq_question_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  faq_question_lecture_session_id UUID REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  faq_question_is_answered BOOLEAN DEFAULT FALSE,
  faq_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_faq_questions_user_id ON faq_questions(faq_question_user_id);
CREATE INDEX IF NOT EXISTS idx_faq_questions_lecture_session_id ON faq_questions(faq_question_lecture_session_id);
CREATE INDEX IF NOT EXISTS idx_faq_questions_lecture_id ON faq_questions(faq_question_lecture_id);
CREATE INDEX IF NOT EXISTS idx_faq_questions_is_answered ON faq_questions(faq_question_is_answered);


CREATE TABLE IF NOT EXISTS faq_answers (
  faq_answer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  faq_answer_question_id UUID NOT NULL REFERENCES faq_questions(faq_question_id) ON DELETE CASCADE,
  faq_answer_answered_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
  faq_answer_text TEXT NOT NULL,
  faq_answer_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_faq_answers_question_id ON faq_answers(faq_answer_question_id);
CREATE INDEX IF NOT EXISTS idx_faq_answers_answered_by ON faq_answers(faq_answer_answered_by);
