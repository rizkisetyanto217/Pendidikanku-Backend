-- =========================================
-- DOWN Migration — ASSESSMENTS (3 tabel final)
-- =========================================
BEGIN;

-- =========================
-- 1) ASSESSMENT_URLS (child)
-- =========================

-- Drop indexes / unique constraints
DROP INDEX IF EXISTS uq_ass_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS uq_ass_urls_assessment_href_alive;
DROP INDEX IF EXISTS ix_ass_urls_by_owner_live;
DROP INDEX IF EXISTS ix_ass_urls_by_masjid_live;
DROP INDEX IF EXISTS ix_ass_urls_purge_due;
DROP INDEX IF EXISTS gin_ass_urls_label_trgm_live;

-- Drop table
DROP TABLE IF EXISTS assessment_urls;

-- =========================
-- 2) ASSESSMENTS (parent)
-- =========================

-- Drop indexes
DROP INDEX IF EXISTS idx_assessments_masjid_created_at;
DROP INDEX IF EXISTS idx_assessments_type_id;
DROP INDEX IF EXISTS idx_assessments_csst;
DROP INDEX IF EXISTS idx_assessments_created_by_teacher;
DROP INDEX IF EXISTS brin_assessments_created_at;

-- Drop FKs (nama constraint bisa beda, tapi kita handle if exists)
ALTER TABLE IF EXISTS assessments DROP CONSTRAINT IF EXISTS fk_assessments_csst;
ALTER TABLE IF EXISTS assessments DROP CONSTRAINT IF EXISTS fk_assessments_csst_masjid_tenant_safe;

-- Drop table
DROP TABLE IF EXISTS assessments;

-- Drop helper index di CSST (tenant-safe)
DROP INDEX IF EXISTS uq_csst_id_masjid;

-- =========================
-- 3) ASSESSMENT_TYPES (master)
-- =========================

-- Drop indexes
DROP INDEX IF EXISTS uq_assessment_types_masjid_key;
DROP INDEX IF EXISTS idx_assessment_types_masjid_active;

-- Drop table
DROP TABLE IF EXISTS assessment_types;

COMMIT;
