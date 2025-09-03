-- =========================================================
-- DOWN #1 — USERS & USERS_PROFILE
-- =========================================================
BEGIN;

-- ---------- USERS_PROFILE ----------
-- drop triggers
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;

-- drop indexes
DROP INDEX IF EXISTS idx_users_profile_location;
DROP INDEX IF EXISTS idx_users_profile_phone;
DROP INDEX IF EXISTS idx_users_profile_gender;
DROP INDEX IF EXISTS idx_users_profile_user_id_alive;

-- drop table
DROP TABLE IF EXISTS users_profile;

-- ---------- USERS ----------
-- drop trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;

-- drop FTS column + index
DROP INDEX IF EXISTS idx_users_user_search;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS user_search;

-- drop other indexes
DROP INDEX IF EXISTS idx_users_full_name_lower;
DROP INDEX IF EXISTS idx_users_user_name_lower;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_full_name;
DROP INDEX IF EXISTS idx_users_user_name;

-- drop table
DROP TABLE IF EXISTS users;

-- NOTE:
-- Fungsi set_updated_at() bisa dipakai tabel lain. Hanya drop jika memang dibuat khusus migration ini.
-- DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;
