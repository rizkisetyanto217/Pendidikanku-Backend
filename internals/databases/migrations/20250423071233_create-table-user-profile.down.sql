-- ===== users_profile (drop child dulu) =====
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;
DROP FUNCTION IF EXISTS set_updated_at_users_profile();

DROP INDEX IF EXISTS idx_users_profile_user_id_alive;
-- UNIQUE constraint ikut terhapus saat DROP TABLE
DROP TABLE IF EXISTS users_profile;

-- ===== users =====
DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
DROP FUNCTION IF EXISTS set_updated_at_users();

-- Index FTS & kolomnya
DROP INDEX IF EXISTS idx_users_user_name_search;
ALTER TABLE users DROP COLUMN IF EXISTS user_name_search;

-- Indeks lainnya
DROP INDEX IF EXISTS idx_users_user_name_lower;
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_user_name;

-- Terakhir: tabel users (pastikan tidak ada child lain yang bergantung)
DROP TABLE IF EXISTS users;

-- Extensions TIDAK di-drop (bisa dipakai objek lain)
-- (pgcrypto, pg_trgm, citext tetap dibiarkan)
