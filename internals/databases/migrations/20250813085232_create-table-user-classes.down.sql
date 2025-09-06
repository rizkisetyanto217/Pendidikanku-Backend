BEGIN;

-- =========================================================
-- DROP INDEXES (user_class_sections)
-- =========================================================
DROP INDEX IF EXISTS uq_user_class_sections_active_per_user_class;
DROP INDEX IF EXISTS idx_user_class_sections_user_class;
DROP INDEX IF EXISTS idx_user_class_sections_section;
DROP INDEX IF EXISTS idx_user_class_sections_assigned_at;
DROP INDEX IF EXISTS idx_user_class_sections_unassigned_at;
DROP INDEX IF EXISTS idx_user_class_sections_masjid;
DROP INDEX IF EXISTS idx_user_class_sections_masjid_active;
DROP INDEX IF EXISTS brin_ucs_created_at;

-- =========================================================
-- DROP INDEXES (user_classes)
-- =========================================================
DROP INDEX IF EXISTS uq_uc_active_per_user_class_term;
DROP INDEX IF EXISTS ix_uc_tenant_user_created;
DROP INDEX IF EXISTS ix_uc_tenant_class_term_active;
DROP INDEX IF EXISTS ix_uc_tenant_status_created;
DROP INDEX IF EXISTS idx_uc_user_alive;
DROP INDEX IF EXISTS idx_uc_class_alive;
DROP INDEX IF EXISTS idx_uc_term_alive;
DROP INDEX IF EXISTS idx_uc_masjid_alive;
DROP INDEX IF EXISTS idx_uc_masjid_student_alive;
DROP INDEX IF EXISTS brin_uc_created_at;

-- =========================================================
-- DROP CONSTRAINTS / FKs
-- =========================================================
-- user_class_sections
ALTER TABLE user_class_sections
  DROP CONSTRAINT IF EXISTS fk_ucs_user_class_masjid_pair,
  DROP CONSTRAINT IF EXISTS fk_ucs_section_masjid_pair,
  DROP CONSTRAINT IF EXISTS chk_ucs_dates;

-- user_classes
ALTER TABLE user_classes
  DROP CONSTRAINT IF EXISTS fk_uc_class_masjid_pair,
  DROP CONSTRAINT IF EXISTS fk_uc_term_masjid_pair,
  DROP CONSTRAINT IF EXISTS fk_uc_masjid_student,
  DROP CONSTRAINT IF EXISTS uq_user_classes_id_masjid,
  DROP CONSTRAINT IF EXISTS chk_uc_dates;

-- (Opsional) kalau saat UP kamu menambahkan pair unik di class_sections:
ALTER TABLE class_sections
  DROP CONSTRAINT IF EXISTS uq_class_sections_id_masjid;

-- =========================================================
-- DROP TABLES
-- =========================================================
DROP TABLE IF EXISTS user_class_sections;
DROP TABLE IF EXISTS user_classes;

COMMIT;
