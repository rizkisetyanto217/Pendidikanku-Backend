-- =========================================
-- DOWN — balikkan perubahan .up.sql
-- =========================================

-- Indexes for class_attendance_session_urls
DROP INDEX IF EXISTS ix_casu_by_owner_live;
DROP INDEX IF EXISTS ix_casu_by_school_live;
DROP INDEX IF EXISTS uq_casu_primary_per_kind_alive;
DROP INDEX IF EXISTS ix_casu_purge_due;
DROP INDEX IF EXISTS gin_casu_label_trgm_live;

-- Table class_attendance_session_urls
DROP TABLE IF EXISTS class_attendance_session_urls;

-- Indexes for class_attendance_sessions
DROP INDEX IF EXISTS uq_cas_id_tenant;
DROP INDEX IF EXISTS uq_cas_slug_per_tenant_alive;
DROP INDEX IF EXISTS gin_cas_slug_trgm_alive;
DROP INDEX IF EXISTS uq_cas_school_schedule_date_alive;
DROP INDEX IF EXISTS idx_cas_school_date_alive;
DROP INDEX IF EXISTS idx_cas_schedule_date_alive;
DROP INDEX IF EXISTS idx_cas_teacher_date_alive;
DROP INDEX IF EXISTS idx_cas_canceled_date_alive;
DROP INDEX IF EXISTS idx_cas_override_date_alive;
DROP INDEX IF EXISTS idx_cas_override_event_alive;
DROP INDEX IF EXISTS idx_cas_rule_alive;
DROP INDEX IF EXISTS uq_cas_sched_start;
DROP INDEX IF EXISTS brin_cas_created_at;

-- Table class_attendance_sessions
DROP TABLE IF EXISTS class_attendance_sessions;

-- NOTE:
-- Tidak men-drop ENUM session_status_enum karena kemungkinan dipakai tabel lain.
-- Kalau ingin drop juga (dan yakin tidak dipakai di tempat lain), baru:
-- DO $$ BEGIN
--   IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
--     EXECUTE 'DROP TYPE session_status_enum';
--   END IF;
-- END$$;
