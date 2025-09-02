-- +migrate Down
BEGIN;

-- =========================================================
-- 1) USERS_PROFILE_FORMAL
-- =========================================================

-- Drop trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_formal ON users_profile_formal;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_profile_formal_user_alive;
DROP INDEX IF EXISTS idx_users_profile_formal_father_phone;
DROP INDEX IF EXISTS idx_users_profile_formal_mother_phone;
DROP INDEX IF EXISTS idx_users_profile_formal_guardian_phone;

-- Drop table
DROP TABLE IF EXISTS users_profile_formal;


-- =========================================================
-- 2) USERS_PROFILE_DOCUMENTS
-- =========================================================

-- Drop trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_documents ON users_profile_documents;

-- Drop indexes
DROP INDEX IF EXISTS uq_user_doc_type_alive;
DROP INDEX IF EXISTS idx_users_profile_documents_user_alive;
DROP INDEX IF EXISTS idx_users_profile_documents_user_type_alive;
DROP INDEX IF EXISTS idx_users_profile_documents_gc_due;
DROP INDEX IF EXISTS idx_users_profile_documents_user_uploaded_alive;

-- Drop table
DROP TABLE IF EXISTS users_profile_documents;


-- =========================================================
-- 3) Utility function (set_updated_at) 
--    Hapus hanya kalau kamu yakin tidak dipakai tabel lain.
-- =========================================================
-- DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;
