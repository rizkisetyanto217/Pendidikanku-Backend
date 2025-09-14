BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist;   -- (opsional untuk EXCLUDE)

-- Enum status sesi (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
END$$;

-- =========================================================
-- CLASS_SCHEDULES — jadwal rutin (FINAL)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi inti
  class_schedules_masjid_id           UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_schedules_section_id          UUID REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,
  class_schedules_class_subject_id    UUID REFERENCES class_subjects(class_subjects_id) ON DELETE RESTRICT,
  class_schedules_csst_id             UUID REFERENCES class_section_subject_teachers(class_section_subject_teachers_id) ON UPDATE CASCADE ON DELETE SET NULL,
  class_schedules_room_id             UUID REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,
  class_schedules_teacher_id          UUID REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- pola rutin (weekly)
  class_schedules_day_of_week         INT  NOT NULL CHECK (class_schedules_day_of_week BETWEEN 1 AND 7),
  class_schedules_start_time          TIME NOT NULL,
  class_schedules_end_time            TIME NOT NULL CHECK (class_schedules_end_time > class_schedules_start_time),
  class_schedules_start_date          DATE NOT NULL,
  class_schedules_end_date            DATE NOT NULL CHECK (class_schedules_end_date >= class_schedules_start_date),

  -- status
  class_schedules_status              session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedules_is_active           BOOLEAN NOT NULL DEFAULT TRUE,

  -- rentang & durasi (menit)
  class_schedules_time_range          int4range
    GENERATED ALWAYS AS (
      int4range(
        (EXTRACT(HOUR FROM class_schedules_start_time)*60 + EXTRACT(MINUTE FROM class_schedules_start_time))::int,
        (EXTRACT(HOUR FROM class_schedules_end_time)*60   + EXTRACT(MINUTE FROM class_schedules_end_time))::int,
        '[)'
      )
    ) STORED,
  class_schedules_duration_minutes    INT
    GENERATED ALWAYS AS (upper(class_schedules_time_range) - lower(class_schedules_time_range)) STORED,

  -- kalender & recurring lanjutan
  class_schedules_timezone            TEXT,
  class_schedules_rrule               TEXT,      -- RFC5545 (opsional untuk pola khusus)
  class_schedules_exdates             DATE[],    -- pengecualian
  class_schedules_rdates              DATE[],    -- tambahan manual
  class_schedules_skip_holidays       BOOLEAN DEFAULT TRUE,

  -- metadata sesi
  class_schedules_title               VARCHAR(160),
  class_schedules_topic               TEXT,
  class_schedules_session_type        TEXT,      -- 'lecture'|'lab'|'exam'|'makeup'|...
  class_schedules_lesson_plan_id      UUID,
  class_schedules_syllabus_ref        TEXT,
  class_schedules_learning_objectives TEXT[],

  -- materi & lampiran
  class_schedules_materials           JSONB,     -- {links:[], files:[]}
  class_schedules_attachments         JSONB,
  class_schedules_attachments_count   INT,

  -- modality & meeting
  class_schedules_modality            TEXT,      -- 'onsite'|'online'|'hybrid'
  class_schedules_meeting_url         TEXT,
  class_schedules_meeting_code        VARCHAR(64),
  class_schedules_is_recorded         BOOLEAN DEFAULT FALSE,
  class_schedules_recording_url       TEXT,

  -- pengajar & kapasitas
  class_schedules_co_teacher_ids      UUID[],
  class_schedules_assistant_ids       UUID[],
  class_schedules_capacity            INT,
  class_schedules_waitlist_capacity   INT,
  class_schedules_enrollment_policy   TEXT,      -- 'open'|'invite'|'closed'

  -- kehadiran & penilaian
  class_schedules_attendance_required       BOOLEAN DEFAULT TRUE,
  class_schedules_attendance_weight         NUMERIC(5,2),
  class_schedules_late_threshold_minutes    SMALLINT,
  class_schedules_absence_policy            TEXT,

  -- visibilitas & publikasi
  class_schedules_visibility_scope    TEXT,      -- 'tenant'|'campus'|'class'|'section'
  class_schedules_is_published        BOOLEAN DEFAULT TRUE,
  class_schedules_publish_at          TIMESTAMPTZ,
  class_schedules_notify_on_create    BOOLEAN DEFAULT FALSE,
  class_schedules_notify_on_change    BOOLEAN DEFAULT FALSE,
  class_schedules_reminder_minutes_before INT[], -- {1440,60,10}
  class_schedules_reminder_channels   TEXT[],    -- ['push','email','wa','sms']

  -- pembatalan & reschedule (jejak)
  class_schedules_canceled_at         TIMESTAMPTZ,
  class_schedules_canceled_reason     TEXT,
  class_schedules_rescheduled_from_id UUID,
  class_schedules_rescheduled_at      TIMESTAMPTZ,

  -- fasilitas ruangan
  class_schedules_room_setup          TEXT,
  class_schedules_equipment_required  TEXT[],
  class_schedules_equipment_provided  TEXT[],

  -- meta umum
  class_schedules_tags                TEXT[],
  class_schedules_external_ref        TEXT,
  class_schedules_notes               TEXT,
  class_schedules_extra               JSONB,

  -- audit & optimistic locking
  class_schedules_created_by_user_id  UUID,
  class_schedules_updated_by_user_id  UUID,
  class_schedules_deleted_by_user_id  UUID,
  class_schedules_row_version         INT DEFAULT 1,
  class_schedules_etag                TEXT,

  -- timestamps
  class_schedules_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_deleted_at          TIMESTAMPTZ
);

-- =========================================================
-- CLASS_SCHEDULE_OVERRIDES — override occurrence (FINAL)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedule_overrides (
  class_schedule_overrides_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_schedule_overrides_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_schedule_overrides_schedule_id UUID NOT NULL REFERENCES class_schedules(class_schedule_id) ON DELETE CASCADE,

  -- occurrence yang terkena
  class_schedule_overrides_date    DATE NOT NULL,

  -- jenis perubahan
  class_schedule_overrides_kind    TEXT NOT NULL, -- 'cancel'|'move'|'extend'|'shorten'|'room_change'
  class_schedule_overrides_reason  TEXT,

  -- perubahan jika kind ≠ 'cancel'
  class_schedule_overrides_new_room_id     UUID,
  class_schedule_overrides_new_start_time  TIME,
  class_schedule_overrides_new_end_time    TIME,
  class_schedule_overrides_new_teacher_id  UUID,
  class_schedule_overrides_new_meeting_url TEXT,
  class_schedule_overrides_new_room_setup  TEXT,
  class_schedule_overrides_new_equipment   TEXT[],

  -- notifikasi & aplikasi
  class_schedule_overrides_notify_impacted BOOLEAN DEFAULT FALSE,
  class_schedule_overrides_applied_by_user_id UUID,
  class_schedule_overrides_applied_at      TIMESTAMPTZ,

  -- meta
  class_schedule_overrides_extra   JSONB,

  -- audit
  class_schedule_overrides_created_by_user_id UUID,
  class_schedule_overrides_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_overrides_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- CLASS_EVENTS — event ad-hoc/special (FINAL)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_events (
  class_events_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_events_masjid_id     UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- target opsional
  class_events_section_id       UUID,
  class_events_class_id         UUID,
  class_events_class_subject_id UUID,

  -- audiens fleksibel
  class_events_is_global        BOOLEAN DEFAULT FALSE,
  class_events_section_ids      UUID[],
  class_events_class_ids        UUID[],
  class_events_grade_levels     INT[],
  class_events_audience_tags    TEXT[],

  -- info event
  class_events_title            VARCHAR(160) NOT NULL,
  class_events_subtitle         VARCHAR(160),
  class_events_desc             TEXT,
  class_events_category         TEXT,

  -- waktu & zona
  class_events_timezone         TEXT,
  class_events_date             DATE NOT NULL,  -- start
  class_events_end_date         DATE,           -- end (opsional multi-hari)
  class_events_start_time       TIME,
  class_events_end_time         TIME,
  class_events_is_all_day       BOOLEAN DEFAULT FALSE,
  class_events_duration_minutes INT,            -- bantu (jika bukan all-day)

  -- lokasi / modality
  class_events_room_id          UUID,
  class_events_modality         TEXT,           -- 'onsite'|'online'|'hybrid'
  class_events_meeting_url      TEXT,
  class_events_meeting_code     VARCHAR(64),
  class_events_is_recorded      BOOLEAN DEFAULT FALSE,
  class_events_recording_url    TEXT,

  -- pengisi acara
  class_events_teacher_id       UUID,
  class_events_co_teacher_ids   UUID[],
  class_events_organizer_user_id UUID,

  -- kapasitas & RSVP
  class_events_capacity         INT,
  class_events_waitlist_capacity INT,
  class_events_enrollment_policy TEXT,          -- 'open'|'invite'|'closed'
  class_events_rsvp_required    BOOLEAN DEFAULT FALSE,
  class_events_require_attendance BOOLEAN DEFAULT FALSE,
  class_events_checkin_code     VARCHAR(16),

  -- publikasi & reminder
  class_events_is_published     BOOLEAN DEFAULT TRUE,
  class_events_publish_at       TIMESTAMPTZ,
  class_events_notify_on_create BOOLEAN DEFAULT FALSE,
  class_events_notify_on_change BOOLEAN DEFAULT FALSE,
  class_events_reminder_minutes_before INT[],
  class_events_reminder_channels TEXT[],

  -- media & lampiran
  class_events_banner_url       TEXT,
  class_events_attachments      JSONB,
  class_events_attachments_count INT,

  -- status & pembatalan
  class_events_status           TEXT,          -- 'planned'|'ongoing'|'completed'|'canceled'
  class_events_canceled_at      TIMESTAMPTZ,
  class_events_canceled_reason  TEXT,

  -- meta
  class_events_external_ref     TEXT,
  class_events_tags             TEXT[],
  class_events_notes            TEXT,
  class_events_extra            JSONB,

  -- audit & locking
  class_events_created_by_user_id UUID,
  class_events_updated_by_user_id UUID,
  class_events_deleted_by_user_id UUID,
  class_events_row_version      INT DEFAULT 1,
  class_events_etag             TEXT,

  class_events_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_deleted_at       TIMESTAMPTZ
);

-- =========================================================
-- CLASS_EVENT_ATTENDEES — RSVP/kehadiran event (FINAL)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_event_attendees (
  class_event_attendees_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_attendees_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_event_attendees_event_id  UUID NOT NULL REFERENCES class_events(class_events_id) ON DELETE CASCADE,

  -- identitas (isi salah satu)
  class_event_attendees_user_id           UUID,
  class_event_attendees_masjid_student_id UUID,
  class_event_attendees_guardian_id       UUID,

  -- RSVP & kehadiran
  class_event_attendees_rsvp_status       TEXT,        -- 'invited'|'going'|'maybe'|'declined'|'waitlist'
  class_event_attendees_checked_in_at     TIMESTAMPTZ,
  class_event_attendees_no_show           BOOLEAN DEFAULT FALSE,

  -- tiket & notifikasi
  class_event_attendees_ticket_code       VARCHAR(64),
  class_event_attendees_ticket_qr_url     TEXT,
  class_event_attendees_reminded_at       TIMESTAMPTZ,

  -- catatan
  class_event_attendees_note              TEXT,
  class_event_attendees_extra             JSONB,

  -- audit
  class_event_attendees_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_attendees_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_attendees_deleted_at        TIMESTAMPTZ
);

COMMIT;