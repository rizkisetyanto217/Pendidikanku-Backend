-- +migrate Down
-- =========================================
-- DOWN Migration — Class Attendance Session Participants (student + teacher)
-- =========================================
BEGIN;

-- =========================================
-- C) CLASS_ATTENDANCE_SESSION_PARTICIPANT_URLS
-- =========================================
DROP TABLE IF EXISTS class_attendance_session_participant_urls CASCADE;

-- =========================================
-- B) CLASS_ATTENDANCE_SESSION_PARTICIPANTS
-- =========================================
DROP TABLE IF EXISTS class_attendance_session_participants CASCADE;

-- =========================================
-- A) CLASS_ATTENDANCE_SESSION_PARTICIPANT_TYPES
-- =========================================
DROP TABLE IF EXISTS class_attendance_session_participant_types CASCADE;

-- =========================================
-- ENUMS (drop jika ada)
-- NOTE:
--   Pastikan enum ini tidak dipakai table lain
--   sebelum menjalankan DOWN ini.
-- =========================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'participant_kind_enum') THEN
    DROP TYPE participant_kind_enum;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'teacher_role_enum') THEN
    DROP TYPE teacher_role_enum;
  END IF;

  -- IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_state_enum') THEN
  --   DROP TYPE attendance_state_enum;
  -- END IF;
END$$;

COMMIT;
