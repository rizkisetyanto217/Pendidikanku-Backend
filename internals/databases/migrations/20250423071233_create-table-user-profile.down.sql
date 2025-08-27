-- =========================
-- ========= DOWN ==========
-- =========================
BEGIN;

-- Hapus trigger dulu
DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
DROP FUNCTION IF EXISTS set_updated_at_users();

DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;
DROP FUNCTION IF EXISTS set_updated_at_users_profile();

-- Hapus index terkait users
DROP INDEX IF EXISTS idx_users_user_name;
DROP INDEX IF EXISTS idx_users_full_name;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_lower;
DROP INDEX IF EXISTS idx_users_full_name_lower;
DROP INDEX IF EXISTS idx_users_user_search;
DROP INDEX IF EXISTS idx_users_user_name_search;

-- Hapus index terkait users_profile
DROP INDEX IF EXISTS idx_users_profile_user_id_alive;

-- Hapus constraint unik users_profile
ALTER TABLE IF EXISTS users_profile
  DROP CONSTRAINT IF EXISTS users_profile_user_id_key;

-- Hapus tabel (anak → induk)
DROP TABLE IF EXISTS users_profile;
DROP TABLE IF EXISTS users;

COMMIT;
