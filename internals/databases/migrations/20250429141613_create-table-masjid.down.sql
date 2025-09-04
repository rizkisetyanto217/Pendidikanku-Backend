BEGIN;

-- =============================
-- DROP TRIGGERS & FUNCTIONS
-- =============================

-- masjids_profiles: drop trigger & function
DROP TRIGGER IF EXISTS trg_set_updated_at_masjids_profiles ON masjids_profiles;
DROP FUNCTION IF EXISTS set_updated_at_masjids_profiles();

-- masjids: drop trigger & function verifikasi
DROP TRIGGER IF EXISTS trg_sync_verification ON masjids;
DROP FUNCTION IF EXISTS sync_masjid_verification_flags();

-- =============================
-- DROP TABLES (child -> parent)
-- =============================

DROP TABLE IF EXISTS masjids_profiles;
DROP TABLE IF EXISTS user_follow_masjid;
DROP TABLE IF EXISTS masjids;

-- =============================
-- DROP ENUM (jika tidak dipakai)
-- =============================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    DROP TYPE verification_status_enum;
  END IF;
END$$;

COMMIT;
