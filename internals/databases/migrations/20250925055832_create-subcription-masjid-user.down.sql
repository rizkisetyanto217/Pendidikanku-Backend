BEGIN;

-- ============================ --
-- DOWN: MASJID SERVICE SUBSCRIPTIONS
-- ============================ --

-- Drop indexes (aman bila sudah tidak ada)
DROP INDEX IF EXISTS uq_mss_masjid_current_alive;
DROP INDEX IF EXISTS idx_mss_masjid_alive;
DROP INDEX IF EXISTS idx_mss_plan_alive;
DROP INDEX IF EXISTS idx_mss_status_alive;
DROP INDEX IF EXISTS idx_mss_current_window;
DROP INDEX IF EXISTS gist_mss_period;
DROP INDEX IF EXISTS idx_mss_end_at_alive;
DROP INDEX IF EXISTS ux_mss_provider_ref_alive;
DROP INDEX IF EXISTS brin_mss_created_at;
DROP INDEX IF EXISTS brin_mss_updated_at;

-- Drop table
DROP TABLE IF EXISTS masjid_service_subscriptions CASCADE;


-- ============================ --
-- DOWN: USER SERVICE SUBSCRIPTIONS
-- ============================ --

-- Drop indexes (aman bila sudah tidak ada)
DROP INDEX IF EXISTS uq_uss_user_current_alive;
DROP INDEX IF EXISTS idx_uss_user_alive;
DROP INDEX IF EXISTS idx_uss_plan_alive;
DROP INDEX IF EXISTS idx_uss_status_alive;
DROP INDEX IF EXISTS idx_uss_current_window;
DROP INDEX IF EXISTS gist_uss_period;
DROP INDEX IF EXISTS idx_uss_end_at_alive;
DROP INDEX IF EXISTS ux_uss_provider_ref_alive;
DROP INDEX IF EXISTS brin_uss_created_at;
DROP INDEX IF EXISTS brin_uss_updated_at;

-- Drop table
DROP TABLE IF EXISTS user_service_subscriptions CASCADE;

COMMIT;
