BEGIN;

-- 1) Matikan trigger validasi yang versi "with teacher"
DROP TRIGGER IF EXISTS trg_class_schedules_validate_links ON class_schedules;

-- 2) Kembalikan fungsi validasi ke versi awal (tanpa teacher)
CREATE OR REPLACE FUNCTION fn_class_schedules_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_sec_masjid UUID; v_sec_class UUID;
  v_cs_masjid  UUID; v_cs_class  UUID;
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

  -- Tenant harus sama
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

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- 3) Hidupkan lagi trigger validasi versi awal (tanpa kolom teacher)
CREATE CONSTRAINT TRIGGER trg_class_schedules_validate_links
  AFTER INSERT OR UPDATE OF
    class_schedules_masjid_id,
    class_schedules_section_id,
    class_schedules_class_subject_id
  ON class_schedules
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_class_schedules_validate_links();

-- 4) Hapus exclusion constraint per-guru (anti-bentrok guru)
ALTER TABLE class_schedules
  DROP CONSTRAINT IF EXISTS excl_sched_teacher_overlap;

-- 5) Hapus index terkait teacher
DROP INDEX IF EXISTS idx_class_schedules_teacher_dow;
DROP INDEX IF EXISTS idx_class_schedules_teacher;

-- 6) Hapus FK ke masjid_teachers
ALTER TABLE class_schedules
  DROP CONSTRAINT IF EXISTS fk_class_schedules_teacher;

-- 7) Hapus kolom teacher
ALTER TABLE class_schedules
  DROP COLUMN IF EXISTS class_schedules_teacher_id;

COMMIT;
