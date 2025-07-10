CREATE TABLE IF NOT EXISTS lecture_sessions_quiz (
  lecture_sessions_quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_quiz_title VARCHAR(255) NOT NULL,
  lecture_sessions_quiz_description TEXT,
  lecture_sessions_quiz_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  lecture_sessions_quiz_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_quiz_lecture_session_id ON lecture_sessions_quiz(lecture_sessions_quiz_lecture_session_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_quiz_created_at ON lecture_sessions_quiz(lecture_sessions_quiz_created_at);


  CREATE TABLE IF NOT EXISTS user_lecture_sessions_quiz (
    user_lecture_sessions_quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_lecture_sessions_quiz_grade_result FLOAT,
    user_lecture_sessions_quiz_quiz_id UUID NOT NULL REFERENCES lecture_sessions_quiz(lecture_sessions_quiz_id) ON DELETE CASCADE,
    user_lecture_sessions_quiz_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_lecture_sessions_quiz_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );

  -- Indexing
  CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_quiz_quiz_id ON user_lecture_sessions_quiz(user_lecture_sessions_quiz_quiz_id);
  CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_quiz_user_id ON user_lecture_sessions_quiz(user_lecture_sessions_quiz_user_id);
  CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_quiz_created_at ON user_lecture_sessions_quiz(user_lecture_sessions_quiz_created_at);
