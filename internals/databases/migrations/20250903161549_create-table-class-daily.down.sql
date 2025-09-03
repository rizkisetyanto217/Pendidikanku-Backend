BEGIN;

-- 1) Hapus trigger dulu (kalau tabelnya ada)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'class_daily') THEN
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_daily_validate_links') THEN
      DROP TRIGGER trg_class_daily_validate_links ON class_daily;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_daily_touch_updated_at') THEN
      DROP TRIGGER trg_class_daily_touch_updated_at ON class_daily;
    END IF;
  END IF;
END$$;

-- 2) Hapus exclusion constraints (opsional; DROP TABLE juga akan menghapus)
ALTER TABLE IF EXISTS class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_room_overlap;
ALTER TABLE IF EXISTS class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_section_overlap;
ALTER TABLE IF EXISTS class_daily DROP CONSTRAINT IF EXISTS excl_class_daily_teacher_overlap;

-- 3) Hapus index (opsional; DROP TABLE juga akan menghapus)
DROP INDEX IF EXISTS uq_class_daily_masjid_section_date_timerange;
DROP INDEX IF EXISTS uq_class_daily_attendance_unique_alive;
DROP INDEX IF EXISTS idx_class_daily_masjid_date;
DROP INDEX IF EXISTS idx_class_daily_section_date;
DROP INDEX IF EXISTS idx_class_daily_teacher_date;
DROP INDEX IF EXISTS idx_class_daily_room_date;
DROP INDEX IF EXISTS idx_class_daily_active;

-- 4) Hapus tabel
DROP TABLE IF EXISTS class_daily;

-- 5) Hapus function trigger yang dibuat di UP
DROP FUNCTION IF EXISTS fn_class_daily_validate_links();
DROP FUNCTION IF EXISTS fn_touch_class_daily_updated_at();

-- (Opsional) Jangan otomatis drop extension karena bisa dipakai object lain:
-- DROP EXTENSION IF EXISTS btree_gist;

COMMIT;
