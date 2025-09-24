-- =========================================================
-- DOWN — USERS & USERS_PROFILE (rollback)
-- =========================================================
BEGIN;

-- -------------------------------
-- 1) Hapus index di user_profiles
-- -------------------------------
DROP INDEX IF EXISTS idx_users_profile_display_name_trgm;
DROP INDEX IF EXISTS idx_users_profile_location;
DROP INDEX IF EXISTS idx_users_profile_phone;
DROP INDEX IF EXISTS idx_users_profile_gender;
DROP INDEX IF EXISTS idx_users_profile_user_id_alive;

-- -------------------------------
-- 2) Hapus tabel user_profiles
-- -------------------------------
DROP TABLE IF EXISTS user_profiles CASCADE;

-- -------------------------------
-- 3) Hapus index di users
-- -------------------------------
DROP INDEX IF EXISTS idx_users_user_search;
DROP INDEX IF EXISTS idx_users_full_name_lower;
DROP INDEX IF EXISTS idx_users_user_name_lower;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_full_name;
DROP INDEX IF EXISTS idx_users_user_name;

-- -------------------------------
-- 4) Hapus tabel users
-- -------------------------------
DROP TABLE IF EXISTS users CASCADE;

-- -------------------------------
-- 5) (Opsional) Hapus extensions
-- -------------------------------
-- Note: hati-hati jika dipakai tabel lain.
-- DROP EXTENSION IF EXISTS btree_gin;
-- DROP EXTENSION IF EXISTS citext;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
