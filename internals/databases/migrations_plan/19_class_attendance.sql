BEGIN;

-- ================================
-- Prasyarat
-- ================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist; -- untuk EXCLUDE ops (jika dipakai nanti)

-- ================================
-- ENUM status sesi (idempotent)
-- ================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('planned','ongoing','completed','canceled');
  END IF;
END$$;

-- ================================
-- TABLE: CLASS_ATTENDANCE_SESSIONS
-- ================================
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant guard
  class_attendance_sessions_masjid_id UUID NOT NULL,

  -- Relasi utama: assignment (CSST)
  class_attendance_sessions_csst_id UUID NOT NULL
    REFERENCES class_section_subject_teachers (class_section_subject_teachers_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- Opsional: room
  class_attendance_sessions_class_room_id UUID
    REFERENCES class_rooms (class_room_id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Metadata dasar
  class_attendance_sessions_date DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note  TEXT,

  -- Guru (tetap/pengganti)
  class_attendance_sessions_teacher_id UUID
    REFERENCES masjid_teachers (masjid_teacher_id) ON DELETE SET NULL,

  -- Soft delete
  class_attendance_sessions_deleted_at TIMESTAMPTZ,

  -- =========================
  -- Improvement Columns
  -- =========================

  -- 1) Konteks akademik
  class_attendance_sessions_section_id       UUID,
  class_attendance_sessions_class_subject_id UUID,
  class_attendance_sessions_term_id          UUID,
  class_attendance_sessions_academic_year_id UUID,
  class_attendance_sessions_schedule_id      UUID,
  class_attendance_sessions_event_id         UUID,

  -- 2) Waktu & timezone
  class_attendance_sessions_timezone   TEXT,
  class_attendance_sessions_start_time TIME,
  class_attendance_sessions_end_time   TIME,
  class_attendance_sessions_started_at TIMESTAMPTZ,
  class_attendance_sessions_ended_at   TIMESTAMPTZ,

  -- Rentang menit (generated)
  class_attendance_sessions_time_range int4range
    GENERATED ALWAYS AS (
      CASE
        WHEN class_attendance_sessions_start_time IS NULL
          OR class_attendance_sessions_end_time   IS NULL
        THEN NULL
        ELSE int4range(
               (EXTRACT(HOUR FROM class_attendance_sessions_start_time)*60
              + EXTRACT(MINUTE FROM class_attendance_sessions_start_time))::int,
               (EXTRACT(HOUR FROM class_attendance_sessions_end_time)*60
              + EXTRACT(MINUTE FROM class_attendance_sessions_end_time))::int,
               '[)'
             )
      END
    ) STORED,

  class_attendance_sessions_duration_minutes INT
    GENERATED ALWAYS AS (
      CASE
        WHEN class_attendance_sessions_time_range IS NULL
        THEN NULL
        ELSE upper(class_attendance_sessions_time_range) - lower(class_attendance_sessions_time_range)
      END
    ) STORED,

  -- 3) Status & tipe
  class_attendance_sessions_status          session_status_enum NOT NULL DEFAULT 'planned',
  class_attendance_sessions_type            TEXT, -- 'regular','makeup','exam','remedial','event'
  class_attendance_sessions_is_locked       BOOLEAN DEFAULT FALSE,
  class_attendance_sessions_canceled_at     TIMESTAMPTZ,
  class_attendance_sessions_canceled_reason TEXT,

  -- 4) Pengajar tambahan
  class_attendance_sessions_substitute_teacher_id UUID,
  class_attendance_sessions_co_teacher_ids   UUID[],
  class_attendance_sessions_assistant_ids    UUID[],

  -- 5) Modality & meeting
  class_attendance_sessions_modality      TEXT, -- 'onsite','online','hybrid'
  class_attendance_sessions_meeting_url   TEXT,
  class_attendance_sessions_meeting_code  VARCHAR(64),
  class_attendance_sessions_is_recorded   BOOLEAN DEFAULT FALSE,
  class_attendance_sessions_recording_url TEXT,

  -- 6) Materi & lampiran
  class_attendance_sessions_materials         JSONB,
  class_attendance_sessions_attachments       JSONB,
  class_attendance_sessions_attachments_count INT,

  -- 7) Kehadiran & penilaian
  class_attendance_sessions_attendance_required    BOOLEAN DEFAULT TRUE,
  class_attendance_sessions_late_threshold_minutes SMALLINT,
  class_attendance_sessions_attendance_weight      NUMERIC(5,2),
  class_attendance_sessions_assessment_weight      NUMERIC(5,2),

  -- 8) Kapasitas
  class_attendance_sessions_expected_students INT,
  class_attendance_sessions_capacity          INT,
  class_attendance_sessions_waitlist_capacity INT,

  -- 9) Rekap hasil kehadiran
  class_attendance_sessions_present_count INT,
  class_attendance_sessions_absent_count  INT,
  class_attendance_sessions_late_count    INT,
  class_attendance_sessions_excused_count INT,
  class_attendance_sessions_sick_count    INT,
  class_attendance_sessions_leave_count   INT,

  -- 10) Publikasi & notifikasi
  class_attendance_sessions_is_published     BOOLEAN DEFAULT TRUE,
  class_attendance_sessions_publish_at       TIMESTAMPTZ,
  class_attendance_sessions_notify_on_create BOOLEAN DEFAULT FALSE,
  class_attendance_sessions_notify_on_change BOOLEAN DEFAULT FALSE,
  class_attendance_sessions_reminder_minutes_before INT[],
  class_attendance_sessions_reminder_channels       TEXT[],

  -- 11) Metadata
  class_attendance_sessions_tags  TEXT[],
  class_attendance_sessions_extra JSONB,

  -- 12) Audit & versioning
  class_attendance_sessions_created_by_user_id UUID,
  class_attendance_sessions_updated_by_user_id UUID,
  class_attendance_sessions_row_version        INT DEFAULT 1,
  class_attendance_sessions_etag               TEXT,

  -- 13) Timestamps
  class_attendance_sessions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- 14) Check-in/Check-out
  class_attendance_sessions_checkin_opens_at   TIMESTAMPTZ,
  class_attendance_sessions_checkin_closes_at  TIMESTAMPTZ,
  class_attendance_sessions_checkout_opens_at  TIMESTAMPTZ,
  class_attendance_sessions_checkout_closes_at TIMESTAMPTZ,
  class_attendance_sessions_checkin_radius_m   INT,
  class_attendance_sessions_checkin_qr_code    TEXT,
  class_attendance_sessions_checkin_policy     TEXT,

  -- 15) Lokasi
  class_attendance_sessions_location_name TEXT,
  class_attendance_sessions_location_lat  NUMERIC(9,6),
  class_attendance_sessions_location_lng  NUMERIC(9,6),

  -- 16) Quality metrics
  class_attendance_sessions_started_late_minutes SMALLINT,
  class_attendance_sessions_room_occupied_ratio  NUMERIC(5,2),
  class_attendance_sessions_anomaly_flags        TEXT[],

  -- 17) Lock & approval
  class_attendance_sessions_locked_at          TIMESTAMPTZ,
  class_attendance_sessions_locked_by_user_id  UUID,
  class_attendance_sessions_lock_reason        TEXT,
  class_attendance_sessions_approved_at        TIMESTAMPTZ,
  class_attendance_sessions_approved_by_user_id UUID,
  class_attendance_sessions_approval_note      TEXT,

  -- 18) Integrasi
  class_attendance_sessions_dedup_key TEXT,
  class_attendance_sessions_checksum  TEXT,

  -- 19) Proctoring
  class_attendance_sessions_proctor_ids      UUID[],
  class_attendance_sessions_proctoring_notes TEXT,

  -- 20) Kanal & perangkat
  class_attendance_sessions_created_via    TEXT,  -- 'web','mobile','api'
  class_attendance_sessions_creator_device JSONB, -- {ua,os,app_ver}
  class_attendance_sessions_creator_ip     INET
);

COMMIT;