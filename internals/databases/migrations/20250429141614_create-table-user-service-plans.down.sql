-- +migrate Down
BEGIN;

-- 1) DROP child table (riwayat langganan user)
DROP TABLE IF EXISTS user_service_subscriptions CASCADE;

-- 2) DROP parent table (katalog paket user)
DROP TABLE IF EXISTS user_service_plans CASCADE;

-- 3) DROP enum type (status langganan user)
DROP TYPE IF EXISTS user_subscription_status_enum;

-- (Opsional) DROP extensions — hanya jika tidak dipakai modul lain
-- DROP EXTENSION IF EXISTS btree_gist;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
