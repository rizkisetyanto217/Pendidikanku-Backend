-- =========================================================
-- UP Migration — TABEL YAYASANS (FOUNDATION) CLEAN (v2)
-- =========================================================

-- ===== Prasyarat: extensions =====
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;      -- trigram index
CREATE EXTENSION IF NOT EXISTS cube;         -- earthdistance dep
CREATE EXTENSION IF NOT EXISTS earthdistance;

-- ===== Enum verifikasi =====
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending','approved','rejected');
  END IF;
END$$;

-- =========================================================
-- TABLE yayasans
-- =========================================================
CREATE TABLE IF NOT EXISTS yayasans (
  yayasan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Identitas
  yayasan_name        VARCHAR(150) NOT NULL,
  yayasan_description TEXT,
  yayasan_bio         TEXT,

  -- Kontak & lokasi
  yayasan_address   TEXT,
  yayasan_city      TEXT,
  yayasan_province  TEXT,
  
  -- Media & maps
  yayasan_google_maps_url TEXT,

  -- Logo (single file, 2-slot + retensi 30 hari)
  yayasan_logo_url                   TEXT,
  yayasan_logo_object_key            TEXT,
  yayasan_logo_url_old               TEXT,
  yayasan_logo_object_key_old        TEXT,
  yayasan_logo_delete_pending_until  TIMESTAMPTZ,

  -- Domain & slug
  yayasan_domain VARCHAR(80),
  yayasan_slug   VARCHAR(120) UNIQUE NOT NULL,

  -- Status & verifikasi
  yayasan_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  yayasan_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  yayasan_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  yayasan_verified_at TIMESTAMPTZ NULL,
  yayasan_verification_notes TEXT,

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
  yayasan_deleted_at TIMESTAMPTZ
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

-- Flag & status (row “alive” saja)
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

-- GC logo: pilih yang sudah due (old url tidak kosong dan due sudah lewat)
CREATE INDEX IF NOT EXISTS idx_yayasans_logo_gc_due
  ON yayasans (yayasan_logo_delete_pending_until)
  WHERE yayasan_logo_url_old IS NOT NULL;
