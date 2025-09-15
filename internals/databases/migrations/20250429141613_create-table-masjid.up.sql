-- =========================================================
-- =====================   U P   ===========================
-- =========================================================

-- PRASYARAT
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram search
CREATE EXTENSION IF NOT EXISTS cube;          -- earthdistance requirement
CREATE EXTENSION IF NOT EXISTS earthdistance; -- ll_to_earth()

-- ENUM verifikasi
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END$$;

-- =========================================================
-- MASJIDS (inti/operasional) - TANPA KOORDINAT
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi
  masjid_yayasan_id     UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_current_plan_id UUID REFERENCES masjid_service_plans (masjid_service_plan_id),

  -- Identitas & lokasi ringkas
  masjid_name      VARCHAR(100) NOT NULL,
  masjid_bio_short TEXT,
  masjid_location  TEXT,                -- ringkas: "Kota, Provinsi"
  masjid_city      VARCHAR(80),

  -- Domain & slug
  masjid_domain VARCHAR(50),
  masjid_slug   VARCHAR(100) NOT NULL UNIQUE,

  -- Status & verifikasi
  masjid_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  masjid_verified_at TIMESTAMPTZ,
  masjid_verification_notes TEXT,

  -- Kontak & admin
  masjid_contact_person_name  VARCHAR(100),
  masjid_contact_person_phone VARCHAR(30),

  -- Flag
  masjid_is_islamic_school BOOLEAN NOT NULL DEFAULT FALSE,

  -- Levels (tag-style; contoh: ["Kursus","Ilmu Qur'an","Sekolah Nonformal"])
  masjid_levels JSONB,

  -- Full-text search (GENERATED; tanpa trigger)
  masjid_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_name,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_bio_short,'')), 'C')
    || setweight(to_tsvector('simple', coalesce(masjid_levels::text,'')), 'D')
  ) STORED,

  -- Audit
  masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
   masjid_last_activity_at     TIMESTAMPTZ,
  masjid_deleted_at TIMESTAMPTZ,

  -- Validasi ringan
  CONSTRAINT chk_masjid_levels_is_array
    CHECK (masjid_levels IS NULL OR jsonb_typeof(masjid_levels) = 'array')
);

-- INDEXES (masjids)
CREATE INDEX IF NOT EXISTS idx_masjids_name_trgm
  ON masjids USING gin (masjid_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_masjids_location_trgm
  ON masjids USING gin (masjid_location gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_masjids_name_lower
  ON masjids (LOWER(masjid_name));

-- Domain unik CI (tanpa WHERE/partial)
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_domain_ci
  ON masjids (LOWER(masjid_domain));

-- FK helpers
CREATE INDEX IF NOT EXISTS idx_masjids_yayasan
  ON masjids (masjid_yayasan_id);
CREATE INDEX IF NOT EXISTS idx_masjids_current_plan
  ON masjids (masjid_current_plan_id);

-- Levels (JSONB) => operator ?, ?|, ?&, @>
CREATE INDEX IF NOT EXISTS gin_masjids_levels
  ON masjids USING gin (masjid_levels);

-- FTS & arsip waktu
CREATE INDEX IF NOT EXISTS idx_masjids_search
  ON masjids USING gin (masjid_search);
CREATE INDEX IF NOT EXISTS brin_masjids_created_at
  ON masjids USING brin (masjid_created_at);

-- Status
CREATE INDEX IF NOT EXISTS idx_masjids_active_alive
  ON masjids(masjid_is_active)
  WHERE masjid_deleted_at IS NULL;


-- =========================================================
-- USER_FOLLOW_MASJID (relasi sederhana)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_follow_masjid (
  user_follow_masjid_user_id    UUID NOT NULL,
  user_follow_masjid_masjid_id  UUID NOT NULL,
  user_follow_masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT pk_user_follow_masjid
    PRIMARY KEY (user_follow_masjid_user_id, user_follow_masjid_masjid_id),
  CONSTRAINT fk_user_follow_masjid_user
    FOREIGN KEY (user_follow_masjid_user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_user_follow_masjid_masjid
    FOREIGN KEY (user_follow_masjid_masjid_id) REFERENCES masjids(masjid_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_user_id
  ON user_follow_masjid (user_follow_masjid_user_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_masjid_id
  ON user_follow_masjid (user_follow_masjid_masjid_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_created_at
  ON user_follow_masjid (user_follow_masjid_created_at);



-- =========================================================
-- TABEL: masjids_profiles (tanpa school_type)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids_profiles (
  masjid_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi 1:1 ke masjid
  masjid_profile_masjid_id UUID NOT NULL UNIQUE REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Deskripsi & sejarah
  masjid_profile_description  TEXT,
  masjid_profile_founded_year INT,

  -- Alamat & kontak publik
  masjid_profile_address       TEXT,
  masjid_profile_contact_phone VARCHAR(30),
  masjid_profile_contact_email VARCHAR(120),

  -- Sosial/link publik (termasuk Google Maps)
  masjid_profile_google_maps_url           TEXT,
  masjid_profile_instagram_url             TEXT,
  masjid_profile_whatsapp_url              TEXT,
  masjid_profile_youtube_url               TEXT,
  masjid_profile_facebook_url              TEXT,
  masjid_profile_tiktok_url                TEXT,
  masjid_profile_whatsapp_group_ikhwan_url TEXT,
  masjid_profile_whatsapp_group_akhwat_url TEXT,
  masjid_profile_website_url               TEXT,

  -- Koordinat (di profiles)
  masjid_profile_latitude  DECIMAL(9,6),
  masjid_profile_longitude DECIMAL(9,6),

  -- Profil sekolah (opsional) — TANPA school_type
  masjid_profile_school_npsn              VARCHAR(20),
  masjid_profile_school_nss               VARCHAR(20),
  masjid_profile_school_accreditation     VARCHAR(10),
  masjid_profile_school_principal_user_id UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_profile_school_phone             VARCHAR(30),
  masjid_profile_school_email             VARCHAR(120),
  masjid_profile_school_address           TEXT,
  masjid_profile_school_student_capacity  INT,
  masjid_profile_school_is_boarding       BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit
  masjid_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_deleted_at TIMESTAMPTZ,

  -- Validasi ringan
  CONSTRAINT chk_mpp_founded_year
    CHECK (masjid_profile_founded_year IS NULL OR masjid_profile_founded_year BETWEEN 1800 AND EXTRACT(YEAR FROM now())::int),
  CONSTRAINT chk_mpp_latlon_pair
    CHECK (
      (masjid_profile_latitude IS NULL AND masjid_profile_longitude IS NULL)
      OR (masjid_profile_latitude BETWEEN -90 AND 90 AND masjid_profile_longitude BETWEEN -180 AND 180)
    ),
  CONSTRAINT chk_mpp_contact_email
  CHECK (
    masjid_profile_contact_email IS NULL
    OR masjid_profile_contact_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
  ),
  CONSTRAINT chk_mpp_school_email
  CHECK (
    masjid_profile_school_email IS NULL
    OR masjid_profile_school_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
  ),
  CONSTRAINT chk_mpp_student_capacity
    CHECK (masjid_profile_school_student_capacity IS NULL OR masjid_profile_school_student_capacity >= 0),
  CONSTRAINT chk_mpp_school_accreditation
    CHECK (masjid_profile_school_accreditation IS NULL OR masjid_profile_school_accreditation IN ('A','B','C','Ungraded','-'))
);

-- =========================================================
-- (RE)CREATE KOLOM FTS TANPA school_type
-- =========================================================
ALTER TABLE masjids_profiles
  ADD COLUMN IF NOT EXISTS masjid_profile_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_profile_description,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_profile_address, masjid_profile_school_address, '')), 'B')
    || setweight(
         to_tsvector('simple',
           coalesce(masjid_profile_instagram_url,'') || ' ' ||
           coalesce(masjid_profile_youtube_url,'')   || ' ' ||
           coalesce(masjid_profile_facebook_url,'')  || ' ' ||
           coalesce(masjid_profile_tiktok_url,'')    || ' ' ||
           coalesce(masjid_profile_website_url,'')   || ' ' ||
           coalesce(masjid_profile_google_maps_url,'')
         ), 'D'
       )
  ) STORED;

-- =========================================================
-- INDEXES
-- =========================================================
-- Lookups dasar
CREATE INDEX IF NOT EXISTS idx_mpp_masjid_id
  ON masjids_profiles (masjid_profile_masjid_id);
CREATE INDEX IF NOT EXISTS idx_mpp_principal_user_id
  ON masjids_profiles (masjid_profile_school_principal_user_id);

-- Email & atribut ringan
CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_lower
  ON masjids_profiles (LOWER(masjid_profile_contact_email));
CREATE INDEX IF NOT EXISTS idx_mpp_school_email_lower
  ON masjids_profiles (LOWER(masjid_profile_school_email));
CREATE INDEX IF NOT EXISTS idx_mpp_accreditation
  ON masjids_profiles (masjid_profile_school_accreditation);
CREATE INDEX IF NOT EXISTS idx_mpp_founded_year
  ON masjids_profiles (masjid_profile_founded_year);
CREATE INDEX IF NOT EXISTS idx_mpp_is_boarding
  ON masjids_profiles (masjid_profile_school_is_boarding);

-- Geospasial (nearest-neighbor) via earthdistance
CREATE INDEX IF NOT EXISTS idx_mpp_earth
  ON masjids_profiles USING gist (
    ll_to_earth(masjid_profile_latitude::float8, masjid_profile_longitude::float8)
  );

-- Full-text search & arsip waktu
CREATE INDEX IF NOT EXISTS idx_mpp_search
  ON masjids_profiles USING gin (masjid_profile_search);
CREATE INDEX IF NOT EXISTS brin_mpp_created_at
  ON masjids_profiles USING brin (masjid_profile_created_at);

-- Unik NPSN/NSS bila diisi (Postgres mengizinkan multiple NULL)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_npsn
  ON masjids_profiles (masjid_profile_school_npsn);
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_nss
  ON masjids_profiles (masjid_profile_school_nss);
