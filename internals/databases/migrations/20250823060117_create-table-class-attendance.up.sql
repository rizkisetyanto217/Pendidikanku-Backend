
-- =========================================================
-- Tabel: CLASS_ATTENDANCE_SESSIONS → refer ke CLASS_SCHEDULES
-- =========================================================
CREATE TABLE class_attendance_sessions (
  class_attendance_sessions_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant guard
  class_attendance_sessions_masjid_id     UUID NOT NULL,

  -- Relasi utama: jadwal (bukan CSST)
  class_attendance_sessions_schedule_id   UUID NOT NULL,

  -- Opsional override (boleh beda dari jadwal)
  class_attendance_sessions_class_room_id UUID
    REFERENCES class_rooms (class_room_id) ON DELETE SET NULL,
  class_attendance_sessions_teacher_id    UUID
    REFERENCES masjid_teachers (masjid_teacher_id) ON DELETE SET NULL,

  -- Metadata sesi
  class_attendance_sessions_date          DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title         TEXT,
  class_attendance_sessions_general_info  TEXT NOT NULL,
  class_attendance_sessions_note          TEXT,

  -- Rekap hasil kehadiran
  class_attendance_sessions_present_count INT,
  class_attendance_sessions_absent_count  INT,
  class_attendance_sessions_late_count    INT,
  class_attendance_sessions_excused_count INT,
  class_attendance_sessions_sick_count    INT,
  class_attendance_sessions_leave_count   INT,

  -- Soft delete
  class_attendance_sessions_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_deleted_at    TIMESTAMPTZ,

  -- Tenant-safe FK (komposit) → class_schedules
  CONSTRAINT fk_cas_schedule_tenant
    FOREIGN KEY (class_attendance_sessions_masjid_id, class_attendance_sessions_schedule_id)
    REFERENCES class_schedules (class_schedules_masjid_id, class_schedule_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================
-- Indexing attendance
-- =========================

-- Unik: satu sesi per (masjid, schedule, date) saat belum soft-deleted
CREATE UNIQUE INDEX uq_cas_masjid_schedule_date
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_schedule_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Tenant & tanggal (kalender)
CREATE INDEX idx_cas_masjid_date
  ON class_attendance_sessions (class_attendance_sessions_masjid_id, class_attendance_sessions_date DESC)
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Lookup berdasarkan schedule (alive)
CREATE INDEX idx_cas_schedule_alive
  ON class_attendance_sessions (class_attendance_sessions_schedule_id)
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Query per guru per masjid per tanggal (alive)
CREATE INDEX idx_cas_masjid_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_teacher_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Optional: filter ruangan (alive)
CREATE INDEX idx_cas_class_room_alive
  ON class_attendance_sessions (class_attendance_sessions_class_room_id)
  WHERE class_attendance_sessions_deleted_at IS NULL;


BEGIN;

-- =========================================================
-- CLASS_ATTENDANCE_SESSION_URL — selaras dengan announcement_urls
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_session_url (
  class_attendance_session_url_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  class_attendance_session_url_masjid_id          UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_attendance_session_url_session_id         UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'banner','image','video','attachment','link')
  class_attendance_session_url_kind               VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  class_attendance_session_url_href               TEXT,        -- URL publik (boleh NULL jika murni object storage)
  class_attendance_session_url_object_key         TEXT,        -- object key aktif di storage
  class_attendance_session_url_object_key_old     TEXT,        -- object key lama (retensi in-place replace)
  class_attendance_session_url_mime               VARCHAR(80), -- opsional

  -- Tampilan
  class_attendance_session_url_label              VARCHAR(160),
  class_attendance_session_url_order              INT NOT NULL DEFAULT 0,
  class_attendance_session_url_is_primary         BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  class_attendance_session_url_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_deleted_at         TIMESTAMPTZ,          -- soft delete (versi-per-baris)
  class_attendance_session_url_delete_pending_until TIMESTAMPTZ          -- tenggat purge (baris aktif dgn *_old atau baris soft-deleted)
);

-- =========================================================
-- INDEXING / OPTIMIZATION (paritas dg announcement_urls)
-- =========================================================

-- Lookup per session (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_casu_by_owner_live
  ON class_attendance_session_url (
    class_attendance_session_url_session_id,
    class_attendance_session_url_kind,
    class_attendance_session_url_is_primary DESC,
    class_attendance_session_url_order,
    class_attendance_session_url_created_at
  )
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_casu_by_masjid_live
  ON class_attendance_session_url (class_attendance_session_url_masjid_id)
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Satu primary per (session, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casu_primary_per_kind_alive
  ON class_attendance_session_url (class_attendance_session_url_session_id, class_attendance_session_url_kind)
  WHERE class_attendance_session_url_deleted_at IS NULL
    AND class_attendance_session_url_is_primary = TRUE;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_casu_purge_due
  ON class_attendance_session_url (class_attendance_session_url_delete_pending_until)
  WHERE class_attendance_session_url_delete_pending_until IS NOT NULL
    AND (
      (class_attendance_session_url_deleted_at IS NULL  AND class_attendance_session_url_object_key_old IS NOT NULL) OR
      (class_attendance_session_url_deleted_at IS NOT NULL AND class_attendance_session_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_casu_label_trgm_live
  ON class_attendance_session_url USING GIN (class_attendance_session_url_label gin_trgm_ops)
  WHERE class_attendance_session_url_deleted_at IS NULL;

COMMIT;
