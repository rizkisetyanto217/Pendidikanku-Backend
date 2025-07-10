-- =============================
-- 📋 Tabel Pertanyaan Kuisioner
-- =============================
CREATE TABLE IF NOT EXISTS questionnaire_questions (
  questionnaire_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  questionnaire_question_text TEXT NOT NULL,
  questionnaire_question_type INT NOT NULL CHECK (questionnaire_question_type IN (1, 2, 3)), -- ENUM: 1=rating, 2=text, 3=choice
  questionnaire_question_options TEXT[], -- digunakan hanya jika type = 3 (choice)
  questionnaire_question_event_id UUID REFERENCES events(event_id) ON DELETE CASCADE,
  questionnaire_question_lecture_session_id UUID REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  questionnaire_question_scope INT NOT NULL DEFAULT 1 CHECK (questionnaire_question_scope IN (1, 2, 3)), -- ENUM: 1=general, 2=event, 3=lecture
  questionnaire_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_questionnaire_question_event_id ON questionnaire_questions(questionnaire_question_event_id);
CREATE INDEX IF NOT EXISTS idx_questionnaire_question_session_id ON questionnaire_questions(questionnaire_question_lecture_session_id);
CREATE INDEX IF NOT EXISTS idx_questionnaire_question_scope ON questionnaire_questions(questionnaire_question_scope);


-- =============================
-- 🧾 Tabel Jawaban Kuisioner User
-- =============================
CREATE TABLE IF NOT EXISTS user_questionnaire_answers (
  user_questionnaire_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_questionnaire_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  user_questionnaire_type INT NOT NULL CHECK (user_questionnaire_type IN (1, 2)), -- ENUM: 1=lecture, 2=event
  user_questionnaire_reference_id UUID, -- ID dari lecture_session atau event
  user_questionnaire_question_id UUID REFERENCES questionnaire_questions(questionnaire_question_id) ON DELETE CASCADE,
  user_questionnaire_answer TEXT NOT NULL,
  user_questionnaire_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ✅ Indexing untuk efisiensi query
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_user_id ON user_questionnaire_answers(user_questionnaire_user_id);
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_ref_id ON user_questionnaire_answers(user_questionnaire_reference_id);
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_question_id ON user_questionnaire_answers(user_questionnaire_question_id);
CREATE INDEX IF NOT EXISTS idx_user_questionnaire_created_at ON user_questionnaire_answers(user_questionnaire_created_at);
