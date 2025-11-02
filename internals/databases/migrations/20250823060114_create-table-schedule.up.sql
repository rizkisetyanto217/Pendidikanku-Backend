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

-- +migrate Up
-- =========================================================
-- UP — class_schedule_rules + CSST snapshot
-- =========================================================
BEGIN;

-- ===== Enums =====
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'week_parity_enum') THEN
    CREATE TYPE week_parity_enum AS ENUM ('all','odd','even');
  END IF;
END$$;

-- ===== Tenant-safe uniqueness di tabel referensi =====
-- class_schedules → untuk FK komposit (id, school)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_schedules_id_school') THEN
    ALTER TABLE class_schedules
      ADD CONSTRAINT uq_class_schedules_id_school
      UNIQUE (class_schedule_id, class_schedule_school_id);
  END IF;
END$$;

-- class_section_subject_teachers (CSST) → FK komposit (id, school)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_csst_id_school') THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT uq_csst_id_school
      UNIQUE (class_section_subject_teacher_id, class_section_subject_teacher_school_id);
  END IF;
END$$;

-- =========================================================
-- TABLE: class_schedule_rules (slot mingguan) + link ke CSST
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedule_rules (
  class_schedule_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant + pointer ke header (FK komposit → tenant-safe)
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

  -- DEFAULT PENUGASAN: CSST (tenant-safe, WAJIB)
  class_schedule_rule_csst_id        UUID NOT NULL,
  class_schedule_rule_csst_school_id UUID NOT NULL,

  -- ===== Snapshot CSST (denormalized) =====
  class_schedule_rule_csst_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  -- ===== Generated columns dari snapshot (untuk query cepat)
  class_schedule_rule_csst_teacher_id       UUID GENERATED ALWAYS AS ((class_schedule_rule_csst_snapshot->>'teacher_id')::uuid) STORED,
  class_schedule_rule_csst_section_id       UUID GENERATED ALWAYS AS ((class_schedule_rule_csst_snapshot->>'section_id')::uuid) STORED,
  class_schedule_rule_csst_class_subject_id UUID GENERATED ALWAYS AS ((class_schedule_rule_csst_snapshot->>'class_subject_id')::uuid) STORED,
  class_schedule_rule_csst_room_id          UUID GENERATED ALWAYS AS ((class_schedule_rule_csst_snapshot->>'room_id')::uuid) STORED,

  -- audit
  class_schedule_rule_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_rule_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_rule_deleted_at TIMESTAMPTZ,

  -- ===== Generated untuk anti-overlap (EXCLUDE gist) =====
  class_schedule_rule_start_min SMALLINT GENERATED ALWAYS AS (
    (EXTRACT(HOUR FROM class_schedule_rule_start_time)::INT * 60)
    + EXTRACT(MINUTE FROM class_schedule_rule_start_time)::INT
  ) STORED,
  class_schedule_rule_end_min SMALLINT GENERATED ALWAYS AS (
    (EXTRACT(HOUR FROM class_schedule_rule_end_time)::INT * 60)
    + EXTRACT(MINUTE FROM class_schedule_rule_end_time)::INT
  ) STORED
);

-- FK ke CSST (komposit, tenant-safe)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_csr_csst_tenant') THEN
    ALTER TABLE class_schedule_rules
      ADD CONSTRAINT fk_csr_csst_tenant
      FOREIGN KEY (class_schedule_rule_csst_id, class_schedule_rule_csst_school_id)
      REFERENCES class_section_subject_teachers (class_section_subject_teacher_id, class_section_subject_teacher_school_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- Indexes (class_schedule_rules)
CREATE INDEX IF NOT EXISTS idx_csr_by_schedule_dow
  ON class_schedule_rules (class_schedule_rule_schedule_id, class_schedule_rule_day_of_week)
  WHERE class_schedule_rule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csr_by_school
  ON class_schedule_rules (class_schedule_rule_school_id)
  WHERE class_schedule_rule_deleted_at IS NULL;

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

CREATE INDEX IF NOT EXISTS idx_csr_deleted_at
  ON class_schedule_rules (class_schedule_rule_deleted_at);

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

-- =========================================================
-- Snapshot builder dari CSST → JSONB (dipakai di triggers)
-- =========================================================
CREATE OR REPLACE FUNCTION build_csst_snapshot(p_csst_id UUID, p_school_id UUID)
RETURNS JSONB
LANGUAGE plpgsql AS $$
DECLARE
  r RECORD;
BEGIN
  SELECT
    t.class_section_subject_teacher_id               AS csst_id,
    t.class_section_subject_teacher_school_id        AS school_id,
    t.class_section_subject_teacher_teacher_id       AS teacher_id,
    t.class_section_subject_teacher_section_id       AS section_id,
    t.class_section_subject_teacher_class_subject_id AS class_subject_id,
    t.class_section_subject_teacher_room_id          AS room_id,
    t.class_section_subject_teacher_group_url        AS group_url,
    t.class_section_subject_teacher_slug             AS slug,
    t.class_section_subject_teacher_description      AS description
  INTO r
  FROM class_section_subject_teachers t
  WHERE t.class_section_subject_teacher_id = p_csst_id
    AND t.class_section_subject_teacher_school_id = p_school_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'CSST % (school %) not found', p_csst_id, p_school_id;
  END IF;

  RETURN jsonb_build_object(
    'csst_id',          r.csst_id,
    'school_id',        r.school_id,
    'teacher_id',       r.teacher_id,
    'section_id',       r.section_id,
    'class_subject_id', r.class_subject_id,
    'room_id',          r.room_id,
    'group_url',        r.group_url,
    'slug',             r.slug,
    'description',      r.description
  );
END $$;

-- =========================================================
-- Trigger: isi/refresh snapshot di class_schedule_rules
-- (tanpa menyentuh created_at/updated_at)
-- =========================================================
CREATE OR REPLACE FUNCTION trg_csr_fill_csst_snapshot()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  -- tenant guard: school rule harus sama dengan school CSST
  IF NEW.class_schedule_rule_school_id <> NEW.class_schedule_rule_csst_school_id THEN
    RAISE EXCEPTION 'School mismatch: rule(%) vs csst(%)',
      NEW.class_schedule_rule_school_id, NEW.class_schedule_rule_csst_school_id;
  END IF;

  NEW.class_schedule_rule_csst_snapshot :=
    build_csst_snapshot(
      NEW.class_schedule_rule_csst_id,
      NEW.class_schedule_rule_csst_school_id
    );

  RETURN NEW; -- tidak mutasi timestamp
END $$;

DROP TRIGGER IF EXISTS trg_csr_fill_csst_snapshot_biu ON class_schedule_rules;
CREATE TRIGGER trg_csr_fill_csst_snapshot_biu
BEFORE INSERT OR UPDATE OF
  class_schedule_rule_csst_id,
  class_schedule_rule_csst_school_id
ON class_schedule_rules
FOR EACH ROW
EXECUTE FUNCTION trg_csr_fill_csst_snapshot();

-- Backfill snapshot rules (tanpa menyentuh updated_at)
UPDATE class_schedule_rules csr
SET class_schedule_rule_csst_snapshot =
      build_csst_snapshot(
        csr.class_schedule_rule_csst_id,
        csr.class_schedule_rule_csst_school_id
      )
WHERE csr.class_schedule_rule_deleted_at IS NULL
  AND (csr.class_schedule_rule_csst_snapshot IS NULL
       OR csr.class_schedule_rule_csst_snapshot = '{}'::jsonb);

COMMIT;




-- =========================================
-- TABLE: national_holidays (dikelola admin)
-- =========================================
CREATE TABLE IF NOT EXISTS national_holidays (
  national_holiday_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- opsional identitas
  national_holiday_slug VARCHAR(160),

  -- tanggal: satu hari (start=end) atau rentang
  national_holiday_start_date DATE NOT NULL,
  national_holiday_end_date   DATE NOT NULL CHECK (national_holiday_end_date >= national_holiday_start_date),

  national_holiday_title  VARCHAR(200) NOT NULL,
  national_holiday_reason TEXT,

  national_holiday_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  -- jika perlu fixed-date tahunan (mis. 08-17)
  national_holiday_is_recurring_yearly BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  national_holiday_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  national_holiday_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  national_holiday_deleted_at TIMESTAMPTZ
);

-- Indexes (national_holidays)
CREATE UNIQUE INDEX IF NOT EXISTS uq_national_holidays_slug_alive
  ON national_holidays (LOWER(national_holiday_slug))
  WHERE national_holiday_deleted_at IS NULL
    AND national_holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_national_holidays_date_range_alive
  ON national_holidays (national_holiday_start_date, national_holiday_end_date)
  WHERE national_holiday_deleted_at IS NULL
    AND national_holiday_is_active = TRUE;

CREATE INDEX IF NOT EXISTS gin_national_holidays_slug_trgm_alive
  ON national_holidays USING GIN (LOWER(national_holiday_slug) gin_trgm_ops)
  WHERE national_holiday_deleted_at IS NULL
    AND national_holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_national_holidays_created_at
  ON national_holidays USING BRIN (national_holiday_created_at);



-- =========================================
-- TABLE: school_holidays (libur custom per school/sekolah)
-- =========================================
CREATE TABLE IF NOT EXISTS school_holidays (
  school_holiday_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  school_holiday_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- opsional identitas
  school_holiday_slug VARCHAR(160),

  -- tanggal: satu hari (start=end) atau rentang
  school_holiday_start_date DATE NOT NULL,
  school_holiday_end_date   DATE NOT NULL CHECK (school_holiday_end_date >= school_holiday_start_date),

  school_holiday_title  VARCHAR(200) NOT NULL,
  school_holiday_reason TEXT,

  school_holiday_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- biasanya libur sekolah tidak berulang tahunan; tetap disediakan jika perlu
  school_holiday_is_recurring_yearly BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  school_holiday_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_holiday_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_holiday_deleted_at TIMESTAMPTZ
);

-- Indexes (school_holidays)
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_holidays_slug_per_tenant_alive
  ON school_holidays (school_holiday_school_id, LOWER(school_holiday_slug))
  WHERE school_holiday_deleted_at IS NULL
    AND school_holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_school_holidays_tenant_alive
  ON school_holidays (school_holiday_school_id)
  WHERE school_holiday_deleted_at IS NULL
    AND school_holiday_is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_school_holidays_date_range_alive
  ON school_holidays (school_holiday_start_date, school_holiday_end_date)
  WHERE school_holiday_deleted_at IS NULL
    AND school_holiday_is_active = TRUE;

CREATE INDEX IF NOT EXISTS gin_school_holidays_slug_trgm_alive
  ON school_holidays USING GIN (LOWER(school_holiday_slug) gin_trgm_ops)
  WHERE school_holiday_deleted_at IS NULL
    AND school_holiday_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_school_holidays_created_at
  ON school_holidays USING BRIN (school_holiday_created_at);