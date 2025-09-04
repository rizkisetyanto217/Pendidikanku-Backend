BEGIN;

-- =========================
-- Prasyarat khusus class_schedules
-- =========================
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- untuk exclusion constraint dengan GiST

-- Enum status jadwal (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
END$$;

-- =========================================================
-- CLASS_SCHEDULES (pakai class_subjects; + relasi ke masjid_teachers)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_schedules_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- induk jadwal → section
  class_schedules_section_id  UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- mapel konteks (kelas+mapel[+term]) → class_subjects
  class_schedules_class_subject_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON DELETE RESTRICT,

  -- room (nullable)
  class_schedules_room_id UUID
    REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- ✨ guru (nullable) → masjid_teachers
  class_schedules_teacher_id UUID,

  -- pola berulang
  class_schedules_day_of_week INT  NOT NULL CHECK (class_schedules_day_of_week BETWEEN 1 AND 7),
  class_schedules_start_time  TIME NOT NULL,
  class_schedules_end_time    TIME NOT NULL CHECK (class_schedules_end_time > class_schedules_start_time),

  -- batas berlaku
  class_schedules_start_date  DATE NOT NULL,
  class_schedules_end_date    DATE NOT NULL CHECK (class_schedules_end_date >= class_schedules_start_date),

  -- status & metadata
  class_schedules_status    session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedules_is_active BOOLEAN NOT NULL DEFAULT TRUE,

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

  -- timestamps standar GORM (dikelola aplikasi)
  class_schedules_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedules_deleted_at TIMESTAMPTZ
);

-- Pastikan kolom teacher ada (idempotent kalau tabel lama)
ALTER TABLE class_schedules
  ADD COLUMN IF NOT EXISTS class_schedules_teacher_id UUID;

-- FK ke masjid_teachers (idempotent): drop kalau ada salah, lalu add
ALTER TABLE class_schedules
  DROP CONSTRAINT IF EXISTS fk_class_schedules_teacher;
ALTER TABLE class_schedules
  ADD CONSTRAINT fk_class_schedules_teacher
  FOREIGN KEY (class_schedules_teacher_id)
  REFERENCES masjid_teachers(masjid_teacher_id)
  ON DELETE SET NULL;

-- =========================
-- Indexing
-- =========================
CREATE INDEX IF NOT EXISTS idx_class_schedules_tenant_dow
  ON class_schedules (class_schedules_masjid_id, class_schedules_day_of_week);

CREATE INDEX IF NOT EXISTS idx_class_schedules_section_dow_time
  ON class_schedules (class_schedules_section_id, class_schedules_day_of_week, class_schedules_start_time, class_schedules_end_time);

CREATE INDEX IF NOT EXISTS idx_class_schedules_room_dow
  ON class_schedules (class_schedules_room_id, class_schedules_day_of_week);

CREATE INDEX IF NOT EXISTS idx_class_schedules_class_subject
  ON class_schedules (class_schedules_class_subject_id);

CREATE INDEX IF NOT EXISTS idx_class_schedules_active
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active AND class_schedules_deleted_at IS NULL;

-- ✨ index untuk teacher
CREATE INDEX IF NOT EXISTS idx_class_schedules_teacher_dow
  ON class_schedules (class_schedules_teacher_id, class_schedules_day_of_week);
CREATE INDEX IF NOT EXISTS idx_class_schedules_teacher
  ON class_schedules (class_schedules_teacher_id);

-- =========================
-- Exclusion Constraints (anti-bentrok)
-- =========================
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

-- ✨ anti-bentrok per-guru (opsional tapi disarankan)
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_teacher_overlap;
ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_teacher_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_teacher_id  WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_teacher_id IS NOT NULL AND class_schedules_deleted_at IS NULL);

-- =========================
-- Validator konsistensi (tenant & class & teacher)
-- =========================
CREATE OR REPLACE FUNCTION fn_class_schedules_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_sec_masjid UUID; v_sec_class UUID;
  v_cs_masjid  UUID; v_cs_class  UUID;
  v_t_masjid   UUID;
BEGIN
  -- SECTION
  SELECT class_sections_masjid_id, class_sections_class_id
    INTO v_sec_masjid, v_sec_class
  FROM class_sections
  WHERE class_sections_id = NEW.class_schedules_section_id
    AND class_sections_deleted_at IS NULL;

  IF v_sec_masjid IS NULL THEN
    RAISE EXCEPTION 'Section invalid/terhapus';
  END IF;

  -- CLASS_SUBJECTS
  SELECT class_subjects_masjid_id, class_subjects_class_id
    INTO v_cs_masjid, v_cs_class
  FROM class_subjects
  WHERE class_subjects_id = NEW.class_schedules_class_subject_id
    AND class_subjects_deleted_at IS NULL;

  IF v_cs_masjid IS NULL THEN
    RAISE EXCEPTION 'Class_subjects invalid/terhapus';
  END IF;

  -- Tenant harus sama (schedule vs section & class_subjects)
  IF NEW.class_schedules_masjid_id <> v_sec_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: schedule(%) vs section(%)',
      NEW.class_schedules_masjid_id, v_sec_masjid;
  END IF;
  IF NEW.class_schedules_masjid_id <> v_cs_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: schedule(%) vs class_subjects(%)',
      NEW.class_schedules_masjid_id, v_cs_masjid;
  END IF;

  -- Class harus sama (section vs class_subjects)
  IF v_sec_class <> v_cs_class THEN
    RAISE EXCEPTION 'Class mismatch: section.class_id(%) != class_subjects.class_id(%)',
      v_sec_class, v_cs_class;
  END IF;

  -- ✨ TEACHER (opsional): jika diisi, harus satu tenant & alive
  IF NEW.class_schedules_teacher_id IS NOT NULL THEN
    SELECT masjid_teacher_masjid_id
      INTO v_t_masjid
    FROM masjid_teachers
    WHERE masjid_teacher_id = NEW.class_schedules_teacher_id
      AND masjid_teacher_deleted_at IS NULL;

    IF v_t_masjid IS NULL THEN
      RAISE EXCEPTION 'Teacher invalid/terhapus';
    END IF;

    IF NEW.class_schedules_masjid_id <> v_t_masjid THEN
      RAISE EXCEPTION 'Masjid mismatch: schedule(%) vs teacher(%)',
        NEW.class_schedules_masjid_id, v_t_masjid;
    END IF;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Perbarui trigger (tambahkan kolom teacher di daftar event)
DROP TRIGGER IF EXISTS trg_class_schedules_validate_links ON class_schedules;
CREATE CONSTRAINT TRIGGER trg_class_schedules_validate_links
  AFTER INSERT OR UPDATE OF
    class_schedules_masjid_id,
    class_schedules_section_id,
    class_schedules_class_subject_id,
    class_schedules_teacher_id
  ON class_schedules
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_class_schedules_validate_links();

COMMIT;
