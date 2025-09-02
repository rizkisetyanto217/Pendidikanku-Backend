-- =========================================================
-- DOWN: USERS_TEACHER
-- =========================================================
BEGIN;

DROP TRIGGER IF EXISTS trg_set_updated_at_users_teacher ON users_teacher;

DROP INDEX IF EXISTS idx_users_teacher_field_trgm;
DROP INDEX IF EXISTS idx_users_teacher_field_lower;
DROP INDEX IF EXISTS idx_users_teacher_search;
DROP INDEX IF EXISTS idx_users_teacher_active;
DROP INDEX IF EXISTS brin_users_teacher_created_at;

DROP TABLE IF EXISTS users_teacher;

-- catatan: fungsi set_updated_at() tidak di-drop karena mungkin dipakai tabel lain
COMMIT;
