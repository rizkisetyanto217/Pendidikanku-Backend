BEGIN;

-- Pastikan operator GISt untuk UUID/DATE tersedia (untuk exclusion constraints)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- CLASS_DAILY — occurrence harian (turunan dari schedule)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_daily (
  class_daily_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_daily_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- tanggal occurrence
  class_daily_date DATE NOT NULL,

  -- linkage sumber (opsional)
  class_daily_schedule_id   UUID REFERENCES class_schedules(class_schedule_id) ON DELETE SET NULL,
  class_daily_attendance_id UUID REFERENCES class_attendance_sessions(class_attendance_sessions_id) ON DELETE SET NULL,

  -- section wajib (persist supaya stabil walau sumber berubah)
  class_daily_section_id UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- snapshot/override opsional
  class_daily_subject_id         UUID REFERENCES subjects(subject_id) ON DELETE RESTRICT,
  class_daily_academic_terms_id  UUID REFERENCES academic_terms(academic_terms_id) ON DELETE CASCADE,
  class_daily_teacher_id         UUID REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,
  class_daily_room_id            UUID REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- waktu pada tanggal tsb
  class_daily_start_time TIME NOT NULL,
  class_daily_end_time   TIME NOT NULL CHECK (class_daily_end_time > class_daily_start_time),

  -- status & metadata
  class_daily_status     session_status_enum NOT NULL DEFAULT 'scheduled',
  class_daily_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  class_daily_room_label TEXT,

  -- generated helpers
  class_daily_time_range int4range
    GENERATED ALWAYS AS (
      int4range(
        (EXTRACT(HOUR FROM class_daily_start_time)*60
         + EXTRACT(MINUTE FROM class_daily_start_time))::int,
        (EXTRACT(HOUR FROM class_daily_end_time)*60
         + EXTRACT(MINUTE FROM class_daily_end_time))::int,
        '[)'
      )
    ) STORED,

  -- ISO DOW 1..7 (Senin=1 … Minggu=7)
  class_daily_day_of_week INT
    GENERATED ALWAYS AS (EXTRACT(ISODOW FROM class_daily_date)) STORED,

  -- timestamps eksplisit
  class_daily_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_daily_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_daily_deleted_at TIMESTAMPTZ
);

-- =========================
-- Indexing & Uniques (alive)
-- =========================

-- Satu occurrence unik per (masjid, section, date, time window)
DROP INDEX IF EXISTS uq_class_daily_masjid_section_date_timerange;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_daily_masjid_section_date_timerange
  ON class_daily (
    class_daily_masjid_id,
    class_daily_section_id,
    class_daily_date,
    class_daily_start_time,
    class_daily_end_time
  )
  WHERE class_daily_deleted_at IS NULL;

-- Satu attendance_id hanya boleh terhubung ke satu occurrence (alive)
DROP INDEX IF EXISTS uq_class_daily_attendance_unique_alive;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_daily_attendance_unique_alive
  ON class_daily (class_daily_attendance_id)
  WHERE class_daily_attendance_id IS NOT NULL
    AND class_daily_deleted_at IS NULL;

-- Indeks bantu query
CREATE INDEX IF NOT EXISTS idx_class_daily_masjid_date
  ON class_daily (class_daily_masjid_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_daily_section_date
  ON class_daily (class_daily_section_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_daily_teacher_date
  ON class_daily (class_daily_teacher_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_daily_room_date
  ON class_daily (class_daily_room_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

DROP INDEX IF EXISTS idx_class_daily_active;
CREATE INDEX IF NOT EXISTS idx_class_daily_active
  ON class_daily (class_daily_is_active)
  WHERE class_daily_is_active AND class_daily_deleted_at IS NULL;

-- =========================================================
-- Exclusion Constraints (anti-bentrok per-hari)
-- =========================================================
ALTER TABLE class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_room_overlap;
ALTER TABLE class_daily ADD CONSTRAINT excl_class_daily_room_overlap
  EXCLUDE USING gist (
    class_daily_masjid_id  WITH =,
    class_daily_room_id    WITH =,
    class_daily_date       WITH =,
    class_daily_time_range WITH &&
  )
  WHERE (class_daily_is_active AND class_daily_room_id IS NOT NULL AND class_daily_deleted_at IS NULL);

ALTER TABLE class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_section_overlap;
ALTER TABLE class_daily ADD CONSTRAINT excl_class_daily_section_overlap
  EXCLUDE USING gist (
    class_daily_masjid_id  WITH =,
    class_daily_section_id WITH =,
    class_daily_date       WITH =,
    class_daily_time_range WITH &&
  )
  WHERE (class_daily_is_active AND class_daily_deleted_at IS NULL);

ALTER TABLE class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_teacher_overlap;
ALTER TABLE class_daily ADD CONSTRAINT excl_class_daily_teacher_overlap
  EXCLUDE USING gist (
    class_daily_masjid_id  WITH =,
    class_daily_teacher_id WITH =,
    class_daily_date       WITH =,
    class_daily_time_range WITH &&
  )
  WHERE (class_daily_is_active AND class_daily_teacher_id IS NOT NULL AND class_daily_deleted_at IS NULL);

-- =========================================================
-- Validators (konsistensi relasi/tenant & periode term)
-- =========================================================

CREATE OR REPLACE FUNCTION fn_class_daily_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_s_masjid UUID; v_s_section UUID; v_s_start DATE; v_s_end DATE; v_s_dow INT;
  v_a_masjid UUID; v_a_section UUID; v_a_date DATE;
  v_t_masjid UUID;
  v_term_masjid UUID;
  v_subj_masjid UUID;
BEGIN
  -- Jika terhubung ke schedule
  IF NEW.class_daily_schedule_id IS NOT NULL THEN
    SELECT class_schedules_masjid_id,
           class_schedules_section_id,
           class_schedules_start_date,
           class_schedules_end_date,
           class_schedules_day_of_week
      INTO v_s_masjid, v_s_section, v_s_start, v_s_end, v_s_dow
    FROM class_schedules
    WHERE class_schedule_id = NEW.class_daily_schedule_id
      AND class_schedules_deleted_at IS NULL;

    IF v_s_masjid IS NULL THEN
      RAISE EXCEPTION 'Schedule source tidak ditemukan/terhapus';
    END IF;
    IF v_s_masjid <> NEW.class_daily_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: schedule(%) vs daily(%)', v_s_masjid, NEW.class_daily_masjid_id;
    END IF;
    IF v_s_section <> NEW.class_daily_section_id THEN
      RAISE EXCEPTION 'Section mismatch: schedule.section(%) vs daily.section(%)', v_s_section, NEW.class_daily_section_id;
    END IF;
    IF NOT (NEW.class_daily_date BETWEEN v_s_start AND v_s_end) THEN
      RAISE EXCEPTION 'Tanggal daily (%) di luar rentang jadwal (%..%)', NEW.class_daily_date, v_s_start, v_s_end;
    END IF;
    -- DOW cocok
    IF EXTRACT(ISODOW FROM NEW.class_daily_date)::INT <> v_s_dow THEN
      RAISE EXCEPTION 'Hari daily (isodow=%) tidak sesuai jadwal (=%)',
        EXTRACT(ISODOW FROM NEW.class_daily_date)::INT, v_s_dow;
    END IF;
  END IF;

  -- Jika terhubung ke attendance session
  IF NEW.class_daily_attendance_id IS NOT NULL THEN
    SELECT class_attendance_sessions_masjid_id,
           class_attendance_sessions_section_id,
           class_attendance_sessions_date
      INTO v_a_masjid, v_a_section, v_a_date
    FROM class_attendance_sessions
    WHERE class_attendance_sessions_id = NEW.class_daily_attendance_id
      AND class_attendance_sessions_deleted_at IS NULL;

    IF v_a_masjid IS NULL THEN
      RAISE EXCEPTION 'Attendance session tidak ditemukan/terhapus';
    END IF;
    IF v_a_masjid <> NEW.class_daily_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: attendance(%) vs daily(%)', v_a_masjid, NEW.class_daily_masjid_id;
    END IF;
    IF v_a_section <> NEW.class_daily_section_id THEN
      RAISE EXCEPTION 'Section mismatch: attendance.section(%) vs daily.section(%)', v_a_section, NEW.class_daily_section_id;
    END IF;
    IF v_a_date <> NEW.class_daily_date THEN
      RAISE EXCEPTION 'Tanggal mismatch: attendance.date(%) vs daily.date(%)', v_a_date, NEW.class_daily_date;
    END IF;
  END IF;

  -- Teacher (opsional) harus se-masjid
  IF NEW.class_daily_teacher_id IS NOT NULL THEN
    SELECT masjid_teacher_masjid_id
      INTO v_t_masjid
    FROM masjid_teachers
    WHERE masjid_teacher_id = NEW.class_daily_teacher_id
      AND masjid_teacher_deleted_at IS NULL;
    IF v_t_masjid IS NULL THEN
      RAISE EXCEPTION 'Guru tidak ditemukan/terhapus';
    END IF;
    IF v_t_masjid <> NEW.class_daily_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: teacher(%) vs daily(%)', v_t_masjid, NEW.class_daily_masjid_id;
    END IF;
  END IF;

  -- Term (opsional) harus se-masjid & tanggal masuk periode
  IF NEW.class_daily_academic_terms_id IS NOT NULL THEN
    SELECT academic_terms_masjid_id
      INTO v_term_masjid
    FROM academic_terms
    WHERE academic_terms_id = NEW.class_daily_academic_terms_id;
    IF v_term_masjid IS NULL THEN
      RAISE EXCEPTION 'Academic term tidak ditemukan';
    END IF;
    IF v_term_masjid <> NEW.class_daily_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: term(%) vs daily(%)', v_term_masjid, NEW.class_daily_masjid_id;
    END IF;
    PERFORM 1
      FROM academic_terms
      WHERE academic_terms_id = NEW.class_daily_academic_terms_id
        AND NEW.class_daily_date >= lower(academic_terms_period)
        AND NEW.class_daily_date <  upper(academic_terms_period);
    IF NOT FOUND THEN
      RAISE EXCEPTION 'Tanggal daily (%) di luar periode term', NEW.class_daily_date;
    END IF;
  END IF;

  -- Subject (opsional) harus se-masjid
  IF NEW.class_daily_subject_id IS NOT NULL THEN
    SELECT subjects_masjid_id
      INTO v_subj_masjid
    FROM subjects
    WHERE subject_id = NEW.class_daily_subject_id
      AND subjects_deleted_at IS NULL;
    IF v_subj_masjid IS NULL THEN
      RAISE EXCEPTION 'Subject tidak ditemukan/terhapus';
    END IF;
    IF v_subj_masjid <> NEW.class_daily_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: subject(%) vs daily(%)', v_subj_masjid, NEW.class_daily_masjid_id;
    END IF;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_class_daily_validate_links') THEN
    DROP TRIGGER trg_class_daily_validate_links ON class_daily;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_class_daily_validate_links
    AFTER INSERT OR UPDATE OF
      class_daily_masjid_id,
      class_daily_section_id,
      class_daily_date,
      class_daily_schedule_id,
      class_daily_attendance_id,
      class_daily_teacher_id,
      class_daily_academic_terms_id,
      class_daily_subject_id
    ON class_daily
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_class_daily_validate_links();
END$$;

-- Touch updated_at
CREATE OR REPLACE FUNCTION fn_touch_class_daily_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_daily_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_class_daily_touch_updated_at') THEN
    DROP TRIGGER trg_class_daily_touch_updated_at ON class_daily;
  END IF;

  CREATE TRIGGER trg_class_daily_touch_updated_at
    BEFORE UPDATE ON class_daily
    FOR EACH ROW
    EXECUTE FUNCTION fn_touch_class_daily_updated_at();
END$$;

COMMIT;
