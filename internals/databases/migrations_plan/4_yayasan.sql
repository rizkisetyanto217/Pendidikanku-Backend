-- =========================================================
-- UP — TABEL YAYASANS (FOUNDATION) — KOLOM SAJA (FINAL + DOC URLS)
-- =========================================================

-- Prasyarat minimal
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;    -- CITEXT

-- Enum status verifikasi
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending','approved','rejected');
  END IF;
END$$;

-- =========================================================
-- CREATE TABLE (KOLOM LENGKAP, TANPA INDEX/TRIGGER)
-- =========================================================
CREATE TABLE IF NOT EXISTS yayasans (
  yayasan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Identitas & Legal
  yayasan_name                   VARCHAR(150) NOT NULL,
  yayasan_alias_name             VARCHAR(100),
  yayasan_legal_number           TEXT,
  yayasan_legal_date             DATE,
  yayasan_registration_number    VARCHAR(100),

  yayasan_founded_year           SMALLINT,


  -- Alamat & Lokasi (granular)
  yayasan_address                TEXT,
  yayasan_city                   TEXT,
  yayasan_province               TEXT,
  yayasan_district_kecamatan     VARCHAR(100),
  yayasan_subdistrict_kelurahan  VARCHAR(100),
  yayasan_postal_code            VARCHAR(10),
  yayasan_province_code          VARCHAR(10),
  yayasan_city_code              VARCHAR(10),
  yayasan_plus_code              VARCHAR(20),    -- Google Plus Code
  yayasan_location_source        VARCHAR(20),    -- 'manual'|'geocoded'
  yayasan_latitude               DECIMAL(9,6),
  yayasan_longitude              DECIMAL(9,6),
  yayasan_google_maps_url        TEXT,

  -- Domain & Slug
  yayasan_domain                 VARCHAR(80),
  yayasan_slug                   VARCHAR(120) NOT NULL,

  -- Status & Verifikasi
  yayasan_is_active              BOOLEAN NOT NULL DEFAULT TRUE,
  yayasan_is_verified            BOOLEAN NOT NULL DEFAULT FALSE,
  yayasan_verification_status    verification_status_enum NOT NULL DEFAULT 'pending',
  yayasan_verified_at            TIMESTAMPTZ,
  yayasan_verification_notes     TEXT,
  yayasan_verified_document_url  TEXT,
  yayasan_last_reviewed_at       TIMESTAMPTZ,

  -- Sosial Media
  yayasan_website_url            TEXT,
  yayasan_instagram_url          TEXT,
  yayasan_whatsapp_url           TEXT,
  yayasan_youtube_url            TEXT,
  yayasan_facebook_url           TEXT,
  yayasan_tiktok_url             TEXT,

  -- Kontak
  yayasan_email                  CITEXT,
  yayasan_phone_number           VARCHAR(32),
  yayasan_whatsapp_number        VARCHAR(20),

  -- Hierarki / Organisasi
  yayasan_short_alias            VARCHAR(30),    -- alias pendek untuk URL/QR

  -- Governance & Organisasi
  yayasan_board_structure        JSONB,          -- struktur pengurus
  yayasan_focus_area             TEXT[],         -- {"pendidikan","sosial","kesehatan"}
  yayasan_mission                TEXT,
  yayasan_vision                 TEXT,
  yayasan_values                 TEXT[],
  yayasan_awards                 JSONB,

  -- Operasional
  yayasan_operating_hours        JSONB,          -- { "mon":[["08:00","16:00"]], ... }
  yayasan_service_area           JSONB,          -- { "radius_km": 10, "cities": ["..."] }


  -- Security & Tagging
  yayasan_tags                   TEXT[],

  -- Preferensi & Privasi
  yayasan_description_short      VARCHAR(200),
  yayasan_description_long       TEXT,

  yayasan_internal_notes         TEXT,




  -- Audit waktu
  yayasan_created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_deleted_at             TIMESTAMPTZ,

  -- Validasi koordinat sederhana
  CONSTRAINT yayasans_lat_chk CHECK (yayasan_latitude  BETWEEN -90  AND 90),
  CONSTRAINT yayasans_lon_chk CHECK (yayasan_longitude BETWEEN -180 AND 180)
);
