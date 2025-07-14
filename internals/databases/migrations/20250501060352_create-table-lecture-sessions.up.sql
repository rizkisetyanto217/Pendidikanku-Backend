CREATE TABLE IF NOT EXISTS lecture_sessions (
  lecture_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- 📝 Informasi Sesi
  lecture_session_title VARCHAR(255) NOT NULL,
  lecture_session_description TEXT,

  -- 👤 Pengajar
  lecture_session_teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  lecture_session_teacher_name VARCHAR(255),

  -- ⏰ Jadwal
  lecture_session_start_time TIMESTAMP NOT NULL,
  lecture_session_end_time TIMESTAMP NOT NULL,

  -- 📍 Lokasi & Gambar
  lecture_session_place TEXT,
  lecture_session_image_url TEXT,

  -- 🔗 Relasi ke lectures
  lecture_session_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,

  -- 🔗 Masjid langsung (cache masjid untuk efisiensi)
  lecture_session_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- ✅ Validasi Admin
  lecture_session_approved_by_admin_id UUID REFERENCES users(id),
  lecture_session_approved_by_admin_at TIMESTAMP,

  -- ✅ Validasi Author
  lecture_session_approved_by_author_id UUID REFERENCES users(id),
  lecture_session_approved_by_author_at TIMESTAMP,

  -- ✅ Validasi Teacher
  lecture_session_approved_by_teacher_id UUID REFERENCES users(id),
  lecture_session_approved_by_teacher_at TIMESTAMP,

  -- ✅ Validasi DKM
  lecture_session_approved_by_dkm_at TIMESTAMP,

  -- 📌 Status publikasi oleh DKM
  lecture_session_is_active BOOLEAN DEFAULT FALSE,

  -- 🕒 Metadata
  lecture_session_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  lecture_session_updated_at TIMESTAMP,
  lecture_session_deleted_at TIMESTAMP
);


-- Tambah masjid_id

-- 🔗 Relasi ke lecture utama
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_lecture_id 
  ON lecture_sessions (lecture_session_lecture_id);

-- ⏰ Jadwal
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_start_time 
  ON lecture_sessions (lecture_session_start_time);

CREATE INDEX IF NOT EXISTS idx_lecture_sessions_end_time 
  ON lecture_sessions (lecture_session_end_time);

-- 👤 Pengajar
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_teacher_id 
  ON lecture_sessions (lecture_session_teacher_id);

-- ✅ Validasi Admin
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_admin 
  ON lecture_sessions (lecture_session_approved_by_admin_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_admin_at 
  ON lecture_sessions (lecture_session_approved_by_admin_at);

-- ✅ Validasi Author
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_author 
  ON lecture_sessions (lecture_session_approved_by_author_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_author_at 
  ON lecture_sessions (lecture_session_approved_by_author_at);

-- ✅ Validasi Teacher
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_teacher 
  ON lecture_sessions (lecture_session_approved_by_teacher_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_approved_by_teacher_at 
  ON lecture_sessions (lecture_session_approved_by_teacher_at);

-- Index untuk masjid_id
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_masjid_id 
  ON lecture_sessions (lecture_session_masjid_id);

-- 📌 Status aktif
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_is_active 
  ON lecture_sessions (lecture_session_is_active);



CREATE TABLE IF NOT EXISTS user_lecture_sessions (
  user_lecture_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Kehadiran dan evaluasi per sesi
  user_lecture_session_attendance_status INT, -- 0 = tidak hadir, 1 = hadir, 2 = hadir online
  user_lecture_session_grade_result FLOAT,

  -- Relasi
  user_lecture_session_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  user_lecture_session_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- ✅ Masjid ID
  user_lecture_session_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_lecture_session_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  user_lecture_session_updated_at TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_user 
  ON user_lecture_sessions(user_lecture_session_user_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_lecture_session 
  ON user_lecture_sessions(user_lecture_session_lecture_session_id);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_attendance_status 
  ON user_lecture_sessions(user_lecture_session_attendance_status);

CREATE INDEX IF NOT EXISTS idx_user_lecture_sessions_masjid_id 
  ON user_lecture_sessions(user_lecture_session_masjid_id);
