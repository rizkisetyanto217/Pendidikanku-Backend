-- =========================================================
-- DOWN Migration — TABEL YAYASANS (FOUNDATION) CLEAN (v2)
-- =========================================================
BEGIN;

-- 1) Drop indexes (aman meski tabel sudah ter-drop sebelumnya)
DROP INDEX IF EXISTS idx_yayasans_name_trgm;
DROP INDEX IF EXISTS idx_yayasans_city_trgm;
DROP INDEX IF EXISTS idx_yayasans_name_lower;
DROP INDEX IF EXISTS idx_yayasans_domain_ci_unique;
DROP INDEX IF EXISTS idx_yayasans_active;
DROP INDEX IF EXISTS idx_yayasans_verified;
DROP INDEX IF EXISTS idx_yayasans_verif_status;
DROP INDEX IF EXISTS idx_yayasans_slug_alive;
DROP INDEX IF EXISTS idx_yayasans_search;
DROP INDEX IF EXISTS idx_yayasans_earth;
DROP INDEX IF EXISTS idx_yayasans_logo_gc_due;

-- 2) Drop table
DROP TABLE IF EXISTS yayasans;

-- 3) (Opsional) Drop enum verification_status_enum
--    Hanya dijalankan jika enum TIDAK dipakai oleh tabel/kolom lain.
DO $$
DECLARE
  used_count INT;
BEGIN
  -- Cek apakah tipe enum masih dipakai oleh kolom manapun
  SELECT COUNT(*)
    INTO used_count
  FROM pg_attribute a
  JOIN pg_class c ON a.attrelid = c.oid
  JOIN pg_type  t ON a.atttypid = t.oid
  WHERE t.typname = 'verification_status_enum'
    AND a.attnum > 0               -- kolom nyata
    AND c.relkind IN ('r','p','v','m','f');  -- table/partition/view/mview/foreign

  IF used_count = 0 THEN
    DROP TYPE IF EXISTS verification_status_enum;
  END IF;
END$$;

COMMIT;

-- Catatan:
-- - Extensions (pgcrypto, pg_trgm, cube, earthdistance) sengaja TIDAK di-drop
--   karena bisa dipakai objek lain. Jika tetap ingin mencabut, lakukan manual:
--   DROP EXTENSION IF EXISTS earthdistance;
--   DROP EXTENSION IF EXISTS cube;
--   DROP EXTENSION IF EXISTS pg_trgm;
--   DROP EXTENSION IF EXISTS pgcrypto;
