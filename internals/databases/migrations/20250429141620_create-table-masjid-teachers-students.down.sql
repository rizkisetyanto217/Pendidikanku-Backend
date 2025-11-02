BEGIN;

-- =========================================================
-- DROP INDEXES (MASJID_TEACHERS)
-- =========================================================
DROP INDEX IF EXISTS ux_mtj_school_user_alive;
DROP INDEX IF EXISTS ux_mtj_code_alive_ci;
DROP INDEX IF EXISTS ux_mtj_nip_alive_ci;
DROP INDEX IF EXISTS ix_mtj_tenant_active_public_created;
DROP INDEX IF EXISTS ix_mtj_tenant_verified_created;
DROP INDEX IF EXISTS ix_mtj_tenant_employment_created;
DROP INDEX IF EXISTS idx_mtj_user_alive;
DROP INDEX IF EXISTS idx_mtj_school_alive;
DROP INDEX IF EXISTS gin_mtj_notes_trgm_alive;
DROP INDEX IF EXISTS brin_mtj_joined_at;
DROP INDEX IF EXISTS brin_mtj_created_at;

-- =========================================================
-- DROP INDEXES (MASJID_STUDENTS)
-- =========================================================
DROP INDEX IF EXISTS uq_ms_user_per_school_live;
DROP INDEX IF EXISTS ux_ms_code_alive_ci;
DROP INDEX IF EXISTS ix_ms_tenant_status_created;
DROP INDEX IF EXISTS idx_ms_school_alive;
DROP INDEX IF EXISTS idx_ms_user_alive;
DROP INDEX IF EXISTS gin_ms_note_trgm_alive;
DROP INDEX IF EXISTS brin_ms_created_at;

-- =========================================================
-- DROP CONSTRAINTS (MASJID_TEACHERS & STUDENTS)
-- =========================================================
ALTER TABLE school_teachers
  DROP CONSTRAINT IF EXISTS uq_mtj_id_school;

ALTER TABLE school_students
  DROP CONSTRAINT IF EXISTS uq_ms_id_school;

-- =========================================================
-- DROP TABLES
-- =========================================================
DROP TABLE IF EXISTS school_teachers;
DROP TABLE IF EXISTS school_students;

-- =========================================================
-- DROP ENUMS
-- =========================================================
DROP TYPE IF EXISTS teacher_employment_enum;

COMMIT;
