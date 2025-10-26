-- +migrate Down
BEGIN;

-- =========================================================
-- DROP OBJECTS IN REVERSE DEPENDENCY ORDER
-- (Indexes on dropped tables fall automatically)
-- =========================================================

-- student_bills → depends on bill_batches & masjid_students
DROP TABLE IF EXISTS student_bills CASCADE;

-- bill_batches (drops all ix_* & uq_* on this table)
DROP TABLE IF EXISTS bill_batches CASCADE;

-- general_billings (drops its indexes)
DROP TABLE IF EXISTS general_billings CASCADE;

-- fee_rules (drops its indexes & EXCLUDE constraints)
DROP TABLE IF EXISTS fee_rules CASCADE;

-- general_billing_kinds (MASTER): keep the table.
-- Revert ONLY the indexes introduced by this migration.

DROP INDEX IF EXISTS uq_gbk_code_per_tenant_alive;
DROP INDEX IF EXISTS uq_gbk_code_global_alive;
DROP INDEX IF EXISTS ix_gbk_tenant_active;

-- =========================================================
-- ENUMS
-- =========================================================
DROP TYPE IF EXISTS fee_scope;

-- (Extensions left intact intentionally:
-- pgcrypto, pg_trgm, btree_gist)

COMMIT;