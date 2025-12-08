-- +migrate Down
-- =========================================
-- DOWN — Class Attendance Sessions (types, sessions, urls)
-- =========================================
BEGIN;

-- =========================================
-- 1) TABLE: class_attendance_session_urls
--    (child dari class_attendance_sessions)
-- =========================================
DROP TABLE IF EXISTS class_attendance_session_urls;

-- =========================================
-- 2) TABLE: class_attendance_sessions
--    (FK ke schedules, csst, rooms, teachers, types)
-- =========================================
DROP TABLE IF EXISTS class_attendance_sessions;

-- =========================================
-- 3) TABLE: class_attendance_session_types
--    (master per tenant)
-- =========================================
DROP TABLE IF EXISTS class_attendance_session_types;

-- =========================================
-- 4) ENUMS (jika memang hanya dipakai modul ini)
-- =========================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    DROP TYPE session_status_enum;
  END IF;
END$$;

COMMIT;
