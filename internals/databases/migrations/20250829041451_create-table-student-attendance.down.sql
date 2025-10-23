-- +migrate Down
-- =========================================
-- DOWN Migration — drop Student Class Session Attendance objects
-- =========================================
BEGIN;

-- Drop in reverse dependency order (URLs -> Attendances -> Types)

-- C) STUDENT_CLASS_SESSION_ATTENDANCE_URLS
DROP TABLE IF EXISTS student_class_session_attendance_urls CASCADE;

-- B) STUDENT_CLASS_SESSION_ATTENDANCES
DROP TABLE IF EXISTS student_class_session_attendances CASCADE;

-- A) STUDENT_CLASS_SESSION_ATTENDANCE_TYPES
DROP TABLE IF EXISTS student_class_session_attendance_types CASCADE;

COMMIT;
