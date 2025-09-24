BEGIN;

-- ================================
-- 4) USER_CLASS_ATTENDANCE_EVENTS_URLS
-- ================================
DROP INDEX IF EXISTS idx_ucae_urls_sender_teacher;
DROP INDEX IF EXISTS idx_ucae_urls_sender_user;
DROP INDEX IF EXISTS idx_ucae_urls_masjid_sender;
DROP INDEX IF EXISTS idx_ucae_urls_primary;
DROP INDEX IF EXISTS idx_ucae_urls_attendance_kind;

DROP TABLE IF EXISTS user_class_attendance_events_urls;

-- ================================
-- 3) USER_CLASS_ATTENDANCE_EVENTS
-- ================================
DROP INDEX IF EXISTS idx_user_class_attendance_events_masjid_user;
DROP INDEX IF EXISTS idx_user_class_attendance_events_checkedin;
DROP INDEX IF EXISTS idx_user_class_attendance_events_masjid_rsvp;
DROP INDEX IF EXISTS idx_user_class_attendance_events_event;
DROP INDEX IF EXISTS ux_user_class_attendance_events_unique_identity;

DROP TABLE IF EXISTS user_class_attendance_events;

-- ================================
-- 2) CLASS_ATTENDANCE_EVENT_URLS
-- ================================
DROP INDEX IF EXISTS gin_class_attendance_event_url_label_trgm_live;
DROP INDEX IF EXISTS idx_class_attendance_event_urls_purge_due;
DROP INDEX IF EXISTS ux_class_attendance_event_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS idx_class_attendance_event_urls_masjid_live;
DROP INDEX IF EXISTS idx_class_attendance_event_urls_owner_live;

DROP TABLE IF EXISTS class_attendance_event_urls;

-- ================================
-- 1) CLASS_ATTENDANCE_EVENTS
-- ================================
DROP INDEX IF EXISTS idx_class_attendance_events_masjid;
DROP INDEX IF EXISTS idx_class_attendance_events_event;

DROP TABLE IF EXISTS class_attendance_events;

COMMIT;
