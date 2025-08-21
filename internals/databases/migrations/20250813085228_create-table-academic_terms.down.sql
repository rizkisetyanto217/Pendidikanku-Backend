-- =========================================================
-- DOWN: academic_terms (rollback)
-- =========================================================

-- 1) Drop trigger
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;
DROP FUNCTION IF EXISTS fn_touch_academic_terms_updated_at;

-- 2) Drop constraints
ALTER TABLE academic_terms
DROP CONSTRAINT IF EXISTS ex_academic_terms_no_overlap_per_tenant;

-- 3) Drop indexes
DROP INDEX IF EXISTS uq_academic_terms_tenant_year_name_live;
DROP INDEX IF EXISTS uq_academic_terms_one_active_per_tenant;
DROP INDEX IF EXISTS ix_academic_terms_tenant_dates;
DROP INDEX IF EXISTS ix_academic_terms_period_gist;
DROP INDEX IF EXISTS ix_academic_terms_tenant_active_live;
DROP INDEX IF EXISTS ix_academic_terms_name_trgm;
DROP INDEX IF EXISTS ix_academic_terms_year;
DROP INDEX IF EXISTS ix_academic_terms_year_trgm;

-- 4) Drop table
DROP TABLE IF EXISTS academic_terms;
