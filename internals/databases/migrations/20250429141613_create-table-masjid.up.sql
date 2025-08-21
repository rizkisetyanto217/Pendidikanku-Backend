-- === Extensions yang dibutuhkan ===
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram text search
CREATE EXTENSION IF NOT EXISTS cube;          -- earthdistance requirement
CREATE EXTENSION IF NOT EXISTS earthdistance; -- distance by lat/lon

-- =========================================
-- 1) Tabel MASJIDS (inti)
-- =========================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_name VARCHAR(100) NOT NULL,
  masjid_bio_short TEXT,
  masjid_location TEXT,
  masjid_latitude  DECIMAL(9,6),
  masjid_longitude DECIMAL(9,6),

  masjid_image_url TEXT,
  masjid_google_maps_url TEXT,

  masjid_domain VARCHAR(50),                    -- opsional; unik case-insensitive via index fungsional
  masjid_slug   VARCHAR(100) UNIQUE NOT NULL,   -- unik untuk URL

  masjid_is_verified BOOLEAN DEFAULT FALSE,

  masjid_instagram_url TEXT,
  masjid_whatsapp_url  TEXT,
  masjid_youtube_url   TEXT,
  masjid_facebook_url  TEXT,
  masjid_tiktok_url    TEXT,
  masjid_whatsapp_group_ikhwan_url TEXT,
  masjid_whatsapp_group_akhwat_url TEXT,

  masjid_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  masjid_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  masjid_deleted_at TIMESTAMP
);

-- Validasi koordinat (tunda VALIDATE jika mau)
ALTER TABLE masjids
  ADD CONSTRAINT masjids_lat_chk CHECK (masjid_latitude  BETWEEN -90  AND 90)  NOT VALID,
  ADD CONSTRAINT masjids_lon_chk CHECK (masjid_longitude BETWEEN -180 AND 180) NOT VALID;

-- === Pencarian nama & lokasi ===
CREATE INDEX IF NOT EXISTS idx_masjids_name_trgm
  ON masjids USING gin (masjid_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_masjids_location_trgm
  ON masjids USING gin (masjid_location gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_masjids_name_lower
  ON masjids (LOWER(masjid_name));

-- Filter cepat by verifikasi / soft-delete
CREATE INDEX IF NOT EXISTS idx_masjids_verified
  ON masjids (masjid_is_verified);

CREATE INDEX IF NOT EXISTS idx_masjids_slug_alive
  ON masjids (masjid_slug)
  WHERE masjid_deleted_at IS NULL;

-- Unik domain case-insensitive (hanya baris yang domain-nya terisi)
CREATE UNIQUE INDEX IF NOT EXISTS idx_masjids_domain_ci_unique
  ON masjids (LOWER(masjid_domain))
  WHERE masjid_domain IS NOT NULL;

-- Full-text search gabungan (name + location + bio)
ALTER TABLE masjids
  ADD COLUMN IF NOT EXISTS masjid_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_name,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_bio_short,'')), 'C')
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_masjids_search
  ON masjids USING gin (masjid_search);

-- Geospasial (ORDER BY terdekat)
CREATE INDEX IF NOT EXISTS idx_masjids_earth
  ON masjids
  USING gist (ll_to_earth(masjid_latitude::float8, masjid_longitude::float8));

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at_masjids() RETURNS trigger AS $$
BEGIN
  NEW.masjid_updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjids ON masjids;
CREATE TRIGGER trg_set_updated_at_masjids
BEFORE UPDATE ON masjids
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_masjids();

-- =========================================
-- 2) Tabel USER_FOLLOW_MASJID (relasi mengikuti)
-- =========================================
CREATE TABLE IF NOT EXISTS user_follow_masjid (
  follow_user_id   UUID NOT NULL,
  follow_masjid_id UUID NOT NULL,
  follow_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (follow_user_id, follow_masjid_id),
  FOREIGN KEY (follow_user_id)   REFERENCES users(id)       ON DELETE CASCADE,
  FOREIGN KEY (follow_masjid_id) REFERENCES masjids(masjid_id) ON DELETE CASCADE
);

-- Index lookup dua arah
CREATE INDEX IF NOT EXISTS idx_follow_masjid_id ON user_follow_masjid (follow_masjid_id);
CREATE INDEX IF NOT EXISTS idx_follow_user_id   ON user_follow_masjid (follow_user_id);

-- =========================================
-- 3) Tabel MASJIDS_PROFILES (1:1 ke masjid)
-- =========================================
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

-- Trigger updated_at untuk profile
CREATE OR REPLACE FUNCTION set_updated_at_masjids_profiles() RETURNS trigger AS $$
BEGIN
  NEW.masjid_profile_updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjids_profiles ON masjids_profiles;
CREATE TRIGGER trg_set_updated_at_masjids_profiles
BEFORE UPDATE ON masjids_profiles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_masjids_profiles();
