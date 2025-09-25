-- +migrate Down
BEGIN;

-- 2) DROP parent table (catalog plans)
DROP TABLE IF EXISTS masjid_service_plans CASCADE;

-- 3) DROP enum type
DROP TYPE IF EXISTS masjid_subscription_status_enum;

-- (Opsional) DROP extensions
-- ⚠️ Hanya lakukan jika Anda yakin tidak dipakai modul lain.
-- DROP EXTENSION IF EXISTS btree_gist;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
