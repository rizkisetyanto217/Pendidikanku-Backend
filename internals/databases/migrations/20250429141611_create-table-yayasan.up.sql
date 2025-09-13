-- =========================================================
-- TABEL YAYASANS (FOUNDATION)
-- =========================================================
-- ===== Prasyarat: extensions yg dipakai index =====
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;


-- ===== Enum verifikasi (pending|approved|rejected) =====
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum'
  ) THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending','approved','rejected');
  END IF;
END$$;


CREATE TABLE IF NOT EXISTS yayasans (
  yayasan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Identitas & legal
  yayasan_name        VARCHAR(150) NOT NULL,
  yayasan_legal_number TEXT,                 -- nomor akta/pendirian/penyesuaian
  yayasan_legal_date   DATE,                 -- tanggal akta
  yayasan_npwp         VARCHAR(32),          -- opsional: NPWP yayasan

  -- Kontak & lokasi
  yayasan_address   TEXT,
  yayasan_city      TEXT,
  yayasan_province  TEXT,
  yayasan_latitude  DECIMAL(9,6),
  yayasan_longitude DECIMAL(9,6),

  -- Media & maps
  yayasan_logo_url              TEXT,
  yayasan_logo_trash_url        TEXT,        -- URL lama saat diganti (masuk trash)
  yayasan_logo_delete_pending_until TIMESTAMP, -- auto-delete 30 hari
  yayasan_google_maps_url       TEXT,

  -- Domain & slug
  yayasan_domain VARCHAR(80),
  yayasan_slug   VARCHAR(120) UNIQUE NOT NULL,

  -- Status & verifikasi
  yayasan_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  yayasan_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  yayasan_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  yayasan_verified_at TIMESTAMPTZ NULL,
  yayasan_verification_notes TEXT,

  -- Sosial
  yayasan_website_url  TEXT,
  yayasan_instagram_url TEXT,
  yayasan_whatsapp_url  TEXT,
  yayasan_youtube_url   TEXT,
  yayasan_facebook_url  TEXT,
  yayasan_tiktok_url    TEXT,

  -- Full-text search gabungan (name + alamat + kota/provinsi)
  yayasan_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(yayasan_name,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(yayasan_address,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(yayasan_city,'')), 'C')
    || setweight(to_tsvector('simple', coalesce(yayasan_province,'')), 'C')
  ) STORED,

  -- Audit
  yayasan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_deleted_at TIMESTAMPTZ,

  -- Validasi koordinat
  CONSTRAINT yayasans_lat_chk CHECK (yayasan_latitude  BETWEEN -90  AND 90),
  CONSTRAINT yayasans_lon_chk CHECK (yayasan_longitude BETWEEN -180 AND 180)
);

-- =========================================================
-- INDEXES
-- =========================================================

-- Trigram search untuk name & city
CREATE INDEX IF NOT EXISTS idx_yayasans_name_trgm
  ON yayasans USING gin (yayasan_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_yayasans_city_trgm
  ON yayasans USING gin (yayasan_city gin_trgm_ops);

-- Filter case-insensitive by name
CREATE INDEX IF NOT EXISTS idx_yayasans_name_lower
  ON yayasans (LOWER(yayasan_name));

-- Domain unik case-insensitive (hanya yang set)
CREATE UNIQUE INDEX IF NOT EXISTS idx_yayasans_domain_ci_unique
  ON yayasans (LOWER(yayasan_domain))
  WHERE yayasan_domain IS NOT NULL;

-- Flag & status (hanya row “alive”)
CREATE INDEX IF NOT EXISTS idx_yayasans_active
  ON yayasans (yayasan_is_active) WHERE yayasan_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_yayasans_verified
  ON yayasans (yayasan_is_verified) WHERE yayasan_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_yayasans_verif_status
  ON yayasans (yayasan_verification_status) WHERE yayasan_deleted_at IS NULL;

-- Slug cepat untuk row “alive”
CREATE INDEX IF NOT EXISTS idx_yayasans_slug_alive
  ON yayasans (yayasan_slug) WHERE yayasan_deleted_at IS NULL;

-- Full-text search gabungan
CREATE INDEX IF NOT EXISTS idx_yayasans_search
  ON yayasans USING gin (yayasan_search);

-- Geospatial nearest-neighbor
CREATE INDEX IF NOT EXISTS idx_yayasans_earth
  ON yayasans USING gist (ll_to_earth(yayasan_latitude::float8, yayasan_longitude::float8));

-- GC logo: pilih yang sudah due (trash tidak kosong dan due sudah lewat)
CREATE INDEX IF NOT EXISTS idx_yayasans_logo_gc_due
  ON yayasans (yayasan_logo_delete_pending_until)
  WHERE yayasan_logo_trash_url IS NOT NULL;

-- =========================================================
-- TRIGGERS
-- =========================================================

-- 2) Sinkronisasi verifikasi → boolean flag + timestamptz
CREATE OR REPLACE FUNCTION sync_yayasan_verification_flags() RETURNS trigger AS $$
BEGIN
  IF NEW.yayasan_verification_status = 'approved' THEN
    NEW.yayasan_is_verified := TRUE;
    IF NEW.yayasan_verified_at IS NULL THEN
      NEW.yayasan_verified_at := now();
    END IF;
  ELSE
    NEW.yayasan_is_verified := FALSE;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_yayasan_verification ON yayasans;
CREATE TRIGGER trg_sync_yayasan_verification
BEFORE INSERT OR UPDATE ON yayasans
FOR EACH ROW
EXECUTE FUNCTION sync_yayasan_verification_flags();
