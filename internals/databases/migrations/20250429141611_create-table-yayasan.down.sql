BEGIN;

-- =========================================================
-- ROLLBACK: TABEL YAYASANS (DROP triggers, functions, indexes, table)
-- =========================================================

-- 1) Triggers (jika ada)
DROP TRIGGER IF EXISTS trg_set_updated_at_yayasans ON yayasans;
DROP TRIGGER IF EXISTS trg_sync_yayasan_verification ON yayasans;
DROP TRIGGER IF EXISTS trg_handle_yayasan_logo_trash ON yayasans;

-- 2) Functions (jika ada)
DROP FUNCTION IF EXISTS set_updated_at_yayasans();
DROP FUNCTION IF EXISTS sync_yayasan_verification_flags();
DROP FUNCTION IF EXISTS handle_yayasan_logo_trash();

-- 3) Indexes (jika ada)
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

-- 4) Table
DROP TABLE IF EXISTS yayasans;

-- =========================================================
-- ROLLBACK: PERUBAHAN PADA MASJIDS (kolom & index relasi yayasan)
-- =========================================================

-- Index relasi yayasan
DROP INDEX IF EXISTS idx_masjids_yayasan;

-- Kolom relasi yayasan (otomatis drop FK constraint yang menempel)
ALTER TABLE IF EXISTS masjids
  DROP COLUMN IF EXISTS masjid_yayasan_id;

COMMIT;
