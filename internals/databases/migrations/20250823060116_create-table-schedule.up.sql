-- =========================================================
-- Prasyarat
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- (masih ok untuk GiST index)

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
  UNIQUE (class_schedules_masjid_id, class_schedule_id)
);

-- =========================
-- Indexing jadwal (read pattern umum)
-- =========================
CREATE INDEX idx_class_schedules_tenant_dow
  ON class_schedules (class_schedules_masjid_id, class_schedules_day_of_week);

CREATE INDEX idx_class_schedules_active
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

CREATE INDEX idx_class_schedules_csst
  ON class_schedules (class_schedules_csst_id)
  WHERE class_schedules_deleted_at IS NULL;

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