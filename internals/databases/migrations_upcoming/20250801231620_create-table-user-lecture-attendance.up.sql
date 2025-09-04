-- =====================================================================
-- Migration: user_lecture_sessions_attendance (optimized)
-- DB: PostgreSQL
-- =====================================================================

BEGIN;

-- -------------------------------------------------
-- Extensions (idempotent)
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =====================================================================
-- ===============================  UP  =================================
-- =====================================================================

CREATE TABLE IF NOT EXISTS user_lecture_sessions_attendance (
  user_lecture_sessions_attendance_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_lecture_sessions_attendance_user_id     UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_lecture_sessions_attendance_lecture_session_id UUID NOT NULL
    REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,

  user_lecture_sessions_attendance_lecture_id  UUID NOT NULL
    REFERENCES lectures(lecture_id) ON DELETE CASCADE,

  -- 0=unknown, 1=onsite, 2=online, 3=absent  (sesuaikan kalau perlu)
  user_lecture_sessions_attendance_status      INTEGER NOT NULL DEFAULT 0
    CHECK (user_lecture_sessions_attendance_status IN (0,1,2,3)),

  user_lecture_sessions_attendance_notes           TEXT,
  user_lecture_sessions_attendance_personal_notes  TEXT,

  user_lecture_sessions_attendance_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_sessions_attendance_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_sessions_attendance_deleted_at  TIMESTAMPTZ NULL
);


-- =================================================
-- Indexing (query-first design)
-- =================================================

-- 1) Satu record AKTIF per (user, session)  → cegah duplikat saat belum dihapus
--    Gunakan UNIQUE PARTIAL INDEX (soft delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_user_session_active
  ON user_lecture_sessions_attendance(
    user_lecture_sessions_attendance_user_id,
    user_lecture_sessions_attendance_lecture_session_id
  )
  WHERE user_lecture_sessions_attendance_deleted_at IS NULL;

-- 2) Feed aktivitas user (list by user, terbaru, aktif)
CREATE INDEX IF NOT EXISTS idx_attendance_user_created_active
  ON user_lecture_sessions_attendance(
    user_lecture_sessions_attendance_user_id,
    user_lecture_sessions_attendance_created_at DESC
  )
  WHERE user_lecture_sessions_attendance_deleted_at IS NULL;

-- 3) Rekap per sesi (mis. hitung hadir/online/absent), aktif saja
CREATE INDEX IF NOT EXISTS idx_attendance_session_status_active
  ON user_lecture_sessions_attendance(
    user_lecture_sessions_attendance_lecture_session_id,
    user_lecture_sessions_attendance_status
  )
  WHERE user_lecture_sessions_attendance_deleted_at IS NULL;

-- 4) Rekap per lecture (agregat lintas sesi), aktif saja
CREATE INDEX IF NOT EXISTS idx_attendance_lecture_status_active
  ON user_lecture_sessions_attendance(
    user_lecture_sessions_attendance_lecture_id,
    user_lecture_sessions_attendance_status
  )
  WHERE user_lecture_sessions_attendance_deleted_at IS NULL;

-- 5) Lookup cepat (user + lecture) untuk progres tema
CREATE INDEX IF NOT EXISTS idx_attendance_user_lecture_active
  ON user_lecture_sessions_attendance(
    user_lecture_sessions_attendance_user_id,
    user_lecture_sessions_attendance_lecture_id
  )
  WHERE user_lecture_sessions_attendance_deleted_at IS NULL;

-- 6) Housekeeping & audit
CREATE INDEX IF NOT EXISTS idx_attendance_deleted_at
  ON user_lecture_sessions_attendance(user_lecture_sessions_attendance_deleted_at);
CREATE INDEX IF NOT EXISTS idx_attendance_updated_at
  ON user_lecture_sessions_attendance(user_lecture_sessions_attendance_updated_at DESC);



COMMIT;