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

-- Catatan: jika ada tabel lain yang FK ke tabel-tabel ini,
-- drop dulu tabel/constraint tersebut. Hindari CASCADE kecuali memang ingin hard teardown.

DROP TABLE IF EXISTS masjids_profiles;
DROP TABLE IF EXISTS user_follow_masjid;
DROP TABLE IF EXISTS masjids;

-- =============================
-- DROP ENUM (hanya jika tidak dipakai)
-- =============================
DO $$
DECLARE
  cnt int;
BEGIN
  -- Hitung berapa kolom yang masih memakai tipe enum ini
  SELECT COUNT(*) INTO cnt
  FROM pg_attribute a
  WHERE a.atttypid = 'verification_status_enum'::regtype
    AND a.attnum > 0
    AND NOT a.attisdropped;

  IF cnt = 0 THEN
    EXECUTE 'DROP TYPE verification_status_enum';
  END IF;
END$$;

COMMIT;
