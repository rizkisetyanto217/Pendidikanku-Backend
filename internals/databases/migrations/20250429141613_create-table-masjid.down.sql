-- === 3) Drop MASJIDS_PROFILES terlebih dulu (child) ===
DROP TRIGGER  IF EXISTS trg_set_updated_at_masjids_profiles ON masjids_profiles;
DROP FUNCTION IF EXISTS set_updated_at_masjids_profiles();
DROP TABLE   IF EXISTS masjids_profiles;

-- === 2) Drop USER_FOLLOW_MASJID (child) ===
DROP INDEX IF EXISTS idx_follow_user_id;
DROP INDEX IF EXISTS idx_follow_masjid_id;
DROP TABLE IF EXISTS user_follow_masjid;

-- === 1) Bersihkan index/kolom/trigger di MASJIDS lalu drop tabelnya ===
DROP TRIGGER  IF EXISTS trg_set_updated_at_masjids ON masjids;
DROP FUNCTION IF EXISTS set_updated_at_masjids();

-- FTS
DROP INDEX IF EXISTS idx_masjids_search;
ALTER TABLE masjids DROP COLUMN IF EXISTS masjid_search;

-- Index pencarian & filter
DROP INDEX IF EXISTS idx_masjids_earth;
DROP INDEX IF EXISTS idx_masjids_slug_alive;
DROP INDEX IF EXISTS idx_masjids_verified;
DROP INDEX IF EXISTS idx_masjids_name_lower;
DROP INDEX IF EXISTS idx_masjids_location_trgm;
DROP INDEX IF EXISTS idx_masjids_name_trgm;
DROP INDEX IF EXISTS idx_masjids_domain_ci_unique;

-- Constraints koordinat
ALTER TABLE masjids DROP CONSTRAINT IF EXISTS masjids_lon_chk;
ALTER TABLE masjids DROP CONSTRAINT IF EXISTS masjids_lat_chk;

-- Terakhir: drop tabel parent
DROP TABLE IF EXISTS masjids;

-- (Extensions tidak di-drop; kemungkinan dipakai objek lain)
