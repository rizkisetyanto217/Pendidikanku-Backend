-- +migrate Down
BEGIN;

DROP INDEX IF EXISTS uq_csr_unique_slot_per_schedule;
DROP INDEX IF EXISTS idx_csr_by_masjid;
DROP INDEX IF EXISTS idx_csr_by_schedule_dow;
DROP TABLE IF EXISTS class_schedule_rules;

DROP INDEX IF EXISTS idx_sched_date_bounds_alive;
DROP INDEX IF EXISTS idx_sched_active_alive;
DROP INDEX IF EXISTS idx_sched_tenant_alive;
DROP TABLE IF EXISTS class_schedules;

-- (Enum dibiarkan agar aman untuk objek lain)
COMMIT;