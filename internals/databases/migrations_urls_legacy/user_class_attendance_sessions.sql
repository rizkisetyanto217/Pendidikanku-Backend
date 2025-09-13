

-- =========================================
-- C) USER ATTENDANCE URLS (child, multi-lampiran)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance_urls (
  user_attendance_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant scope langsung di child
  user_attendance_urls_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Relasi ke parent attendance
  user_attendance_urls_attendance_id UUID NOT NULL
    REFERENCES user_attendance(user_attendance_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Relasi ke USER_TYPE (MEDIA_KIND: AUDIO/IMAGE/VIDEO/FILE) - nullable
  user_attendance_type_id UUID
  REFERENCES public.user_attendance_type(user_attendance_type_id)
  ON UPDATE CASCADE ON DELETE SET NULL,

  -- Metadata
  user_attendance_urls_label VARCHAR(120),

  -- URL aktif (wajib)
  user_attendance_urls_href TEXT NOT NULL,

  -- housekeeping (opsional)
  user_attendance_urls_trash_url            TEXT,
  user_attendance_urls_delete_pending_until TIMESTAMPTZ,

  -- uploader (opsional)
  user_attendance_urls_uploader_teacher_id UUID, -- FK ke masjid_teachers bisa ditambah terpisah jika dibutuhkan
  -- ganti: users -> masjid_students (uploader adalah santri)
  user_attendance_urls_uploader_student_id UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE SET NULL,

  -- timestamps
  user_attendance_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_urls_deleted_at TIMESTAMPTZ
);

-- Jalur akses umum
CREATE INDEX IF NOT EXISTS idx_user_attendance_urls_attendance
  ON user_attendance_urls(user_attendance_urls_attendance_id)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_urls_masjid_created_desc
  ON user_attendance_urls(user_attendance_urls_masjid_id, user_attendance_urls_created_at DESC)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uau_uploader_teacher
  ON user_attendance_urls(user_attendance_urls_uploader_teacher_id)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uau_uploader_student
  ON user_attendance_urls(user_attendance_urls_uploader_student_id)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uau_type_id
  ON user_attendance_urls(user_attendance_type_id)
  WHERE user_attendance_urls_deleted_at IS NULL;

-- Optimizations
CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_attendance_href_alive
  ON user_attendance_urls(user_attendance_urls_attendance_id, user_attendance_urls_href)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uau_attendance_created_desc
  ON user_attendance_urls(user_attendance_urls_attendance_id, user_attendance_urls_created_at DESC)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_uau_created_at
  ON user_attendance_urls USING BRIN (user_attendance_urls_created_at);