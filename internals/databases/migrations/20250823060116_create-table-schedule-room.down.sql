BEGIN;

-- CLASS_ROOMS: kembalikan index ke versi non-partial
DROP TRIGGER IF EXISTS trg_touch_updated_at_class_rooms ON class_rooms;
DROP FUNCTION IF EXISTS fn_touch_updated_at_class_rooms();

DROP INDEX IF EXISTS idx_class_rooms_features_gin;
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin
  ON class_rooms USING GIN (class_rooms_features jsonb_path_ops);

DROP INDEX IF EXISTS idx_class_rooms_name_trgm;
CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm
  ON class_rooms USING GIN (class_rooms_name gin_trgm_ops);

DROP INDEX IF EXISTS idx_class_rooms_location_trgm;
CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm
  ON class_rooms USING GIN (class_rooms_location gin_trgm_ops);

DROP INDEX IF EXISTS idx_class_rooms_tenant_active;
CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_active
  ON class_rooms (class_rooms_masjid_id, class_rooms_is_active);

DROP INDEX IF EXISTS uq_class_rooms_tenant_name_ci;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_name_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_name));

DROP INDEX IF EXISTS uq_class_rooms_tenant_code_ci;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_code_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_code))
  WHERE class_rooms_code IS NOT NULL AND length(trim(class_rooms_code)) > 0;

-- CLASS_SCHEDULES: drop trigger & kembalikan index/constraints ke non-deleted filter
DROP TRIGGER IF EXISTS trg_touch_updated_at_class_schedules ON class_schedules;
DROP FUNCTION IF EXISTS fn_touch_updated_at_class_schedules();

-- validator
DROP TRIGGER IF EXISTS trg_validate_schedule_teacher_mtj ON class_schedules;
DROP FUNCTION IF EXISTS fn_validate_schedule_teacher_mtj();

DROP TRIGGER IF EXISTS trg_validate_schedule_term ON class_schedules;
DROP FUNCTION IF EXISTS fn_validate_schedule_term();

-- exclusion constraints: versi tanpa deleted_at
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_teacher_overlap;
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_section_overlap;
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS excl_sched_room_overlap;

ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_room_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_room_id     WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_room_id IS NOT NULL);

ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_section_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_section_id  WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active);

ALTER TABLE class_schedules ADD CONSTRAINT excl_sched_teacher_overlap
  EXCLUDE USING gist (
    class_schedules_masjid_id   WITH =,
    class_schedules_teacher_id  WITH =,
    class_schedules_day_of_week WITH =,
    class_schedules_time_range  WITH &&
  )
  WHERE (class_schedules_is_active AND class_schedules_teacher_id IS NOT NULL);

-- active index: non-deleted filter
DROP INDEX IF EXISTS idx_class_schedules_active;
CREATE INDEX IF NOT EXISTS idx_class_schedules_active
  ON class_schedules (class_schedules_is_active)
  WHERE class_schedules_is_active;

COMMIT;
