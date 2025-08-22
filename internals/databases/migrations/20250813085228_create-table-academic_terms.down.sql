BEGIN;

-- Trigger
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;

-- Indexes
DROP INDEX IF EXISTS ix_academic_terms_tenant_dates;
DROP INDEX IF EXISTS ix_academic_terms_period_gist;
DROP INDEX IF EXISTS ix_academic_terms_tenant_active_live;
DROP INDEX IF EXISTS ix_academic_terms_name_trgm;
DROP INDEX IF EXISTS ix_academic_terms_year;
DROP INDEX IF EXISTS ix_academic_terms_year_trgm_lower;
DROP INDEX IF EXISTS ix_academic_terms_tenant_created_at;
DROP INDEX IF EXISTS ix_academic_terms_tenant_updated_at;

-- Legacy unique (jaga-jaga)
DROP INDEX IF EXISTS uq_academic_terms_tenant_year_name_live;
DROP INDEX IF EXISTS uq_academic_terms_one_active_per_tenant;

-- Table
DROP TABLE IF EXISTS academic_terms;

-- Function
DROP FUNCTION IF EXISTS fn_touch_academic_terms_updated_at();

COMMIT;
