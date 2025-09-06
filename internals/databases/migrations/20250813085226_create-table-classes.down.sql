BEGIN;

-- =========================================================
-- DROP TRIGGERS & FUNCTIONS
-- =========================================================
DROP TRIGGER IF EXISTS trg_classes_quota_nonnegative ON classes;

DROP FUNCTION IF EXISTS fn_classes_quota_nonnegative();
DROP FUNCTION IF EXISTS class_claim(UUID);
DROP FUNCTION IF EXISTS class_release(UUID);

-- =========================================================
-- DROP CONSTRAINTS (cek nama di pg_constraint kalau beda)
-- =========================================================
ALTER TABLE classes
  DROP CONSTRAINT IF EXISTS fk_classes_parent_same_masjid,
  DROP CONSTRAINT IF EXISTS uq_classes_id_masjid,
  DROP CONSTRAINT IF EXISTS fk_classes_term_masjid_pair,
  DROP CONSTRAINT IF EXISTS ck_classes_pricing_nonneg;

ALTER TABLE academic_terms
  DROP CONSTRAINT IF EXISTS uq_academic_terms_id_masjid;

ALTER TABLE class_parent
  DROP CONSTRAINT IF EXISTS uq_class_parent_id_masjid;

-- =========================================================
-- DROP INDEXES
-- =========================================================
DROP INDEX IF EXISTS uq_classes_slug_per_masjid_active;
DROP INDEX IF EXISTS uq_classes_code_per_masjid_active;
DROP INDEX IF EXISTS idx_classes_masjid;
DROP INDEX IF EXISTS idx_classes_parent;
DROP INDEX IF EXISTS idx_classes_active;
DROP INDEX IF EXISTS idx_classes_created_at;
DROP INDEX IF EXISTS idx_classes_slug;
DROP INDEX IF EXISTS idx_classes_code;
DROP INDEX IF EXISTS ix_classes_tenant_term_open_live;
DROP INDEX IF EXISTS ix_classes_reg_window_live;
DROP INDEX IF EXISTS gin_classes_notes_trgm_live;

DROP INDEX IF EXISTS uq_class_parent_slug_per_masjid_active;
DROP INDEX IF EXISTS uq_class_parent_code_per_masjid_active;
DROP INDEX IF EXISTS idx_class_parent_masjid;
DROP INDEX IF EXISTS idx_class_parent_active;
DROP INDEX IF EXISTS idx_class_parent_created_at;
DROP INDEX IF EXISTS idx_class_parent_slug;
DROP INDEX IF EXISTS idx_class_parent_code;
DROP INDEX IF EXISTS idx_class_parent_mode_lower;

-- =========================================================
-- DROP TABLES
-- =========================================================
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS class_parent;

-- =========================================================
-- DROP ENUMS
-- =========================================================
DROP TYPE IF EXISTS billing_cycle_enum;

COMMIT;
