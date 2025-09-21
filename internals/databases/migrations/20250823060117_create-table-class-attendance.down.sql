BEGIN;

-- =========================================================
-- class_attendance_session_url
-- =========================================================
DROP INDEX IF EXISTS gin_casu_label_trgm_live;
DROP INDEX IF EXISTS ix_casu_purge_due;
DROP INDEX IF EXISTS uq_casu_primary_per_kind_alive;
DROP INDEX IF EXISTS ix_casu_by_masjid_live;
DROP INDEX IF EXISTS ix_casu_by_owner_live;

DROP TABLE IF EXISTS class_attendance_session_url;

-- =========================================================
-- class_attendance_sessions
-- =========================================================
DROP INDEX IF EXISTS idx_cas_rule_alive;
DROP INDEX IF EXISTS idx_cas_override_event_alive;
DROP INDEX IF EXISTS idx_cas_override_date_alive;
DROP INDEX IF EXISTS idx_cas_canceled_date_alive;
DROP INDEX IF EXISTS idx_cas_teacher_date_alive;
DROP INDEX IF EXISTS idx_cas_schedule_date_alive;
DROP INDEX IF EXISTS idx_cas_masjid_date_alive;
DROP INDEX IF EXISTS uq_cas_masjid_schedule_date_alive;
DROP INDEX IF EXISTS gin_cas_slug_trgm_alive;
DROP INDEX IF EXISTS uq_cas_slug_per_tenant_alive;

DROP TABLE IF EXISTS class_attendance_sessions;

COMMIT;