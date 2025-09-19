BEGIN;

-- =========================================================
-- Prasyarat
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- GiST equality ops utk EXCLUDE (=, &&)

-- Enum status jadwal (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
END$$;

-- =========================================================
-- Tabel: CLASS_SCHEDULES (kompatibel dgn class_events)
-- =========================================================
CREATE TABLE class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_schedules_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- induk jadwal → section
  class_schedules_section_id UUID
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- konteks kurikulum (kelas+mapel[+term]) → class_subjects
  class_schedules_class_subject_id UUID
    REFERENCES class_subjects(class_subjects_id) ON DELETE RESTRICT,

  -- assignment CSST (section+class_subject+teacher) — opsional
  class_schedules_csst_id UUID
    REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- room (nullable)
  class_schedules_room_id UUID
    REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- cache guru (opsional)
  class_schedules_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- relasi ke EVENT (opsional; satu event terkait jadwal ini)
  class_schedules_event_id UUID
    REFERENCES class_events(class_events_id) ON DELETE SET NULL,

  -- pola berulang
  class_schedules_day_of_week INT  NOT NULL
    CHECK (class_schedules_day_of_week BETWEEN 1 AND 7),
  class_schedules_start_time  TIME NOT NULL,
  class_schedules_end_time    TIME NOT NULL
    CHECK (class_schedules_end_time > class_schedules_start_time),

  -- batas berlaku
  class_schedules_start_date  DATE NOT NULL,
  class_schedules_end_date    DATE NOT NULL
    CHECK (class_schedules_end_date >= class_schedules_start_date),

  -- status & metadata
  class_schedules_status    session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedules_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- rentang menit [start, end)
  class_schedules_time_range int4range
    GENERATED ALWAYS AS (
      int4range(
        ((EXTRACT(HOUR FROM class_schedules_start_time))::int * 60
          + (EXTRACT(MINUTE FROM class_schedules_start_time))::int),
        ((EXTRACT(HOUR FROM class_schedules_end_time))::int * 60
          + (EXTRACT(MINUTE FROM class_schedules_end_time))::int),
        '[)'
      )
    ) STORED,

  -- timestamps
  class_schedules_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_deleted_at TIMESTAMPTZ,

  -- Tenant-safe composite uniqueness (untuk FK CAS)
  UNIQUE (class_schedules_masjid_id, class_schedule_id),

  -- =========================
  -- Exclusion Constraints (anti-bentrok inti)
  -- =========================
  CONSTRAINT excl_sched_room_overlap
    EXCLUDE USING gist (
      class_schedules_masjid_id   WITH =,
      class_schedules_room_id     WITH =,
      class_schedules_day_of_week WITH =,
      class_schedules_time_range  WITH &&
    )
    WHERE (class_schedules_is_active
           AND class_schedules_room_id IS NOT NULL
           AND class_schedules_deleted_at IS NULL),

  CONSTRAINT excl_sched_section_overlap
    EXCLUDE USING gist (
      class_schedules_masjid_id   WITH =,
      class_schedules_section_id  WITH =,
      class_schedules_day_of_week WITH =,
      class_schedules_time_range  WITH &&
    )
    WHERE (class_schedules_is_active
           AND class_schedules_deleted_at IS NULL),

  CONSTRAINT excl_sched_teacher_overlap
    EXCLUDE USING gist (
      class_schedules_masjid_id   WITH =,
      class_schedules_teacher_id  WITH =,
      class_schedules_day_of_week WITH =,
      class_schedules_time_range  WITH &&
    )
    WHERE (class_schedules_is_active
           AND class_schedules_teacher_id IS NOT NULL
           AND class_schedules_deleted_at IS NULL),

  -- Bentuk link yang valid (CSST penuh ATAU pair section+class_subject)
  CONSTRAINT ck_class_schedules_link_shape
  CHECK (
    (class_schedules_csst_id IS NOT NULL AND class_schedules_section_id IS NULL AND class_schedules_class_subject_id IS NULL)
    OR
    (class_schedules_csst_id IS NULL AND class_schedules_section_id IS NOT NULL AND class_schedules_class_subject_id IS NOT NULL)
  )
);

-- =========================
-- Indexing jadwal (read pattern umum)
-- =========================
CREATE INDEX idx_class_schedules_tenant_dow
  ON class_schedules (class_schedules_masjid_id, class_schedules_day_of_week);

CREATE INDEX idx_class_schedules_section_dow_time
  ON class_schedules (class_schedules_section_id, class_schedules_day_of_week, class_schedules_start_time, class_schedules_end_time);

CREATE INDEX idx_class_schedules_room_dow
  ON class_schedules (class_schedules_room_id, class_schedules_day_of_week);

CREATE INDEX idx_class_schedules_class_subject
  ON class_schedules (class_schedules_class_subject_id);

CREATE INDEX idx_class_schedules_active
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

CREATE INDEX idx_class_schedules_teacher_dow
  ON class_schedules (class_schedules_teacher_id, class_schedules_day_of_week);

CREATE INDEX idx_class_schedules_teacher
  ON class_schedules (class_schedules_teacher_id);

CREATE INDEX idx_class_schedules_csst
  ON class_schedules (class_schedules_csst_id)
  WHERE class_schedules_deleted_at IS NULL;

CREATE INDEX idx_sched_masjid_section_alive
  ON class_schedules (class_schedules_masjid_id, class_schedules_section_id)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

CREATE INDEX idx_sched_masjid_teacher_alive
  ON class_schedules (class_schedules_masjid_id, class_schedules_teacher_id)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

CREATE INDEX idx_sched_masjid_subject_alive
  ON class_schedules (class_schedules_masjid_id, class_schedules_class_subject_id)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

-- lookup event
CREATE INDEX idx_class_schedules_event
  ON class_schedules (class_schedules_event_id);

-- GiST utk pencarian slot by hari+range
CREATE INDEX gist_sched_dow_timerange
  ON class_schedules
  USING gist (class_schedules_day_of_week, class_schedules_time_range);

-- Per tanggal (jika sering filter by date-range)
CREATE INDEX idx_sched_date_bounds
  ON class_schedules (class_schedules_start_date, class_schedules_end_date);