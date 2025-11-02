-- +migrate Down
BEGIN;

-- ===== user_general_billings: drop indexes dulu (aman bila sudah tidak ada) =====
DROP INDEX IF EXISTS ix_ugb_created_at_alive;
DROP INDEX IF EXISTS ix_ugb_status_alive;
DROP INDEX IF EXISTS ix_ugb_billing_alive;
DROP INDEX IF EXISTS ix_ugb_school_alive;
DROP INDEX IF EXISTS uq_ugb_per_payer_alive;
DROP INDEX IF EXISTS uq_ugb_per_student_alive;
-- (opsional) bila sempat dibuat
DROP INDEX IF EXISTS ix_ugb_meta_gin_alive;

-- ===== general_billings: drop indexes =====
DROP INDEX IF EXISTS ix_gb_school_updated_at_alive;
DROP INDEX IF EXISTS ix_gb_school_created_at_alive;
DROP INDEX IF EXISTS ix_gb_kind_alive;
DROP INDEX IF EXISTS ix_gb_due_alive;
DROP INDEX IF EXISTS ix_gb_tenant_kind_active_created;
DROP INDEX IF EXISTS uq_general_billings_code_per_tenant_alive;

-- ===== drop tables (anak dulu baru parent) =====
DROP TABLE IF EXISTS user_general_billings CASCADE;
DROP TABLE IF EXISTS general_billings CASCADE;

COMMIT;
