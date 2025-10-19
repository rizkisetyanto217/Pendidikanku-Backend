BEGIN;

-- =========================================================
-- 1) DROP CHILD TABLES DULU (karena FK ke payments)
-- =========================================================
DROP TABLE IF EXISTS payment_gateway_events CASCADE;

-- =========================================================
-- 2) DROP payments
-- =========================================================
DROP TABLE IF EXISTS payments CASCADE;

-- =========================================================
-- 3) DROP ENUM TYPES (hanya jika sudah tidak dipakai kolom lain)
--    Cek di information_schema.columns; kalau tidak ada yang pakai, baru di-drop
-- =========================================================

-- payment_status
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE udt_name = 'payment_status'
  ) THEN
    DROP TYPE IF EXISTS payment_status;
  END IF;
END$$;

-- payment_method
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE udt_name = 'payment_method'
  ) THEN
    DROP TYPE IF EXISTS payment_method;
  END IF;
END$$;

-- payment_gateway_provider
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE udt_name = 'payment_gateway_provider'
  ) THEN
    DROP TYPE IF EXISTS payment_gateway_provider;
  END IF;
END$$;

-- gateway_event_status
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE udt_name = 'gateway_event_status'
  ) THEN
    DROP TYPE IF EXISTS gateway_event_status;
  END IF;
END$$;

COMMIT;
