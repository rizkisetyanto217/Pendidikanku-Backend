-- =========================================================
-- DOWN MIGRATION (PostgreSQL)
-- =========================================================

-- ============================
-- 1) DONATIONS: triggers, idx, table
-- ============================

-- Trigger & function sinkronisasi
DROP TRIGGER IF EXISTS trg_donations_sync_user_spp_paid ON donations;
DROP FUNCTION IF EXISTS donations_sync_user_spp_paid();

-- Indexes
DROP INDEX IF EXISTS idx_donations_parent_order_id;
DROP INDEX IF EXISTS idx_donations_user_spp_billing_id;
DROP INDEX IF EXISTS idx_donations_masjid_id;
DROP INDEX IF EXISTS idx_donations_user_id;
DROP INDEX IF EXISTS idx_donations_order_id_lower;
DROP INDEX IF EXISTS idx_donations_target_id;
DROP INDEX IF EXISTS idx_donations_target_type;
DROP INDEX IF EXISTS idx_donations_status;

-- Table
DROP TABLE IF EXISTS donations;

-- ============================
-- 2) USER SPP BILLINGS: idx, constraints, table
-- ============================

-- Unique constraint (dibuat via ALTER)
ALTER TABLE IF EXISTS user_spp_billings
  DROP CONSTRAINT IF EXISTS uq_user_spp_billing_per_user;

-- Indexes
DROP INDEX IF EXISTS idx_user_spp_billings_user;
DROP INDEX IF EXISTS idx_user_spp_billings_billing;

-- Table
DROP TABLE IF EXISTS user_spp_billings;

-- ============================
-- 3) SPP BILLINGS: triggers, fks, idx, table
-- ============================

-- Constraint trigger & function tenant check
DROP TRIGGER IF EXISTS trg_spp_term_tenant_check ON spp_billings;
DROP FUNCTION IF EXISTS fn_spp_term_tenant_check();

-- FK ke academic_terms
ALTER TABLE IF EXISTS spp_billings
  DROP CONSTRAINT IF EXISTS fk_spp_billing_term;

-- Indexes (unik & biasa, live & dasar)
DROP INDEX IF EXISTS uq_spp_billings_batch;

DROP INDEX IF EXISTS ix_spp_billings_term_live;
DROP INDEX IF EXISTS gin_spp_billings_title_trgm_live;
DROP INDEX IF EXISTS ix_spp_billings_tenant_month_year_live;
DROP INDEX IF EXISTS ix_spp_billings_tenant_class_due_live;
DROP INDEX IF EXISTS ix_spp_billings_tenant_created_live;

DROP INDEX IF EXISTS idx_spp_billings_class;
DROP INDEX IF EXISTS idx_spp_billings_masjid;

-- Table
DROP TABLE IF EXISTS spp_billings;
