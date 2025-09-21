-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard
  class_attendance_sessions_masjid_id   UUID NOT NULL,

  -- relasi utama: header jadwal (template)
  class_attendance_sessions_schedule_id UUID NOT NULL,

  -- FK komposit tenant-safe → schedules (butuh UNIQUE (class_schedules_masjid_id, class_schedule_id) di class_schedules)
  CONSTRAINT fk_cas_schedule_tenant
    FOREIGN KEY (class_attendance_sessions_masjid_id, class_attendance_sessions_schedule_id)
    REFERENCES class_schedules (class_schedules_masjid_id, class_schedule_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- (opsional) jejak rule (slot mingguan) asal occurrence
  class_attendance_sessions_rule_id UUID
    REFERENCES class_schedule_rules(class_schedule_rules_id) ON DELETE SET NULL,

  -- >>> SLUG (opsional; unik per tenant saat alive)
  class_attendance_sessions_slug VARCHAR(160),

  -- occurrence
  class_attendance_sessions_date      DATE NOT NULL,
  class_attendance_sessions_starts_at TIMESTAMPTZ,
  class_attendance_sessions_ends_at   TIMESTAMPTZ,

  -- lifecycle
  class_attendance_sessions_status            session_status_enum NOT NULL DEFAULT 'scheduled', -- scheduled|ongoing|completed|canceled
  class_attendance_sessions_attendance_status TEXT NOT NULL DEFAULT 'open',                      -- open|closed
  class_attendance_sessions_locked            BOOLEAN NOT NULL DEFAULT FALSE,

  -- OVERRIDES (ubah harian tanpa mengubah rules)
  class_attendance_sessions_is_override BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_sessions_is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_sessions_original_start_at TIMESTAMPTZ,
  class_attendance_sessions_original_end_at   TIMESTAMPTZ,
  class_attendance_sessions_kind TEXT,                    -- 'lesson','ceremony','counseling','exam', ...
  class_attendance_sessions_override_reason TEXT,

  -- Override karena EVENT (opsional)
  class_attendance_sessions_override_event_id UUID
    REFERENCES class_events(class_events_id) ON DELETE SET NULL,
  class_attendance_sessions_override_attendance_event_id UUID
    REFERENCES class_attendance_events(class_attendance_events_id) ON DELETE SET NULL,

  -- override resource (opsional)
  class_attendance_sessions_teacher_id    UUID REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,
  class_attendance_sessions_class_room_id UUID REFERENCES class_rooms(class_room_id)         ON DELETE SET NULL,
  class_attendance_sessions_csst_id       UUID REFERENCES class_section_subject_teachers(class_section_subject_teachers_id) ON DELETE SET NULL,

  -- info & rekap
  class_attendance_sessions_title         TEXT,
  class_attendance_sessions_general_info  TEXT NOT NULL DEFAULT '',
  class_attendance_sessions_note          TEXT,
  class_attendance_sessions_present_count INT,
  class_attendance_sessions_absent_count  INT,
  class_attendance_sessions_late_count    INT,
  class_attendance_sessions_excused_count INT,
  class_attendance_sessions_sick_count    INT,
  class_attendance_sessions_leave_count   INT,

  -- audit
  class_attendance_sessions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_deleted_at TIMESTAMPTZ,

  -- CHECKS (tanpa trigger/func)
  CONSTRAINT chk_cas_time_order
    CHECK (
      class_attendance_sessions_starts_at IS NULL
      OR class_attendance_sessions_ends_at IS NULL
      OR class_attendance_sessions_ends_at >= class_attendance_sessions_starts_at
    ),
  CONSTRAINT chk_cas_time_order_original
    CHECK (
      class_attendance_sessions_original_start_at IS NULL
      OR class_attendance_sessions_original_end_at IS NULL
      OR class_attendance_sessions_original_end_at >= class_attendance_sessions_original_start_at
    ),
  CONSTRAINT chk_cas_attendance_status
    CHECK (class_attendance_sessions_attendance_status IN ('open','closed')),
  CONSTRAINT chk_cas_override_event_requires_flag
    CHECK (
      class_attendance_sessions_override_event_id IS NULL
      OR class_attendance_sessions_is_override = TRUE
    )
);

-- =========================================================
-- Indexes
-- =========================================================

-- SLUG unik per tenant (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_slug_per_tenant_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    lower(class_attendance_sessions_slug)
  )
  WHERE class_attendance_sessions_deleted_at IS NULL
    AND class_attendance_sessions_slug IS NOT NULL;

-- (opsional) pencarian slug cepat
CREATE INDEX IF NOT EXISTS gin_cas_slug_trgm_alive
  ON class_attendance_sessions USING GIN (lower(class_attendance_sessions_slug) gin_trgm_ops)
  WHERE class_attendance_sessions_deleted_at IS NULL
    AND class_attendance_sessions_slug IS NOT NULL;

-- Satu baris per (tenant, schedule, date) yang masih hidup
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_masjid_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_schedule_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Kalender per tenant
CREATE INDEX IF NOT EXISTS idx_cas_masjid_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Per schedule (ambil range tanggal cepat)
CREATE INDEX IF NOT EXISTS idx_cas_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_schedule_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Lookup per guru
CREATE INDEX IF NOT EXISTS idx_cas_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_teacher_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Quality-of-life untuk operasi mass override/cancel
CREATE INDEX IF NOT EXISTS idx_cas_canceled_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_is_canceled,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_override_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_is_override,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Override event lookup
CREATE INDEX IF NOT EXISTS idx_cas_override_event_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_override_event_id
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Join cepat ke rule (opsional)
CREATE INDEX IF NOT EXISTS idx_cas_rule_alive
  ON class_attendance_sessions (
    class_attendance_sessions_rule_id
  )
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
