-- +migrate Up
-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
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
  class_schedule_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- SLUG (opsional; unik per tenant saat alive)
  class_schedule_slug VARCHAR(160),

  -- masa berlaku
  class_schedule_start_date DATE NOT NULL,
  class_schedule_end_date   DATE NOT NULL
    CHECK (class_schedule_end_date >= class_schedule_start_date),

  -- status & metadata
  class_schedule_status    session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedule_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_schedule_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_deleted_at TIMESTAMPTZ,

  -- tenant-safe key (buat FK komposit)
  UNIQUE (class_schedule_school_id, class_schedule_id)
);

-- Indexes (class_schedules)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_schedules_slug_per_tenant_alive
  ON class_schedules (class_schedule_school_id, LOWER(class_schedule_slug))
  WHERE class_schedule_deleted_at IS NULL
    AND class_schedule_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_class_schedules_slug_trgm_alive
  ON class_schedules USING GIN (LOWER(class_schedule_slug) gin_trgm_ops)
  WHERE class_schedule_deleted_at IS NULL
    AND class_schedule_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_tenant_alive
  ON class_schedules (class_schedule_school_id)
  WHERE class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_active_alive
  ON class_schedules (class_schedule_is_active)
  WHERE class_schedule_is_active = TRUE
    AND class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_date_bounds_alive
  ON class_schedules (class_schedule_start_date, class_schedule_end_date)
  WHERE class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_class_schedules_created_at
  ON class_schedules USING BRIN (class_schedule_created_at);

-- =========================================================
-- Tambahan UNIQUE untuk urutan kolom yang cocok dengan FK (id, school)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_schedules_id_school') THEN
    ALTER TABLE class_schedules
      ADD CONSTRAINT uq_class_schedules_id_school
      UNIQUE (class_schedule_id, class_schedule_school_id);
  END IF;
END$$;

-- =========================================================
-- Pastikan CSST punya UNIQUE (id, school) untuk FK komposit
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_csst_id_school') THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT uq_csst_id_school
      UNIQUE (class_section_subject_teacher_id, class_section_subject_teacher_school_id);
  END IF;
END$$;

-- =========================================================
-- TABLE: class_schedule_rules (single-tenant column; no trigger)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedule_rules (
  class_schedule_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant + pointer ke header (FK komposit → tenant-safe di header)
  class_schedule_rule_school_id   UUID NOT NULL,
  class_schedule_rule_schedule_id UUID NOT NULL,
  CONSTRAINT fk_csr_schedule_tenant
    FOREIGN KEY (class_schedule_rule_schedule_id, class_schedule_rule_school_id)
    REFERENCES class_schedules (class_schedule_id, class_schedule_school_id)
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

  -- DEFAULT PENUGASAN: CSST (FK satu kolom)
  class_schedule_rule_csst_id            UUID NOT NULL,
  class_schedule_rule_csst_slug_cache    VARCHAR(100),

  -- ===== Cache CSST (diisi backend) =====
  class_schedule_rule_csst_cache JSONB NOT NULL,

  -- ===== Generated columns dari cache =====
  class_schedule_rule_csst_teacher_id   UUID
    GENERATED ALWAYS AS ((class_schedule_rule_csst_cache->>'teacher_id')::uuid) STORED,
  class_schedule_rule_csst_class_section_id UUID
    GENERATED ALWAYS AS ((class_schedule_rule_csst_cache->>'section_id')::uuid) STORED,
  class_schedule_rule_csst_class_subject_id UUID
    GENERATED ALWAYS AS ((class_schedule_rule_csst_cache->>'class_subject_id')::uuid) STORED,
  class_schedule_rule_csst_class_room_id UUID
    GENERATED ALWAYS AS ((class_schedule_rule_csst_cache->>'room_id')::uuid) STORED,

  -- Ambil school_id dari cache untuk guard tenant
  class_schedule_rule_csst_school_id_from_cache UUID
    GENERATED ALWAYS AS ((class_schedule_rule_csst_cache->>'school_id')::uuid) STORED,

  -- audit
  class_schedule_rule_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_rule_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_rule_deleted_at TIMESTAMPTZ,

  -- waktu (menit) untuk exclusion
  class_schedule_rule_start_min SMALLINT GENERATED ALWAYS AS (
    (EXTRACT(HOUR FROM class_schedule_rule_start_time)::INT * 60)
    + EXTRACT(MINUTE FROM class_schedule_rule_start_time)::INT
  ) STORED,
  class_schedule_rule_end_min SMALLINT GENERATED ALWAYS AS (
    (EXTRACT(HOUR FROM class_schedule_rule_end_time)::INT * 60)
    + EXTRACT(MINUTE FROM class_schedule_rule_end_time)::INT
  ) STORED,

  -- ===== Tenant guard via cache (tanpa trigger)
  CONSTRAINT ck_csr_cache_tenant_guard CHECK (
    (class_schedule_rule_csst_cache ? 'school_id')
    AND (class_schedule_rule_csst_school_id_from_cache = class_schedule_rule_school_id)
  )
);

-- FK ke CSST (single-column; PK CSST adalah UUID global)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_csr_csst_singlecol') THEN
    ALTER TABLE class_schedule_rules
      ADD CONSTRAINT fk_csr_csst_singlecol
      FOREIGN KEY (class_schedule_rule_csst_id)
      REFERENCES class_section_subject_teachers (class_section_subject_teacher_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_csr_by_schedule_dow
  ON class_schedule_rules (class_schedule_rule_schedule_id, class_schedule_rule_day_of_week)
  WHERE class_schedule_rule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csr_by_school
  ON class_schedule_rules (class_schedule_rule_school_id)
  WHERE class_schedule_rule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csr_deleted_at
  ON class_schedule_rules (class_schedule_rule_deleted_at);

-- Unik: cegah duplikasi slot persis untuk CSST yang sama dalam satu schedule
CREATE UNIQUE INDEX IF NOT EXISTS uq_csr_unique_slot_per_schedule_csst
  ON class_schedule_rules (
    class_schedule_rule_schedule_id,
    class_schedule_rule_csst_id,
    class_schedule_rule_day_of_week,
    class_schedule_rule_start_time,
    class_schedule_rule_end_time
  )
  WHERE class_schedule_rule_deleted_at IS NULL;

-- Exclusion constraint: cegah overlap waktu per (schedule, CSST, hari)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ex_csr_no_overlap_per_csst') THEN
    ALTER TABLE class_schedule_rules
      ADD CONSTRAINT ex_csr_no_overlap_per_csst
      EXCLUDE USING gist (
        class_schedule_rule_schedule_id WITH =,
        class_schedule_rule_csst_id     WITH =,
        class_schedule_rule_day_of_week WITH =,
        int4range(class_schedule_rule_start_min, class_schedule_rule_end_min, '[]') WITH &&
      )
      WHERE (class_schedule_rule_deleted_at IS NULL);
  END IF;
END$$;
