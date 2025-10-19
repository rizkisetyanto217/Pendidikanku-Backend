-- +migrate Down
BEGIN;

-- ============================
-- 1) USER general billings (child)
-- ============================
DROP TABLE IF EXISTS user_general_billings;

-- ============================
-- 2) GENERAL billings (parent)
-- ============================
DROP TABLE IF EXISTS general_billings;

-- ============================
-- 3) SPP user billings (child)
-- ============================
DROP TABLE IF EXISTS user_spp_billings;

-- ============================
-- 4) SPP billings (parent)
-- ============================
DROP TABLE IF EXISTS spp_billings;

-- ============================
-- 5) SPP fee rules (independen)
-- ============================
DROP TABLE IF EXISTS spp_fee_rules;

-- ============================
-- 6) MASTER: general_billing_kinds
-- ============================
DROP TABLE IF EXISTS general_billing_kinds;

COMMIT;
