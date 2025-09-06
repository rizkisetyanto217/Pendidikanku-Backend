BEGIN;

-- =========================
-- Drop exclusion constraints
-- =========================
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_teacher_overlap;
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_section_overlap;
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_room_overlap;

-- =========================
-- Drop indexes (opsional; tabel drop juga akan menghapus index)
-- =========================
DROP INDEX IF EXISTS gist_sched_dow_timerange;
DROP INDEX IF EXISTS idx_sched_date_bounds;

DROP INDEX IF EXISTS idx_sched_masjid_subject_alive;
DROP INDEX IF EXISTS idx_sched_masjid_teacher_alive;
DROP INDEX IF EXISTS idx_sched_masjid_section_alive;

DROP INDEX IF EXISTS idx_class_schedules_csst;
DROP INDEX IF EXISTS idx_class_schedules_teacher;
DROP INDEX IF EXISTS idx_class_schedules_teacher_dow;
DROP INDEX IF EXISTS idx_class_schedules_active;
DROP INDEX IF EXISTS idx_class_schedules_class_subject;
DROP INDEX IF EXISTS idx_class_schedules_room_dow;
DROP INDEX IF EXISTS idx_class_schedules_section_dow_time;
DROP INDEX IF EXISTS idx_class_schedules_tenant_dow;

-- =========================
-- Drop table
-- =========================
DROP TABLE IF EXISTS class_schedules CASCADE;

-- =========================
-- (Opsional aman) Drop enum jika tidak dipakai di tempat lain
-- =========================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum')
     AND NOT EXISTS (
       SELECT 1
       FROM pg_attribute a
       JOIN pg_class c ON c.oid = a.attrelid
       JOIN pg_type t  ON t.oid = a.atttypid
       WHERE a.attnum > 0
         AND NOT a.attisdropped
         AND t.typname = 'session_status_enum'
     )
  THEN
    DROP TYPE session_status_enum;
  END IF;
END$$;

COMMIT;
