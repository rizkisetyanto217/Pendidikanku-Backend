CREATE TABLE IF NOT EXISTS lecture_sessions_exams (
  lecture_sessions_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_exam_title VARCHAR(255) NOT NULL,
  lecture_sessions_exam_description TEXT,
  lecture_sessions_exam_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_sessions_exam_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_exams_lecture_id ON lecture_sessions_exams(lecture_sessions_exam_lecture_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_exams_created_at ON lecture_sessions_exams(lecture_sessions_exam_created_at);


CREATE TABLE IF NOT EXISTS user_lecture_sessions_exams (
  user_lecture_sessions_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_lecture_sessions_exam_grade_result FLOAT,
  user_lecture_sessions_exam_exam_id UUID NOT NULL REFERENCES lecture_sessions_exams(lecture_sessions_exam_id) ON DELETE CASCADE,
  user_lecture_sessions_exam_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_sessions_exam_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_exams_exam_id ON user_lecture_sessions_exams(user_lecture_sessions_exam_exam_id);
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_exams_user_id ON user_lecture_sessions_exams(user_lecture_sessions_exam_user_id);
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_exams_created_at ON user_lecture_sessions_exams(user_lecture_sessions_exam_created_at);
