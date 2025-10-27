-- +migrate Down
BEGIN;

-- 1) Drop anak-anak dulu (punya FK ke parent)
DROP TABLE IF EXISTS student_bills CASCADE;

-- 2) Tabel yang berdiri sendiri / refer ke katalog
DROP TABLE IF EXISTS fee_rules CASCADE;
DROP TABLE IF EXISTS general_billings CASCADE;

-- 3) Batch (direferensi student_bills)
DROP TABLE IF EXISTS bill_batches CASCADE;

-- 4) Katalog master
DROP TABLE IF EXISTS general_billing_kinds CASCADE;

-- 5) Enum (baru bisa di-drop setelah semua tabel yang pakai hilang)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fee_scope') THEN
    DROP TYPE fee_scope;
  END IF;
END$$;

-- Catatan: Extensions (pgcrypto, pg_trgm, btree_gist) sengaja tidak di-drop.
-- Jika benar-benar ingin dihapus (hati-hati, bisa dipakai object lain):
-- DROP EXTENSION IF EXISTS btree_gist;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
