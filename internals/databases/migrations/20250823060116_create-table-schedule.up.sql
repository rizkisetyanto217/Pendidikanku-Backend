-- +migrate Up
-- ======================================
-- EXTENSIONS & ENUMS
-- ======================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'week_parity_enum') THEN
    CREATE TYPE week_parity_enum AS ENUM ('all','odd','even');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_schedules (header jadwal)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant scope
  class_schedules_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- >>> SLUG (opsional; unik per tenant saat alive)
  class_schedules_slug VARCHAR(160),

  -- masa berlaku (header)
  class_schedules_start_date  DATE NOT NULL,
  class_schedules_end_date    DATE NOT NULL
    CHECK (class_schedules_end_date >= class_schedules_start_date),

  -- status & metadata
  class_schedules_status      session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedules_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_schedules_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_deleted_at TIMESTAMPTZ,

  -- tenant-safe key (buat FK komposit)
  UNIQUE (class_schedules_masjid_id, class_schedule_id)
);

-- Indexes (class_schedules)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_schedules_slug_per_tenant_alive
  ON class_schedules (class_schedules_masjid_id, lower(class_schedules_slug))
  WHERE class_schedules_deleted_at IS NULL
    AND class_schedules_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_class_schedules_slug_trgm_alive
  ON class_schedules USING GIN (lower(class_schedules_slug) gin_trgm_ops)
  WHERE class_schedules_deleted_at IS NULL
    AND class_schedules_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sched_tenant_alive
  ON class_schedules (class_schedules_masjid_id)
  WHERE class_schedules_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sched_active_alive
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sched_date_bounds_alive
  ON class_schedules (class_schedules_start_date, class_schedules_end_date)
  WHERE class_schedules_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_class_schedules_created_at
  ON class_schedules USING BRIN (class_schedules_created_at);



-- ======================================
-- TABLE: class_schedule_rules (slot mingguan)
-- ======================================
CREATE TABLE IF NOT EXISTS class_schedule_rules (
  class_schedule_rules_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant + pointer ke header (FK KOMPOSIT → tenant-safe)
  class_schedule_rule_masjid_id   UUID NOT NULL,
  class_schedule_rule_schedule_id UUID NOT NULL,
  FOREIGN KEY (class_schedule_rule_schedule_id, class_schedule_rule_masjid_id)
    REFERENCES class_schedules (class_schedule_id, class_schedules_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- pola per pekan
  class_schedule_rule_day_of_week INT  NOT NULL CHECK (class_schedule_rule_day_of_week BETWEEN 1 AND 7), -- ISO 1..7
  class_schedule_rule_start_time  TIME NOT NULL,
  class_schedule_rule_end_time    TIME NOT NULL CHECK (class_schedule_rule_end_time > class_schedule_rule_start_time),

  -- opsi pola
  class_schedule_rule_interval_weeks      INT  NOT NULL DEFAULT 1,
  class_schedule_rule_start_offset_weeks  INT  NOT NULL DEFAULT 0,
  class_schedule_rule_week_parity         week_parity_enum NOT NULL DEFAULT 'all',
  class_schedule_rule_weeks_of_month      INT[],
  class_schedule_rule_last_week_of_month  BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  class_schedule_rule_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_rule_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes (class_schedule_rules)
CREATE INDEX IF NOT EXISTS idx_csr_by_schedule_dow
  ON class_schedule_rules (class_schedule_rule_schedule_id, class_schedule_rule_day_of_week);

CREATE INDEX IF NOT EXISTS idx_csr_by_masjid
  ON class_schedule_rules (class_schedule_rule_masjid_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_csr_unique_slot_per_schedule
  ON class_schedule_rules (
    class_schedule_rule_schedule_id,
    class_schedule_rule_day_of_week,
    class_schedule_rule_start_time,
    class_schedule_rule_end_time
  );



-- =========================================================
-- TABLE: holidays (tenant-wide day-off / rentang libur)
-- =========================================================
CREATE TABLE IF NOT EXISTS holidays (
  holiday_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  holiday_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- SLUG (opsional; unik per tenant saat alive)
  holiday_slug VARCHAR(160),

  -- tanggal: satu hari (start=end) atau rentang
  holiday_start_date DATE NOT NULL,
  holiday_end_date   DATE NOT NULL CHECK (holiday_end_date >= holiday_start_date),

  holiday_title   VARCHAR(200) NOT NULL,
  holiday_reason  TEXT,

  holiday_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  -- untuk libur berulang (fixed-date tiap tahun, contoh 01-01)
  holiday_is_recurring_yearly BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  holiday_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  holiday_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  holiday_deleted_at TIMESTAMPTZ
);

-- Indexes (holidays)
CREATE UNIQUE INDEX IF NOT EXISTS uq_holiday_slug_per_tenant_alive
  ON holidays (holiday_masjid_id, lower(holiday_slug))
  WHERE holiday_deleted_at IS NULL
    AND holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_holiday_slug_trgm_alive
  ON holidays USING GIN (lower(holiday_slug) gin_trgm_ops)
  WHERE holiday_deleted_at IS NULL
    AND holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_holiday_tenant_alive
  ON holidays (holiday_masjid_id)
  WHERE holiday_deleted_at IS NULL AND holiday_is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_holiday_date_range_alive
  ON holidays (holiday_start_date, holiday_end_date)
  WHERE holiday_deleted_at IS NULL AND holiday_is_active = TRUE;

CREATE INDEX IF NOT EXISTS brin_holiday_created_at
  ON holidays USING BRIN (holiday_created_at);
