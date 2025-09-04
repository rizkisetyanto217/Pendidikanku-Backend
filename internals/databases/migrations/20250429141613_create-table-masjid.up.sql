-- =========================================================
-- PRASYARAT EXTENSIONS
-- =========================================================
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
-- TABEL MASJIDS (FRESH CREATE, TANPA URL)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi Yayasan
  masjid_yayasan_id UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Identitas & lokasi
  masjid_name       VARCHAR(100) NOT NULL,
  masjid_bio_short  TEXT,
  masjid_location   TEXT,
  masjid_latitude   DECIMAL(9,6),
  masjid_longitude  DECIMAL(9,6),

  -- Domain & slug
  masjid_domain VARCHAR(50),
  masjid_slug   VARCHAR(100) UNIQUE NOT NULL,

  -- Status & verifikasi
  masjid_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  masjid_verified_at TIMESTAMPTZ NULL,
  masjid_verification_notes TEXT,

  -- Paket aktif
  masjid_current_plan_id UUID REFERENCES masjid_service_plans (masjid_service_plan_id),

  -- Flag sekolah/pesantren
  masjid_is_islamic_school BOOLEAN NOT NULL DEFAULT FALSE,

  -- Full-text search gabungan (name+location+bio)
  masjid_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_name,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_bio_short,'')), 'C')
  ) STORED,

  -- Audit
  masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_deleted_at TIMESTAMPTZ,

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

-- Slug cepat untuk row “alive”
CREATE INDEX IF NOT EXISTS idx_masjids_slug_alive
  ON masjids (masjid_slug) WHERE masjid_deleted_at IS NULL;

-- Full-text search gabungan
CREATE INDEX IF NOT EXISTS idx_masjids_search
  ON masjids USING gin (masjid_search);

-- Geospatial nearest-neighbor
CREATE INDEX IF NOT EXISTS idx_masjids_earth
  ON masjids USING gist (ll_to_earth(masjid_latitude::float8, masjid_longitude::float8));

-- Index relasi yayasan
CREATE INDEX IF NOT EXISTS idx_masjids_yayasan
  ON masjids (masjid_yayasan_id);

-- =========================================================
-- TRIGGER: Sinkronisasi verifikasi (HANYA INI YANG DIPERTAHANKAN)
-- =========================================================
CREATE OR REPLACE FUNCTION sync_masjid_verification_flags() RETURNS trigger AS $$
BEGIN
  IF NEW.masjid_verification_status = 'approved' THEN
    NEW.masjid_is_verified := TRUE;
    IF NEW.masjid_verified_at IS NULL THEN
      NEW.masjid_verified_at := now();
    END IF;
  ELSE
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

-- =========================================================
-- RELATIONSHIPS
-- =========================================================

-- USER_FOLLOW_MASJID
CREATE TABLE IF NOT EXISTS user_follow_masjid (
  user_follow_masjid_user_id    UUID        NOT NULL,
  user_follow_masjid_masjid_id  UUID        NOT NULL,
  user_follow_masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT pk_user_follow_masjid
    PRIMARY KEY (user_follow_masjid_user_id, user_follow_masjid_masjid_id),

  CONSTRAINT fk_user_follow_masjid_user
    FOREIGN KEY (user_follow_masjid_user_id)
    REFERENCES users(id)
    ON DELETE CASCADE,

  CONSTRAINT fk_user_follow_masjid_masjid
    FOREIGN KEY (user_follow_masjid_masjid_id)
    REFERENCES masjids(masjid_id)
    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_user_id
  ON user_follow_masjid (user_follow_masjid_user_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_masjid_id
  ON user_follow_masjid (user_follow_masjid_masjid_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_created_at
  ON user_follow_masjid (user_follow_masjid_masjid_id, user_follow_masjid_created_at DESC);

-- =========================================================
-- MASJIDS_PROFILES
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids_profiles (
  masjid_profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_profile_description TEXT,
  masjid_profile_founded_year INT,
  masjid_profile_masjid_id UUID UNIQUE REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_profile_logo_url TEXT,
  masjid_profile_stamp_url TEXT,
  masjid_profile_ttd_ketua_dkm_url TEXT,
  masjid_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_deleted_at TIMESTAMPTZ
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

-- Selesai