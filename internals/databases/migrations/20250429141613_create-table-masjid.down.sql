-- =========================================
-- DOWN MIGRATION untuk masjids, user_follow_masjid, masjids_profiles
-- =========================================

-- -----------------------------
-- DROP RELATIONSHIPS
-- -----------------------------
DROP TABLE IF EXISTS user_follow_masjid CASCADE;

-- -----------------------------
-- DROP masjids_profiles
-- -----------------------------
-- Hapus trigger
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_set_updated_at_masjids_profiles') THEN
    EXECUTE 'DROP TRIGGER trg_set_updated_at_masjids_profiles ON masjids_profiles';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

-- Hapus function trigger
DROP FUNCTION IF EXISTS set_updated_at_masjids_profiles() CASCADE;

-- Drop tabel
DROP TABLE IF EXISTS masjids_profiles CASCADE;

-- -----------------------------
-- DROP masjids
-- -----------------------------
-- Hapus trigger
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_set_updated_at_masjids') THEN
    EXECUTE 'DROP TRIGGER trg_set_updated_at_masjids ON masjids';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_sync_verification') THEN
    EXECUTE 'DROP TRIGGER trg_sync_verification ON masjids';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_handle_masjid_image_trash') THEN
    EXECUTE 'DROP TRIGGER trg_handle_masjid_image_trash ON masjids';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

-- Hapus function trigger
DROP FUNCTION IF EXISTS set_updated_at_masjids() CASCADE;
DROP FUNCTION IF EXISTS sync_masjid_verification_flags() CASCADE;
DROP FUNCTION IF EXISTS handle_masjid_image_trash() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS idx_masjids_name_trgm;
DROP INDEX IF EXISTS idx_masjids_location_trgm;
DROP INDEX IF EXISTS idx_masjids_name_lower;
DROP INDEX IF EXISTS idx_masjids_domain_ci_unique;
DROP INDEX IF EXISTS idx_masjids_active;
DROP INDEX IF EXISTS idx_masjids_verified;
DROP INDEX IF EXISTS idx_masjids_verif_status;
DROP INDEX IF EXISTS idx_masjids_slug_alive;
DROP INDEX IF EXISTS idx_masjids_search;
DROP INDEX IF EXISTS idx_masjids_earth;
DROP INDEX IF EXISTS idx_masjids_image_gc_due;
DROP INDEX IF EXISTS idx_masjids_image_main_gc_due;
DROP INDEX IF EXISTS idx_masjids_image_bg_gc_due;
DROP INDEX IF EXISTS idx_masjids_yayasan;

-- Drop tabel
DROP TABLE IF EXISTS masjids CASCADE;

-- -----------------------------
-- DROP ENUM
-- -----------------------------
DROP TYPE IF EXISTS verification_status_enum CASCADE;

-- -----------------------------
-- (Optional) DROP EXTENSIONS
-- -----------------------------
-- Kalau memang hanya dipakai untuk masjids dan aman untuk dihapus:
-- DROP EXTENSION IF EXISTS pgcrypto;
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS cube;
-- DROP EXTENSION IF EXISTS earthdistance;

-- =========================================
-- END DOWN
-- =========================================
