CREATE TABLE IF NOT EXISTS lecture_sessions (
    lecture_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lecture_session_title VARCHAR(255) NOT NULL,
    lecture_session_description TEXT,
    lecture_session_teacher JSONB NOT NULL, -- Contoh: {"id": "...", "name": "..."}
    lecture_session_start_time TIMESTAMP NOT NULL,
    lecture_session_end_time TIMESTAMP NOT NULL,
    lecture_session_place TEXT,
    lecture_session_image_url TEXT, -- Gambar opsional per sesi
    lecture_session_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
    lecture_session_capacity INT,
    lecture_session_is_public BOOLEAN DEFAULT TRUE,
    lecture_session_is_registration_required BOOLEAN DEFAULT FALSE,
    lecture_session_is_paid BOOLEAN DEFAULT FALSE,
    lecture_session_price INT,
    lecture_session_payment_deadline TIMESTAMP,
    lecture_session_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_lecture ON lecture_sessions(lecture_session_lecture_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_start_time ON lecture_sessions(lecture_session_start_time);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_end_time ON lecture_sessions(lecture_session_end_time);

-- Index untuk pencarian berdasarkan ID teacher dalam JSON
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_teacher_id 
ON lecture_sessions ((lecture_session_teacher->>'id'));
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_teacher_name 
ON lecture_sessions ((lecture_session_teacher->>'name'));



CREATE TABLE user_lecture_sessions (
  user_lecture_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Kehadiran dan nilai
  user_lecture_session_attendance_status INT, -- 0 = tidak hadir, 1 = hadir, 2 = hadir online
  user_lecture_session_grade_result FLOAT,

  -- Relasi
  user_lecture_session_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  user_lecture_session_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Pendaftaran dan pembayaran
  user_lecture_session_is_registered BOOLEAN DEFAULT FALSE,
  user_lecture_session_has_paid BOOLEAN DEFAULT FALSE,
  user_lecture_session_paid_amount INT,
  user_lecture_session_payment_time TIMESTAMP,

  -- Metadata
  user_lecture_session_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_user 
  ON user_lecture_sessions(user_lecture_session_user_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_lecture_session 
  ON user_lecture_sessions(user_lecture_session_lecture_session_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_attendance_status 
  ON user_lecture_sessions(user_lecture_session_attendance_status);
