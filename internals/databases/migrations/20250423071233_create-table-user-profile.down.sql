-- =========================
-- DOWN: USERS & USERS_PROFILE
-- =========================

-- ---------- users_profile ----------
-- Drop trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_profile_user_id_alive;

-- Drop table (FK ke users, jadi users_profile duluan)
DROP TABLE IF EXISTS users_profile;

-- ---------- users ----------
-- Drop trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;

-- Drop FTS index & column
DROP INDEX IF EXISTS idx_users_user_search;
ALTER TABLE users DROP COLUMN IF EXISTS user_search;

-- Drop trigram indexes
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_trgm;

-- Drop lower(prefix) indexes
DROP INDEX IF EXISTS idx_users_full_name_lower;
DROP INDEX IF EXISTS idx_users_user_name_lower;

-- Drop basic btree indexes
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_full_name;
DROP INDEX IF EXISTS idx_users_user_name;

-- Drop table
DROP TABLE IF EXISTS users;

-- ---------- function ----------
-- Hapus function trigger updated_at (jika tidak dipakai object lain)
DROP FUNCTION IF EXISTS set_updated_at();
