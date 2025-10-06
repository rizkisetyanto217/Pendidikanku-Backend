-- =========================================================
-- DOWN — balikkan perubahan dari file .up.sql
-- =========================================================

-- =====================================================================
-- 1) MASJID PROFILES
--    - Drop indexes
--    - Drop table
-- =====================================================================

-- Indexes (MASJID PROFILES)
DROP INDEX IF EXISTS idx_mpp_principal_user_id_alive;
DROP INDEX IF EXISTS idx_mpp_contact_email_lower_alive;
DROP INDEX IF EXISTS idx_mpp_school_email_lower_alive;
DROP INDEX IF EXISTS idx_mpp_accreditation_alive;
DROP INDEX IF EXISTS idx_mpp_founded_year_alive;
DROP INDEX IF EXISTS idx_mpp_is_boarding_alive;
DROP INDEX IF EXISTS gist_mpp_earth_alive;
DROP INDEX IF EXISTS trgm_mpp_address_alive;
DROP INDEX IF EXISTS trgm_mpp_description_alive;
DROP INDEX IF EXISTS brin_mpp_created_at;
DROP INDEX IF EXISTS brin_mpp_updated_at;
DROP INDEX IF EXISTS ux_mpp_npsn_alive;
DROP INDEX IF EXISTS ux_mpp_nss_alive;

-- Table
DROP TABLE IF EXISTS masjid_profiles;

-- =====================================================================
-- 2) MASJIDS
--    - Drop triggers
--    - Drop functions
--    - Drop indexes
--    - Drop table
-- =====================================================================

-- Triggers
DROP TRIGGER IF EXISTS trg_masjids_sync_is_verified ON masjids;
DROP TRIGGER IF EXISTS trg_masjids_set_updated_at   ON masjids;

-- Functions (harus setelah triggers di-drop)
DROP FUNCTION IF EXISTS sync_masjid_is_verified();
DROP FUNCTION IF EXISTS set_masjid_updated_at();

-- Indexes (MASJIDS)
DROP INDEX IF EXISTS idx_masjids_name_trgm;
DROP INDEX IF EXISTS idx_masjids_location_trgm;
DROP INDEX IF EXISTS idx_masjids_name_lower;
DROP INDEX IF EXISTS ux_masjids_domain_ci;
DROP INDEX IF EXISTS ux_masjids_slug_ci;
DROP INDEX IF EXISTS idx_masjids_slug_lower;
DROP INDEX IF EXISTS idx_masjids_yayasan;
DROP INDEX IF EXISTS idx_masjids_current_plan;
DROP INDEX IF EXISTS gin_masjids_levels;
DROP INDEX IF EXISTS brin_masjids_created_at;
DROP INDEX IF EXISTS idx_masjids_active_alive;
DROP INDEX IF EXISTS idx_masjids_tenant_profile;
DROP INDEX IF EXISTS brin_masjids_icon_delete_pending_until;
DROP INDEX IF EXISTS brin_masjids_logo_delete_pending_until;
DROP INDEX IF EXISTS brin_masjids_background_delete_pending_until;
DROP INDEX IF EXISTS idx_masjids_city_alive;

-- Table
DROP TABLE IF EXISTS masjids;

-- =====================================================================
-- 3) ENUM TYPES
--    - Hapus enum yang dibuat di .up.sql
--    - CASCADE agar tidak nyangkut dependensi tersembunyi
-- =====================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tenant_profile_enum') THEN
    EXECUTE 'DROP TYPE tenant_profile_enum CASCADE';
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    EXECUTE 'DROP TYPE verification_status_enum CASCADE';
  END IF;
END$$;

-- =====================================================================
-- 4) HOUSEKEEPING (opsional)
--    - Objek lama yang mungkin kamu “hidupkan lagi” di masa depan
--    - Tidak ada yang perlu dibalikin di sini karena di .up sudah drop FTS lama
-- =====================================================================
