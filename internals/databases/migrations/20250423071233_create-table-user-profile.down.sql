-- Hapus index tambahan
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_search;
DROP INDEX IF EXISTS idx_users_user_name_lower;

-- Hapus kolom FTS
ALTER TABLE users DROP COLUMN IF EXISTS user_name_search;

-- Hapus index dasar
DROP INDEX IF EXISTS idx_users_user_name;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_role;

-- Hapus tabel terkait
DROP TABLE IF EXISTS users_profile;
DROP TABLE IF EXISTS users;
