CREATE TABLE lecture_sessions_questions (
  lecture_sessions_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_question TEXT NOT NULL,
  lecture_sessions_question_answer TEXT NOT NULL,
  lecture_sessions_question_correct CHAR(1) NOT NULL CHECK (lecture_sessions_question_correct IN ('A', 'B', 'C', 'D')),
  lecture_sessions_question_explanation TEXT,
  lecture_sessions_question_quiz_id UUID REFERENCES lecture_sessions_quiz(lecture_sessions_quiz_id) ON DELETE SET NULL,
  lecture_sessions_question_exam_id UUID REFERENCES lecture_sessions_exams(lecture_sessions_exam_id) ON DELETE SET NULL,


  -- ✅ Masjid ID wajib
  lecture_sessions_question_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,


  lecture_sessions_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ✅ Index untuk pencarian berdasarkan quiz atau exam
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_questions_quiz_id
  ON lecture_sessions_questions(lecture_sessions_question_quiz_id);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_questions_exam_id
  ON lecture_sessions_questions(lecture_sessions_question_exam_id);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_questions_created_at
  ON lecture_sessions_questions(lecture_sessions_question_created_at);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_questions_masjid_id
  ON lecture_sessions_questions(lecture_sessions_question_masjid_id);


  
CREATE TABLE IF NOT EXISTS lecture_sessions_user_questions (
  lecture_sessions_user_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_user_question_answer CHAR(1) NOT NULL CHECK (lecture_sessions_user_question_answer IN ('A', 'B', 'C', 'D')),
  lecture_sessions_user_question_is_correct BOOLEAN NOT NULL,
  lecture_sessions_user_question_question_id UUID NOT NULL REFERENCES lecture_sessions_questions(lecture_sessions_question_id) ON DELETE CASCADE,

  -- ✅ Masjid ID wajib
  lecture_sessions_user_question_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_user_question_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_user_questions_question_id 
  ON lecture_sessions_user_questions(lecture_sessions_user_question_question_id);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_user_questions_masjid_id 
  ON lecture_sessions_user_questions(lecture_sessions_user_question_masjid_id);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_user_questions_created_at 
  ON lecture_sessions_user_questions(lecture_sessions_user_question_created_at);
