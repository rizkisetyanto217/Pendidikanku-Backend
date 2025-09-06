BEGIN;

-- =========================================
-- DROP INDEXES
-- =========================================
DROP INDEX IF EXISTS idx_user_teachers_field_trgm;
DROP INDEX IF EXISTS idx_user_teachers_field_lower;
DROP INDEX IF EXISTS idx_user_teachers_search;

DROP INDEX IF EXISTS idx_user_teachers_active;
DROP INDEX IF EXISTS ix_user_teachers_active_verified_created;

DROP INDEX IF EXISTS gin_user_teachers_specialties;
DROP INDEX IF EXISTS gin_user_teachers_certificates;
DROP INDEX IF EXISTS gin_user_teachers_links;

DROP INDEX IF EXISTS brin_user_teachers_created_at;

-- =========================================
-- DROP TABLE
-- =========================================
DROP TABLE IF EXISTS user_teachers;

COMMIT;
