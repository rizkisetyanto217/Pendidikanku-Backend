-- =========================================================
-- PRASYARAT EXTENSIONS
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram search
CREATE EXTENSION IF NOT EXISTS cube;          -- earthdistance requirement
CREATE EXTENSION IF NOT EXISTS earthdistance; -- ll_to_earth()

-- =========================================================
-- ENUMS
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END$$;

-- =========================================================
-- TABEL MASJIDS (INLINE + OPTIMIZED)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Identitas & lokasi
  masjid_name       VARCHAR(100) NOT NULL,
  masjid_bio_short  TEXT,
  masjid_location   TEXT,
  masjid_latitude   DECIMAL(9,6),
  masjid_longitude  DECIMAL(9,6),

  -- Media & maps
  masjid_image_url  TEXT,
  masjid_image_trash_url TEXT,                  -- URL lama saat diganti / dihapus (masuk trash)
  masjid_image_delete_pending_until TIMESTAMP,  -- jadwal auto-delete (default 30 hari dari penggantian)
  masjid_google_maps_url TEXT,

  -- Domain & slug
  masjid_domain VARCHAR(50),
  masjid_slug   VARCHAR(100) UNIQUE NOT NULL,   -- tetap UNIQUE global untuk kesederhanaan

  -- Status & verifikasi
  masjid_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  masjid_verified_at TIMESTAMP,
  masjid_verification_notes TEXT,

  -- Paket aktif
  masjid_current_plan_id UUID REFERENCES masjid_service_plans (masjid_service_plan_id),

  -- Sosial
  masjid_instagram_url TEXT,
  masjid_whatsapp_url  TEXT,
  masjid_youtube_url   TEXT,
  masjid_facebook_url  TEXT,
  masjid_tiktok_url    TEXT,
  masjid_whatsapp_group_ikhwan_url TEXT,
  masjid_whatsapp_group_akhwat_url TEXT,

  -- Full-text search gabungan (name+location+bio)
  masjid_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_name,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_bio_short,'')), 'C')
  ) STORED,

  -- Audit
  masjid_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  masjid_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  masjid_deleted_at TIMESTAMP,

  -- Validasi koordinat
  CONSTRAINT masjids_lat_chk CHECK (masjid_latitude  BETWEEN -90  AND 90),
  CONSTRAINT masjids_lon_chk CHECK (masjid_longitude BETWEEN -180 AND 180)
);

-- =========================================================
-- INDEXES (READ & FILTER OPTIMIZATION)
-- =========================================================

-- Trigram search untuk name & location
CREATE INDEX IF NOT EXISTS idx_masjids_name_trgm
  ON masjids USING gin (masjid_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_masjids_location_trgm
  ON masjids USING gin (masjid_location gin_trgm_ops);

-- Filter case-insensitive by name
CREATE INDEX IF NOT EXISTS idx_masjids_name_lower
  ON masjids (LOWER(masjid_name));

-- Domain unik case-insensitive (hanya yang set)
CREATE UNIQUE INDEX IF NOT EXISTS idx_masjids_domain_ci_unique
  ON masjids (LOWER(masjid_domain))
  WHERE masjid_domain IS NOT NULL;

-- Flag & status (hanya row “alive”)
CREATE INDEX IF NOT EXISTS idx_masjids_active
  ON masjids (masjid_is_active) WHERE masjid_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_masjids_verified
  ON masjids (masjid_is_verified) WHERE masjid_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_masjids_verif_status
  ON masjids (masjid_verification_status) WHERE masjid_deleted_at IS NULL;

-- Slug cepat untuk row “alive” (meski sudah ada UNIQUE, ini buat filter cepat/non-unique)
CREATE INDEX IF NOT EXISTS idx_masjids_slug_alive
  ON masjids (masjid_slug) WHERE masjid_deleted_at IS NULL;

-- Full-text search gabungan
CREATE INDEX IF NOT EXISTS idx_masjids_search
  ON masjids USING gin (masjid_search);

-- Geospatial nearest-neighbor
CREATE INDEX IF NOT EXISTS idx_masjids_earth
  ON masjids USING gist (ll_to_earth(masjid_latitude::float8, masjid_longitude::float8));

-- GC gambar: pilih yang sudah due (trash tidak kosong dan due sudah lewat)
CREATE INDEX IF NOT EXISTS idx_masjids_image_gc_due
  ON masjids (masjid_image_delete_pending_until)
  WHERE masjid_image_trash_url IS NOT NULL;

-- =========================================================
-- TRIGGERS
-- =========================================================

-- 1) Auto-updated_at
CREATE OR REPLACE FUNCTION set_updated_at_masjids() RETURNS trigger AS $$
BEGIN
  NEW.masjid_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjids ON masjids;
CREATE TRIGGER trg_set_updated_at_masjids
BEFORE UPDATE ON masjids
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_masjids();

-- 2) Sinkronisasi verifikasi → boolean flag + timestamp
CREATE OR REPLACE FUNCTION sync_masjid_verification_flags() RETURNS trigger AS $$
BEGIN
  IF NEW.masjid_verification_status = 'approved' THEN
    NEW.masjid_is_verified := TRUE;
    IF NEW.masjid_verified_at IS NULL THEN
      NEW.masjid_verified_at := now();
    END IF;
  ELSE
    -- untuk 'pending' dan 'rejected'
    NEW.masjid_is_verified := FALSE;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_verification ON masjids;
CREATE TRIGGER trg_sync_verification
BEFORE INSERT OR UPDATE ON masjids
FOR EACH ROW
EXECUTE FUNCTION sync_masjid_verification_flags();

-- 3) Image “Trash 30 hari”
--    - Saat masjid_image_url diganti/di-clear, masukkan URL lama ke trash + set due 30 hari
--    - Kalau user “restore” (mengisi kembali dengan nilai trash), kosongkan trash & due
CREATE OR REPLACE FUNCTION handle_masjid_image_trash() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'UPDATE' THEN
    -- Jika URL gambar berubah…
    IF NEW.masjid_image_url IS DISTINCT FROM OLD.masjid_image_url THEN
      -- CASE A: restore (user mengembalikan ke URL trash yang ada)
      IF OLD.masjid_image_trash_url IS NOT NULL
         AND NEW.masjid_image_url = OLD.masjid_image_trash_url THEN
        NEW.masjid_image_trash_url := NULL;
        NEW.masjid_image_delete_pending_until := NULL;

      -- CASE B: pindahkan gambar lama ke trash, jadwalkan auto-delete 30 hari
      ELSIF OLD.masjid_image_url IS NOT NULL
         AND (NEW.masjid_image_url IS DISTINCT FROM OLD.masjid_image_url) THEN
        NEW.masjid_image_trash_url := OLD.masjid_image_url;
        NEW.masjid_image_delete_pending_until := now() + INTERVAL '30 days';
      END IF;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_handle_masjid_image_trash ON masjids;
CREATE TRIGGER trg_handle_masjid_image_trash
BEFORE UPDATE ON masjids
FOR EACH ROW
EXECUTE FUNCTION handle_masjid_image_trash();

-- =========================================================
-- RELATIONSHIPS
-- =========================================================

-- USER_FOLLOW_MASJID
CREATE TABLE IF NOT EXISTS user_follow_masjid (
  follow_user_id   UUID NOT NULL,
  follow_masjid_id UUID NOT NULL,
  follow_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (follow_user_id, follow_masjid_id),
  FOREIGN KEY (follow_user_id)   REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (follow_masjid_id) REFERENCES masjids(masjid_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_follow_masjid_id ON user_follow_masjid (follow_masjid_id);
CREATE INDEX IF NOT EXISTS idx_follow_user_id   ON user_follow_masjid (follow_user_id);


-- =========================================================
-- TABLES
-- =========================================================
-- MASJIDS_PROFILES
CREATE TABLE IF NOT EXISTS masjids_profiles (
  masjid_profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_profile_description TEXT,
  masjid_profile_founded_year INT,
  masjid_profile_masjid_id UUID UNIQUE REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_profile_logo_url TEXT,
  masjid_profile_stamp_url TEXT,
  masjid_profile_ttd_ketua_dkm_url TEXT,
  masjid_profile_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  masjid_profile_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  masjid_profile_deleted_at TIMESTAMP
);
CREATE OR REPLACE FUNCTION set_updated_at_masjids_profiles() RETURNS trigger AS $$
BEGIN
  NEW.masjid_profile_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_set_updated_at_masjids_profiles ON masjids_profiles;
CREATE TRIGGER trg_set_updated_at_masjids_profiles
BEFORE UPDATE ON masjids_profiles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_masjids_profiles();
