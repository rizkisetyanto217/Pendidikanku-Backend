-- =========================================================
-- DOWN: USERS & USERS_PROFILE
-- =========================================================
BEGIN;

-- ---------- DROP TRIGGERS ----------
DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_formal ON users_profile_formal;
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_documents ON users_profile_documents;

-- ---------- DROP TRIGGER FUNCTION ----------
DROP FUNCTION IF EXISTS set_updated_at;

-- ---------- DROP INDEXES ----------
-- users
DROP INDEX IF EXISTS idx_users_user_name;
DROP INDEX IF EXISTS idx_users_full_name;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_user_name_trgm;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_users_user_name_lower;
DROP INDEX IF EXISTS idx_users_full_name_lower;
DROP INDEX IF EXISTS idx_users_user_search;

-- users_profile
DROP INDEX IF EXISTS idx_users_profile_user_id_alive;
DROP INDEX IF EXISTS idx_users_profile_gender;
DROP INDEX IF EXISTS idx_users_profile_phone;

-- users_profile_formal
DROP INDEX IF EXISTS idx_users_profile_formal_user_alive;
DROP INDEX IF EXISTS idx_users_profile_formal_location;

-- users_profile_documents
DROP INDEX IF EXISTS idx_users_profile_documents_user_alive;
DROP INDEX IF EXISTS idx_users_profile_documents_doctype;

-- ---------- DROP TABLES ----------
DROP TABLE IF EXISTS users_profile_documents CASCADE;
DROP TABLE IF EXISTS users_profile_formal CASCADE;
DROP TABLE IF EXISTS users_profile CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- ---------- DROP EXTENSIONS (opsional, hati-hati kalau dipakai global) ----------
-- DROP EXTENSION IF EXISTS pgcrypto;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS citext;
-- DROP EXTENSION IF EXISTS btree_gin;

COMMIT;
