CREATE TABLE IF NOT EXISTS user_lecture_sessions_attendance (
    user_lecture_sessions_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_lecture_sessions_attendance_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_lecture_sessions_attendance_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
    user_lecture_sessions_attendance_lecture_id UUID NOT NULL 
        REFERENCES lectures(lecture_id) ON DELETE CASCADE,
    user_lecture_sessions_attendance_status INTEGER DEFAULT 0,
    user_lecture_sessions_attendance_notes TEXT,
    user_lecture_sessions_attendance_personal_notes TEXT,
    user_lecture_sessions_attendance_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_lecture_sessions_attendance_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_lecture_sessions_attendance_deleted_at TIMESTAMP
);

-- 🔍 Index untuk efisiensi query
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_attendance_user_id 
    ON user_lecture_sessions_attendance(user_lecture_sessions_attendance_user_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_attendance_lecture_session_id 
    ON user_lecture_sessions_attendance(user_lecture_sessions_attendance_lecture_session_id);


CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_attendance_lecture_id 
    ON user_lecture_sessions_attendance(user_lecture_sessions_attendance_lecture_id);