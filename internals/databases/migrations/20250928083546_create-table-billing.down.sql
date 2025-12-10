-- +migrate Down
BEGIN;

-- Urutan: child dulu, baru parent, baru ENUM

-- 1) Hapus tabel yang bergantung ke general_billings
DROP TABLE IF EXISTS user_general_billings;

-- 2) Hapus tabel lain yang berdiri sendiri di migration ini
DROP TABLE IF EXISTS bill_batches;
DROP TABLE IF EXISTS fee_rules;

-- 3) Hapus header tagihan
DROP TABLE IF EXISTS general_billings;

-- 4) Hapus ENUM (setelah semua tabel yang pakai di-drop)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'general_billing_category') THEN
    DROP TYPE general_billing_category;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fee_scope') THEN
    DROP TYPE fee_scope;
  END IF;
END$$;

-- Catatan:
-- EXTENSION (pgcrypto, pg_trgm, btree_gist, unaccent) sengaja tidak di-DROP
-- karena sangat mungkin dipakai migration lain juga.

COMMIT;
