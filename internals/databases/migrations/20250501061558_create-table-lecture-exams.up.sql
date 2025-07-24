-- ============================
-- Tabel: lecture_exams
-- ============================
CREATE TABLE IF NOT EXISTS lecture_exams (
  lecture_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_exam_title VARCHAR(255) NOT NULL,
  lecture_exam_description TEXT,
  lecture_exam_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_exam_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  lecture_exam_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_exams_lecture_id 
  ON lecture_exams(lecture_exam_lecture_id);

CREATE INDEX IF NOT EXISTS idx_lecture_exams_masjid_id 
  ON lecture_exams(lecture_exam_masjid_id);

CREATE INDEX IF NOT EXISTS idx_lecture_exams_created_at 
  ON lecture_exams(lecture_exam_created_at);



-- ================================
-- Tabel: user_lecture_exams
-- ================================
CREATE TABLE IF NOT EXISTS user_lecture_exams (
  user_lecture_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_lecture_exam_grade_result FLOAT,
  user_lecture_exam_exam_id UUID REFERENCES lecture_exams(lecture_exam_id) ON DELETE CASCADE,
  user_lecture_exam_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_exam_user_name VARCHAR(100);
  user_lecture_exam_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_lecture_exam_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_lecture_exams_exam_id 
  ON user_lecture_exams(user_lecture_exam_exam_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_exams_user_id 
  ON user_lecture_exams(user_lecture_exam_user_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_exams_masjid_id 
  ON user_lecture_exams(user_lecture_exam_masjid_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_exams_created_at 
  ON user_lecture_exams(user_lecture_exam_created_at);
