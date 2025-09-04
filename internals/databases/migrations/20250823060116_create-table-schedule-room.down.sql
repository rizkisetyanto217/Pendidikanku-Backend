BEGIN;

-- =========================
-- Bersihkan TRIGGER & FUNCTION validator
-- =========================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_schedules_validate_links') THEN
    DROP TRIGGER trg_class_schedules_validate_links ON class_schedules;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_class_schedules_validate_links();

-- =========================
-- Hapus index & constraint di CLASS_SCHEDULES
-- (opsional; akan ikut hilang saat DROP TABLE, tapi eksplisit aman)
-- =========================
DROP INDEX IF EXISTS idx_class_schedules_active;
DROP INDEX IF EXISTS idx_class_schedules_class_subject;
DROP INDEX IF EXISTS idx_class_schedules_room_dow;
DROP INDEX IF EXISTS idx_class_schedules_section_dow_time;
DROP INDEX IF EXISTS idx_class_schedules_tenant_dow;

ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_room_overlap;
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_section_overlap;

-- =========================
-- Hapus index di CLASS_ROOMS (opsional eksplisit)
-- =========================
DROP INDEX IF EXISTS uq_class_rooms_tenant_name_ci;
DROP INDEX IF EXISTS uq_class_rooms_tenant_code_ci;
DROP INDEX IF EXISTS idx_class_rooms_tenant_active;
DROP INDEX IF EXISTS idx_class_rooms_features_gin;
DROP INDEX IF EXISTS idx_class_rooms_name_trgm;
DROP INDEX IF EXISTS idx_class_rooms_location_trgm;

-- =========================
-- DROP TABLES
-- =========================
DROP TABLE IF EXISTS class_schedules;
DROP TABLE IF EXISTS class_rooms;

COMMIT;
