BEGIN;

-- ================================
-- DROP TABLES (reverse order)
-- ================================
DROP TABLE IF EXISTS student_payments CASCADE;
DROP TABLE IF EXISTS student_bills CASCADE;
DROP TABLE IF EXISTS school_fee_settings CASCADE;
DROP TABLE IF EXISTS fee_categories CASCADE;

-- ================================
-- EXTENSIONS
-- (biarkan kalau shared, tapi kalau mau benar-benar full rollback bisa di-drop)
-- ================================
-- DROP EXTENSION IF EXISTS btree_gist;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
