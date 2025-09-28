BEGIN;

-- ================================
-- DROP TABLES (reverse order)
-- ================================
DROP TABLE IF EXISTS payment_transactions CASCADE;
DROP TABLE IF EXISTS payment_intents CASCADE;

COMMIT;
