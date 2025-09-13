BEGIN;

-- =========================================================
-- Prasyarat
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================================
-- Tabel: class_attendance_sessions (tanpa section_id; pakai CSST)
-- =========================================================
CREATE TABLE class_attendance_sessions (
  class_attendance_sessions_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant guard
  class_attendance_sessions_masjid_id    UUID NOT NULL,

  -- Relasi utama: assignment (CSST) → menentukan section & subject
  class_attendance_sessions_csst_id      UUID NOT NULL,

  -- Opsional
  class_attendance_sessions_class_room_id UUID,

  -- Metadata sesi
  class_attendance_sessions_date         DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title        TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note         TEXT,

  -- Guru yang mengajar (tetap/pengganti) → masjid_teachers
  class_attendance_sessions_teacher_id   UUID
    REFERENCES masjid_teachers (masjid_teacher_id) ON DELETE SET NULL,

  -- Soft delete
  class_attendance_sessions_deleted_at   TIMESTAMPTZ,

  -- =========================
  -- Foreign Keys (tenant-safe)
  -- =========================

  -- csst_id → class_section_subject_teachers(id)
  CONSTRAINT fk_cas_csst
    FOREIGN KEY (class_attendance_sessions_csst_id)
    REFERENCES class_section_subject_teachers (class_section_subject_teachers_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- class_room_id → class_rooms(id)
  CONSTRAINT fk_cas_class_room
    FOREIGN KEY (class_attendance_sessions_class_room_id)
    REFERENCES class_rooms (class_room_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =========================================================
-- Indexes
-- =========================================================

-- Tenant & tanggal
CREATE INDEX idx_cas_masjid
  ON class_attendance_sessions (class_attendance_sessions_masjid_id);

CREATE INDEX idx_cas_date
  ON class_attendance_sessions (class_attendance_sessions_date DESC);

-- CSST reference (alive only)
CREATE INDEX idx_cas_csst_alive
  ON class_attendance_sessions (class_attendance_sessions_csst_id)
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Query by teacher per masjid per date (alive)
CREATE INDEX idx_cas_masjid_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_teacher_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Room optional
CREATE INDEX idx_cas_class_room
  ON class_attendance_sessions (class_attendance_sessions_class_room_id);

-- Unik: satu sesi per (masjid, CSST, date) saat belum soft-deleted
CREATE UNIQUE INDEX uq_cas_masjid_csst_date
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_csst_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;


COMMIT;