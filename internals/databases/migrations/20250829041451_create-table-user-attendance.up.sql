-- =========================================
-- UP Migration (Refactor Final) — NO triggers / NO DO blocks
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- A) USER_TYPE (master per masjid)
-- =========================================
-- =========================================
-- A) USER_ATTENDANCE_TYPE (master jenis attendance per masjid)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance_type (
  user_attendance_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_attendance_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_attendance_type_code  VARCHAR(32) NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  user_attendance_type_label VARCHAR(80),
  user_attendance_type_desc  TEXT,

  user_attendance_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  user_attendance_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_deleted_at TIMESTAMPTZ
);

-- Unik per masjid + code (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uatt_code_per_masjid_alive
  ON user_attendance_type (
    user_attendance_type_masjid_id,
    UPPER(user_attendance_type_code)
  )
  WHERE user_attendance_type_deleted_at IS NULL;

-- Pencarian label cepat (trigram) untuk data aktif
CREATE INDEX IF NOT EXISTS gin_uatt_label_trgm
  ON user_attendance_type USING GIN (user_attendance_type_label gin_trgm_ops)
  WHERE user_attendance_type_deleted_at IS NULL;

-- Filter umum per masjid (aktif saja)
CREATE INDEX IF NOT EXISTS idx_uatt_masjid_active
  ON user_attendance_type (user_attendance_type_masjid_id, user_attendance_type_is_active)
  WHERE user_attendance_type_deleted_at IS NULL;

-- Listing terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_uatt_masjid_created_desc
  ON user_attendance_type (user_attendance_type_masjid_id, user_attendance_type_created_at DESC)
  WHERE user_attendance_type_deleted_at IS NULL;

-- BRIN untuk time-series besar
CREATE INDEX IF NOT EXISTS brin_uatt_created_at
  ON user_attendance_type USING BRIN (user_attendance_type_created_at);




-- =========================================
-- B) USER ATTENDANCE (per siswa per sesi)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance (
  user_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- ganti: users -> masjid_students
  user_attendance_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- status kehadiran
  user_attendance_status VARCHAR(16) NOT NULL DEFAULT 'present'
    CHECK (user_attendance_status IN ('present','absent','excused','late')),

  -- Relasi ke USER_TYPE (jenis kegiatan per siswa), nullable
   -- Relasi ke USER_ATTENDANCE_TYPE (jenis kegiatan per siswa), nullable
  user_attendance_type_id UUID
    REFERENCES public.user_attendance_type(user_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,


  -- Kolom 'user_quran' sederhana
  user_attendance_desc       TEXT,           -- deskripsi bacaan hari itu
  user_attendance_score      NUMERIC(5,2)    CHECK (user_attendance_score IS NULL OR user_attendance_score BETWEEN 0 AND 100),
  user_attendance_is_passed  BOOLEAN,        -- lulus / tidak

  -- catatan tambahan
  user_attendance_user_note    TEXT,
  user_attendance_teacher_note TEXT,

  -- timestamps
  user_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_deleted_at TIMESTAMPTZ
);

-- 1 baris aktif per (masjid, session, masjid_student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_attendance_alive
  ON user_attendance(user_attendance_masjid_id, user_attendance_session_id, user_attendance_masjid_student_id)
  WHERE user_attendance_deleted_at IS NULL;

-- Jalur akses umum
CREATE INDEX IF NOT EXISTS idx_user_attendance_session
  ON user_attendance(user_attendance_session_id)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_student
  ON user_attendance(user_attendance_masjid_student_id)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_status
  ON user_attendance(user_attendance_status)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_type_id
  ON user_attendance(user_attendance_type_id)
  WHERE user_attendance_deleted_at IS NULL;

-- Listing terbaru per masjid & per sesi
CREATE INDEX IF NOT EXISTS idx_user_attendance_masjid_created_desc
  ON user_attendance(user_attendance_masjid_id, user_attendance_created_at DESC)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_session_created_desc
  ON user_attendance(user_attendance_session_id, user_attendance_created_at DESC)
  WHERE user_attendance_deleted_at IS NULL;

-- BRIN untuk time-series besar
CREATE INDEX IF NOT EXISTS brin_user_attendance_created_at
  ON user_attendance USING BRIN (user_attendance_created_at);

-- Pencarian di deskripsi aktif
CREATE INDEX IF NOT EXISTS gin_user_attendance_desc_trgm
  ON user_attendance USING GIN (user_attendance_desc gin_trgm_ops)
  WHERE user_attendance_deleted_at IS NULL;


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

COMMIT;
