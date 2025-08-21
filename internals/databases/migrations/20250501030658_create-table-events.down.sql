
-- =========================
-- ========= DOWN ==========
-- =========================
BEGIN;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_user_event_regs_touch ON user_event_registrations;
DROP TRIGGER IF EXISTS trg_event_sessions_touch ON event_sessions;
DROP TRIGGER IF EXISTS trg_events_touch ON events;

-- Drop trigger functions
DROP FUNCTION IF EXISTS fn_touch_user_event_regs_updated_at();
DROP FUNCTION IF EXISTS fn_touch_event_sessions_updated_at();
DROP FUNCTION IF EXISTS fn_touch_events_updated_at();

-- Drop indexes (aman jika belum ada)
-- user_event_registrations
DROP INDEX IF EXISTS idx_user_event_regs_registered_only;
DROP INDEX IF EXISTS idx_user_event_regs_session_status;
DROP INDEX IF EXISTS idx_user_event_registrations_masjid_id;
DROP INDEX IF EXISTS idx_user_event_registrations_user_id;
DROP INDEX IF EXISTS idx_user_event_registrations_event_session_id;

-- event_sessions
DROP INDEX IF EXISTS idx_event_sessions_slug_trgm;
DROP INDEX IF EXISTS idx_event_sessions_title_trgm;
DROP INDEX IF EXISTS idx_event_sessions_tsv_gin;
DROP INDEX IF EXISTS idx_event_sessions_public_start;
DROP INDEX IF EXISTS idx_event_sessions_masjid_start;
DROP INDEX IF EXISTS idx_event_sessions_event_start;
DROP INDEX IF EXISTS idx_event_sessions_start_time;
DROP INDEX IF EXISTS idx_event_sessions_event_id;
DROP INDEX IF EXISTS ux_event_sessions_slug_ci;

-- events
DROP INDEX IF EXISTS idx_events_slug_trgm;
DROP INDEX IF EXISTS idx_events_title_trgm;
DROP INDEX IF EXISTS idx_events_tsv_gin;
DROP INDEX IF EXISTS idx_events_masjid_recent;
DROP INDEX IF EXISTS idx_events_masjid_id;
DROP INDEX IF EXISTS ux_events_slug_per_masjid_lower;
DROP INDEX IF EXISTS ux_events_slug_per_masjid_ci;

-- Drop generated columns
ALTER TABLE IF EXISTS event_sessions DROP COLUMN IF EXISTS event_session_search_tsv;
ALTER TABLE IF EXISTS events DROP COLUMN IF EXISTS event_search_tsv;

-- Drop tables (anak → induk)
DROP TABLE IF EXISTS user_event_registrations;
DROP TABLE IF EXISTS event_sessions;
DROP TABLE IF EXISTS events;

COMMIT;