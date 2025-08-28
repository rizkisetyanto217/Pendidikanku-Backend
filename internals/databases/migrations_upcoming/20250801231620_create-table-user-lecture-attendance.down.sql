
-- =====================================================================
-- ==============================  DOWN  ================================
-- =====================================================================

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_attendance_updated_at;
DROP INDEX IF EXISTS idx_attendance_deleted_at;
DROP INDEX IF EXISTS idx_attendance_user_lecture_active;
DROP INDEX IF EXISTS idx_attendance_lecture_status_active;
DROP INDEX IF EXISTS idx_attendance_session_status_active;
DROP INDEX IF EXISTS idx_attendance_user_created_active;
DROP INDEX IF EXISTS uq_attendance_user_session_active;

-- Drop trigger
DROP TRIGGER IF EXISTS trg_attendance_touch ON user_lecture_sessions_attendance;

-- Drop table
DROP TABLE IF EXISTS user_lecture_sessions_attendance;

-- Keep helper function for reuse; uncomment to remove:
-- DROP FUNCTION IF EXISTS fn_touch_updated_at_generic;

COMMIT;