BEGIN;

-- =========================================================
-- 6) USER_CLASS_ATTENDANCE_EVENTS_URLS — DROP
-- =========================================================
-- Indexes
DROP INDEX IF EXISTS idx_ucae_urls_sender_teacher;
DROP INDEX IF EXISTS idx_ucae_urls_sender_user;
DROP INDEX IF EXISTS idx_ucae_urls_masjid_sender;
DROP INDEX IF EXISTS idx_ucae_urls_primary;
DROP INDEX IF EXISTS idx_ucae_urls_attendance_kind;

-- Table
-- NOTE: jika masih ada FK eksternal ke tabel ini, gunakan CASCADE:
-- DROP TABLE IF EXISTS user_class_attendance_events_urls CASCADE;
DROP TABLE IF EXISTS user_class_attendance_events_urls;


-- =========================================================
-- 4) USER_CLASS_ATTENDANCE_EVENTS — DROP
-- =========================================================
-- Indexes
DROP INDEX IF EXISTS idx_user_class_attendance_events_masjid_user;
DROP INDEX IF EXISTS idx_user_class_attendance_events_checkedin;
DROP INDEX IF EXISTS idx_user_class_attendance_events_masjid_rsvp;
DROP INDEX IF EXISTS idx_user_class_attendance_events_event;
DROP INDEX IF EXISTS uq_user_class_attendance_events_unique_identity;

-- (Opsional) compatibility view jika pernah dibuat
DROP VIEW IF EXISTS user_class_event_attendances;

-- Table
-- NOTE: jika masih ada FK eksternal ke tabel ini, gunakan CASCADE:
-- DROP TABLE IF EXISTS user_class_attendance_events CASCADE;
DROP TABLE IF EXISTS user_class_attendance_events;


-- =========================================================
-- 3) CLASS_ATTENDANCE_EVENTS — DROP
-- =========================================================
-- Indexes
DROP INDEX IF EXISTS idx_class_attendance_events_masjid;
DROP INDEX IF EXISTS idx_class_attendance_events_event;

-- Table
-- NOTE: jika masih ada FK eksternal ke tabel ini, gunakan CASCADE:
-- DROP TABLE IF EXISTS class_attendance_events CASCADE;
DROP TABLE IF EXISTS class_attendance_events;

COMMIT;
