BEGIN;

-- =========================================================
-- CLASS_ROOMS — explicit timestamps + soft delete
-- =========================================================
CREATE TABLE IF NOT EXISTS class_rooms (
  class_room_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_rooms_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas ruang
  class_rooms_name      TEXT NOT NULL,
  class_rooms_code      TEXT,
  class_rooms_location  TEXT,
  class_rooms_floor     INT,
  class_rooms_capacity  INT CHECK (class_rooms_capacity >= 0),

  -- karakteristik
  class_rooms_is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
  class_rooms_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- daftar fasilitas (opsional)
  class_rooms_features  JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps eksplisit
  class_rooms_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_rooms_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_rooms_deleted_at TIMESTAMPTZ
);

-- Jika tabel sudah ada: tambahkan kolom timestamps eksplisit (aman berulang)
ALTER TABLE class_rooms
  ADD COLUMN IF NOT EXISTS class_rooms_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS class_rooms_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS class_rooms_deleted_at TIMESTAMPTZ;

-- Backfill dari kolom generic jika ada
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_rooms' AND column_name='created_at') THEN
    UPDATE class_rooms
    SET class_rooms_created_at = COALESCE(class_rooms_created_at, created_at)
    WHERE class_rooms_created_at IS NULL;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_rooms' AND column_name='updated_at') THEN
    UPDATE class_rooms
    SET class_rooms_updated_at = COALESCE(class_rooms_updated_at, updated_at)
    WHERE class_rooms_updated_at IS NULL;
  END IF;
END$$;

-- Uniques per tenant (case-insensitive) → partial pada baris alive
DROP INDEX IF EXISTS uq_class_rooms_tenant_name_ci;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_name_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_name))
  WHERE class_rooms_deleted_at IS NULL;

DROP INDEX IF EXISTS uq_class_rooms_tenant_code_ci;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_code_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_code))
  WHERE class_rooms_deleted_at IS NULL
    AND class_rooms_code IS NOT NULL
    AND length(trim(class_rooms_code)) > 0;

-- Indeks bantu
DROP INDEX IF EXISTS idx_class_rooms_tenant_active;
CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_active
  ON class_rooms (class_rooms_masjid_id, class_rooms_is_active)
  WHERE class_rooms_deleted_at IS NULL;

-- (opsional) filter alive untuk GIN/TRGM agar lebih ringkas
DROP INDEX IF EXISTS idx_class_rooms_features_gin;
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin
  ON class_rooms USING GIN (class_rooms_features jsonb_path_ops)
  WHERE class_rooms_deleted_at IS NULL;

DROP INDEX IF EXISTS idx_class_rooms_name_trgm;
CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm
  ON class_rooms USING GIN (class_rooms_name gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

DROP INDEX IF EXISTS idx_class_rooms_location_trgm;
CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm
  ON class_rooms USING GIN (class_rooms_location gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

-- =========================================================
-- CLASS_SCHEDULES — explicit timestamps + soft delete
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_schedules_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- induk jadwal → section
  class_schedules_section_id  UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- opsional override
  class_schedules_subject_id  UUID REFERENCES subjects(subject_id) ON DELETE RESTRICT,
  class_schedules_semester_id UUID REFERENCES academic_terms(academic_terms_id) ON DELETE CASCADE,
  class_schedules_teacher_id  UUID REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- room (nullable)
  class_schedules_room_id UUID REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- pola berulang
  class_schedules_day_of_week INT  NOT NULL CHECK (class_schedules_day_of_week BETWEEN 1 AND 7),
  class_schedules_start_time  TIME NOT NULL,
  class_schedules_end_time    TIME NOT NULL CHECK (class_schedules_end_time > class_schedules_start_time),

  -- batas berlaku
  class_schedules_start_date  DATE NOT NULL,
  class_schedules_end_date    DATE NOT NULL CHECK (class_schedules_end_date >= class_schedules_start_date),

  -- status & metadata
  class_schedules_status      session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedules_is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  class_schedules_room_label  TEXT,

  -- generated: rentang menit [start, end)
  class_schedules_time_range int4range
    GENERATED ALWAYS AS (
      int4range(
        (EXTRACT(HOUR FROM class_schedules_start_time)*60
         + EXTRACT(MINUTE FROM class_schedules_start_time))::int,
        (EXTRACT(HOUR FROM class_schedules_end_time)*60
         + EXTRACT(MINUTE FROM class_schedules_end_time))::int,
        '[)'
      )
    ) STORED,

  -- timestamps eksplisit
  class_schedules_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_schedules_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_schedules_deleted_at TIMESTAMPTZ
);

-- Backfill dari kolom generic jika ada
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_schedules' AND column_name='created_at') THEN
    UPDATE class_schedules
    SET class_schedules_created_at = COALESCE(class_schedules_created_at, created_at)
    WHERE class_schedules_created_at IS NULL;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_schedules' AND column_name='updated_at') THEN
    UPDATE class_schedules
    SET class_schedules_updated_at = COALESCE(class_schedules_updated_at, updated_at)
    WHERE class_schedules_updated_at IS NULL;
  END IF;
END$$;

-- Pastikan FK (teacher → masjid_teachers, term → academic_terms)
DO $$
DECLARE fkname text;
BEGIN
  SELECT tc.constraint_name INTO fkname
  FROM information_schema.table_constraints tc
  JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name AND tc.table_name = kcu.table_name
  WHERE tc.table_name='class_schedules' AND tc.constraint_type='FOREIGN KEY'
    AND kcu.column_name='class_schedules_teacher_id'
  LIMIT 1;
  IF fkname IS NOT NULL THEN
    EXECUTE format('ALTER TABLE class_schedules DROP CONSTRAINT %I', fkname);
  END IF;
  ALTER TABLE class_schedules
    ADD CONSTRAINT fk_class_schedules_teacher
    FOREIGN KEY (class_schedules_teacher_id)
    REFERENCES masjid_teachers(masjid_teacher_id)
    ON DELETE SET NULL;

  SELECT tc.constraint_name INTO fkname
  FROM information_schema.table_constraints tc
  JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name AND tc.table_name = kcu.table_name
  WHERE tc.table_name='class_schedules' AND tc.constraint_type='FOREIGN KEY'
    AND kcu.column_name='class_schedules_semester_id'
  LIMIT 1;
  IF fkname IS NOT NULL THEN
    EXECUTE format('ALTER TABLE class_schedules DROP CONSTRAINT %I', fkname);
  END IF;
  ALTER TABLE class_schedules
    ADD CONSTRAINT fk_class_schedules_term
    FOREIGN KEY (class_schedules_semester_id)
    REFERENCES academic_terms(academic_terms_id)
    ON DELETE CASCADE;
END$$;

-- Indexing & Optimizations
CREATE INDEX IF NOT EXISTS idx_class_schedules_tenant_term_dow
  ON class_schedules (class_schedules_masjid_id, class_schedules_semester_id, class_schedules_day_of_week);

CREATE INDEX IF NOT EXISTS idx_class_schedules_section_dow_time
  ON class_schedules (class_schedules_section_id, class_schedules_day_of_week, class_schedules_start_time, class_schedules_end_time);

CREATE INDEX IF NOT EXISTS idx_class_schedules_teacher_dow_time
  ON class_schedules (class_schedules_teacher_id, class_schedules_day_of_week, class_schedules_start_time, class_schedules_end_time);

CREATE INDEX IF NOT EXISTS idx_class_schedules_room_dow
  ON class_schedules (class_schedules_room_id, class_schedules_day_of_week);

-- Recreate visible/active index agar hanya baris alive
DROP INDEX IF EXISTS idx_class_schedules_active;
CREATE INDEX IF NOT EXISTS idx_class_schedules_active
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

-- Exclusion Constraints (anti-bentrok) — tambahkan filter deleted_at IS NULL
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_room_overlap;
ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_room_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_room_id     WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_room_id IS NOT NULL AND class_schedules_deleted_at IS NULL);

ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_section_overlap;
ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_section_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_section_id  WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_deleted_at IS NULL);

-- (Opsional) Anti bentrok guru override
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_teacher_overlap;
ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_teacher_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_teacher_id  WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_teacher_id IS NOT NULL AND class_schedules_deleted_at IS NULL);

-- Validators (tenant/periode term & kecocokan guru)
CREATE OR REPLACE FUNCTION fn_validate_schedule_term()
RETURNS trigger AS $$
DECLARE
  v_term_masjid uuid;
  v_period      daterange;
BEGIN
  IF NEW.class_schedules_semester_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT academic_terms_masjid_id, academic_terms_period
    INTO v_term_masjid, v_period
  FROM academic_terms
  WHERE academic_terms_id = NEW.class_schedules_semester_id;

  IF v_term_masjid IS NULL THEN
    RAISE EXCEPTION 'Academic term tidak ditemukan (id=%)', NEW.class_schedules_semester_id;
  END IF;

  IF v_term_masjid <> NEW.class_schedules_masjid_id THEN
    RAISE EXCEPTION 'Masjid term (%) ≠ masjid schedule (%)', v_term_masjid, NEW.class_schedules_masjid_id;
  END IF;

  IF NEW.class_schedules_start_date::date < lower(v_period)
     OR NEW.class_schedules_start_date::date >= upper(v_period) THEN
    RAISE EXCEPTION 'Tanggal mulai jadwal (%) di luar periode term (%)',
      NEW.class_schedules_start_date::date, v_period;
  END IF;

  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_validate_schedule_term') THEN
    CREATE TRIGGER trg_validate_schedule_term
      BEFORE INSERT OR UPDATE OF
        class_schedules_semester_id,
        class_schedules_masjid_id,
        class_schedules_start_date
      ON class_schedules
      FOR EACH ROW
      EXECUTE FUNCTION fn_validate_schedule_term();
  END IF;
END$$;

CREATE OR REPLACE FUNCTION fn_validate_schedule_teacher_mtj()
RETURNS trigger AS $$
DECLARE v_teacher_masjid uuid;
BEGIN
  IF NEW.class_schedules_teacher_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT masjid_teacher_masjid_id
    INTO v_teacher_masjid
  FROM masjid_teachers
  WHERE masjid_teacher_id = NEW.class_schedules_teacher_id
    AND masjid_teacher_deleted_at IS NULL;

  IF v_teacher_masjid IS NULL THEN
    RAISE EXCEPTION 'Guru override tidak ditemukan / sudah dihapus';
  END IF;

  IF v_teacher_masjid <> NEW.class_schedules_masjid_id THEN
    RAISE EXCEPTION 'Masjid guru override (%) ≠ masjid schedule (%)',
      v_teacher_masjid, NEW.class_schedules_masjid_id;
  END IF;

  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_validate_schedule_teacher_mtj') THEN
    CREATE TRIGGER trg_validate_schedule_teacher_mtj
      BEFORE INSERT OR UPDATE OF
        class_schedules_teacher_id,
        class_schedules_masjid_id
      ON class_schedules
      FOR EACH ROW
      EXECUTE FUNCTION fn_validate_schedule_teacher_mtj();
  END IF;
END$$;

COMMIT;
