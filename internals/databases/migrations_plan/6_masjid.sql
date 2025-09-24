-- =========================================================
-- FULL UP MIGRATION (from scratch, NO ALTER TABLE)
-- Masjids core + follows + public profiles
-- + extra columns: audit/legal, integrations, analytics, security, l10n
-- Idempotent & production-ready
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;        -- CITEXT type
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram indexes
CREATE EXTENSION IF NOT EXISTS cube;          -- for earthdistance
CREATE EXTENSION IF NOT EXISTS earthdistance; -- ll_to_earth()

-- ---------- ENUMS ----------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END$$;

-- =========================================================
-- MASJIDS (inti/operasional)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi
  masjid_yayasan_id       UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_current_plan_id  UUID REFERENCES masjid_service_plans (masjid_service_plan_id),
  masjid_verified_by_user_id  UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_created_by_user_id   UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_updated_by_user_id   UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Identitas & lokasi ringkas
  masjid_name      VARCHAR(100) NOT NULL,
  masjid_bio_short TEXT,
  masjid_location  TEXT,                    -- "Kota, Provinsi"
  masjid_city      VARCHAR(80),
  masjid_province  VARCHAR(80),

  masjid_timezone  VARCHAR(50),             -- ex: Asia/Jakarta

  -- Domain & slug
  masjid_domain VARCHAR(50),
  masjid_slug   VARCHAR(100) NOT NULL,

  -- Status & verifikasi
  masjid_is_active            BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_is_verified          BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_verification_status  verification_status_enum NOT NULL DEFAULT 'pending',
  masjid_verified_at          TIMESTAMPTZ,
  masjid_verification_notes   TEXT,


  -- Branding/SEO
  masjid_tagline              VARCHAR(120),
  masjid_logo_url             TEXT,
  masjid_banner_url           TEXT,
  masjid_profile_cover_url    TEXT,

  -- Kontak & admin
  masjid_official_email       CITEXT,
  masjid_official_phone       VARCHAR(30),
  masjid_contact_person_name  VARCHAR(100),
  masjid_contact_person_phone VARCHAR(30),

  masjid_domain_verified_at   TIMESTAMPTZ,

  -- Flag & levels
  masjid_is_islamic_school BOOLEAN NOT NULL DEFAULT FALSE,

  masjid_theme_preset_code VARCHAR(64),             -- refer ui_theme_presets.code
  masjid_theme_custom JSONB,                        -- { 


  -- Audit waktu & IP
  masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_deleted_at TIMESTAMPTZ,
  masjid_last_activity_at     TIMESTAMPTZ,
  masjid_ip_created           INET,                 -- NEW
  masjid_ip_updated           INET,                 -- NEW

  -- Search (generated)
  masjid_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_name,'')), 'A')
    || setweight(to_tsvector('simple', array_to_string(coalesce(masjid_alt_names,'{}'::text[]),' ')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_bio_short,'')), 'C')
    || setweight(to_tsvector('simple', coalesce(masjid_levels::text,'')), 'D')
  ) STORED,

  -- Validasi
  CONSTRAINT chk_masjid_levels_is_array
    CHECK (masjid_levels IS NULL OR jsonb_typeof(masjid_levels) = 'array'),
  CONSTRAINT chk_masjid_theme_custom_is_object
    CHECK (masjid_theme_custom IS NULL OR jsonb_typeof(masjid_theme_custom) = 'object'),
  CONSTRAINT chk_masjid_feature_flags_is_object
    CHECK (masjid_feature_flags IS NULL OR jsonb_typeof(masjid_feature_flags) = 'object'),
  CONSTRAINT chk_masjid_translations_is_object
    CHECK (masjid_translations IS NULL OR jsonb_typeof(masjid_translations) = 'object'),
  CONSTRAINT chk_masjid_external_ids_is_object
    CHECK (masjid_external_ids IS NULL OR jsonb_typeof(masjid_external_ids) = 'object'),
  CONSTRAINT chk_masjid_qibla_bearing_valid
    CHECK (masjid_qibla_bearing_deg IS NULL OR (masjid_qibla_bearing_deg >= 0 AND masjid_qibla_bearing_deg <= 360)),
  CONSTRAINT chk_masjid_capacity_nonneg
    CHECK (
      (masjid_capacity_men IS NULL OR masjid_capacity_men >= 0) AND
      (masjid_capacity_women IS NULL OR masjid_capacity_women >= 0)
    ),
  CONSTRAINT chk_masjid_donation_min_amount_nonneg
    CHECK (masjid_donation_min_amount IS NULL OR masjid_donation_min_amount >= 0)
);

-- ---------- Indexes: MASJIDS ----------
-- CI uniques
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_domain_ci
  ON masjids (LOWER(masjid_domain));
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_slug_ci
  ON masjids (LOWER(masjid_slug));

-- FTS & trigram
CREATE INDEX IF NOT EXISTS idx_masjids_name_trgm
  ON masjids USING gin (masjid_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_masjids_location_trgm
  ON masjids USING gin (masjid_location gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_masjids_search
  ON masjids USING gin (masjid_search);

-- FK helpers & attributes
CREATE INDEX IF NOT EXISTS idx_masjids_yayasan
  ON masjids (masjid_yayasan_id);
CREATE INDEX IF NOT EXISTS idx_masjids_current_plan
  ON masjids (masjid_current_plan_id);
CREATE INDEX IF NOT EXISTS idx_masjids_verified_by
  ON masjids (masjid_verified_by_user_id);
CREATE INDEX IF NOT EXISTS idx_masjids_created_by
  ON masjids (masjid_created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_masjids_updated_by
  ON masjids (masjid_updated_by_user_id);

-- JSONB GIN
CREATE INDEX IF NOT EXISTS gin_masjids_levels
  ON masjids USING gin (masjid_levels);
CREATE INDEX IF NOT EXISTS gin_masjids_feature_flags
  ON masjids USING gin (masjid_feature_flags);
CREATE INDEX IF NOT EXISTS gin_masjids_theme_custom
  ON masjids USING gin (masjid_theme_custom);
CREATE INDEX IF NOT EXISTS gin_masjids_translations
  ON masjids USING gin (masjid_translations);
CREATE INDEX IF NOT EXISTS gin_masjids_external_ids
  ON masjids USING gin (masjid_external_ids);

-- Operational & housekeeping
CREATE INDEX IF NOT EXISTS idx_masjids_active_alive
  ON masjids(masjid_is_active)
  WHERE masjid_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_masjids_billing_status
  ON masjids (masjid_billing_status);
CREATE INDEX IF NOT EXISTS idx_masjids_domain_dns_status
  ON masjids (masjid_domain_dns_status);
CREATE INDEX IF NOT EXISTS idx_masjids_country_code
  ON masjids (masjid_country_code);
CREATE INDEX IF NOT EXISTS idx_masjids_is_flagged
  ON masjids (masjid_is_flagged);
CREATE INDEX IF NOT EXISTS brin_masjids_created_at
  ON masjids USING brin (masjid_created_at);
CREATE INDEX IF NOT EXISTS brin_masjids_last_activity_at
  ON masjids USING brin (masjid_last_activity_at);
CREATE INDEX IF NOT EXISTS brin_masjids_last_indexed_at
  ON masjids USING brin (masjid_last_indexed_at);
CREATE INDEX IF NOT EXISTS brin_masjids_last_engagement_at
  ON masjids USING brin (masjid_last_engagement_at);

-- Emails (CI)
CREATE INDEX IF NOT EXISTS idx_masjids_official_email_lower
  ON masjids (LOWER(masjid_official_email));

-- =========================================================
-- USER_FOLLOW_MASJID (relasi + preferensi notifikasi)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_follow_masjid (
  user_follow_masjid_user_id    UUID NOT NULL,
  user_follow_masjid_masjid_id  UUID NOT NULL,
  user_follow_masjid_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- preferensi & housekeeping
  user_follow_masjid_notify_new_post BOOLEAN NOT NULL DEFAULT TRUE,
  user_follow_masjid_notify_event    BOOLEAN NOT NULL DEFAULT TRUE,
  user_follow_masjid_digest_frequency VARCHAR(10),             -- off|daily|weekly
  user_follow_masjid_mute_until       TIMESTAMPTZ,
  user_follow_masjid_last_notified_at TIMESTAMPTZ,
  user_follow_masjid_source           VARCHAR(20),             -- web|android|ios|import
  user_follow_masjid_tags             TEXT[],                  -- minat user

  CONSTRAINT pk_user_follow_masjid
    PRIMARY KEY (user_follow_masjid_user_id, user_follow_masjid_masjid_id),
  CONSTRAINT fk_user_follow_masjid_user
    FOREIGN KEY (user_follow_masjid_user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_user_follow_masjid_masjid
    FOREIGN KEY (user_follow_masjid_masjid_id) REFERENCES masjids(masjid_id) ON DELETE CASCADE
);

-- Indexes: USER_FOLLOW_MASJID
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_user_id
  ON user_follow_masjid (user_follow_masjid_user_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_masjid_id
  ON user_follow_masjid (user_follow_masjid_masjid_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_created_at
  ON user_follow_masjid (user_follow_masjid_created_at);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_digest_freq
  ON user_follow_masjid (user_follow_masjid_digest_frequency);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_mute_until
  ON user_follow_masjid (user_follow_masjid_mute_until);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_last_notified_at
  ON user_follow_masjid (user_follow_masjid_last_notified_at);
CREATE INDEX IF NOT EXISTS idx_user_follow_masjid_source
  ON user_follow_masjid (user_follow_masjid_source);

-- =========================================================
-- MASJIDS_PROFILES (profil publik + sekolah) + extras
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_profiles (
  masjid_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi 1:1 ke masjid
  masjid_profile_masjid_id UUID NOT NULL UNIQUE REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Deskripsi & sejarah
  masjid_profile_description  TEXT,
  masjid_profile_about_short  VARCHAR(280),
  masjid_profile_visi         TEXT,
  masjid_profile_misi         TEXT,
  masjid_profile_founded_year INT,

  -- Alamat & kontak publik
  masjid_profile_address       TEXT,
  masjid_profile_contact_phone VARCHAR(30),
  masjid_profile_contact_email_alt VARCHAR(120),

  -- Lokasi detail & geospasial
  masjid_profile_google_place_id VARCHAR(64),
  masjid_profile_postal_code     VARCHAR(20),
  masjid_profile_geo_admin       JSONB,         -- {kelurahan, kecamatan, kab_kota, provinsi}

  -- Profil sekolah (opsional)
  masjid_profile_school_npsn              VARCHAR(20),
  masjid_profile_school_nss               VARCHAR(20),
  masjid_profile_school_accreditation     VARCHAR(10),
  masjid_profile_school_principal_user_id UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_profile_school_phone             VARCHAR(30),
  masjid_profile_school_email             VARCHAR(120),
  masjid_profile_school_address           TEXT,
  masjid_profile_school_student_capacity  INT,
  masjid_profile_school_is_boarding       BOOLEAN NOT NULL DEFAULT FALSE,

  -- Donasi & media
  masjid_profile_donation_bank_name    VARCHAR(60),
  masjid_profile_donation_account_no   VARCHAR(60),
  masjid_profile_donation_account_name VARCHAR(120),
  masjid_profile_donation_min_amount   BIGINT,
  masjid_profile_donation_url          TEXT,
  masjid_profile_qris_image_url        TEXT,

  -- Branding tambahan (opsional)
  masjid_profile_logo_url       TEXT,
  masjid_profile_stempel_url    TEXT,
  masjid_profile_ttd_ketua_url  TEXT,

  -- Konten, layanan, legal, analitik
  masjid_profile_services        JSONB,     -- katalog layanan (filter)
  masjid_profile_photo_gallery_count INT,

  -- Search (generated)
  masjid_profile_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(masjid_profile_description,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(masjid_profile_visi,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_profile_misi,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(masjid_profile_address, masjid_profile_school_address, '')), 'C')
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
  ) STORED,

  -- Audit & IP
  masjid_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_deleted_at TIMESTAMPTZ,
  masjid_profile_ip_created INET,                -- NEW
  masjid_profile_ip_updated INET,                -- NEW

  -- Validasi
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
    CHECK (masjid_profile_school_accreditation IS NULL OR masjid_profile_school_accreditation IN ('A','B','C','Ungraded','-')),
  CONSTRAINT chk_mpp_opening_hours_is_object
    CHECK (masjid_profile_opening_hours IS NULL OR jsonb_typeof(masjid_profile_opening_hours) = 'object'),
  CONSTRAINT chk_mpp_facilities_is_object
    CHECK (masjid_profile_facilities IS NULL OR jsonb_typeof(masjid_profile_facilities) = 'object'),
  CONSTRAINT chk_mpp_services_is_object
    CHECK (masjid_profile_services IS NULL OR jsonb_typeof(masjid_profile_services) = 'object'),
  CONSTRAINT chk_mpp_geo_admin_is_object
    CHECK (masjid_profile_geo_admin IS NULL OR jsonb_typeof(masjid_profile_geo_admin) = 'object'),
  CONSTRAINT chk_mpp_translations_is_object
    CHECK (masjid_profile_translations IS NULL OR jsonb_typeof(masjid_profile_translations) = 'object'),
  CONSTRAINT chk_mpp_legal_docs_is_object
    CHECK (masjid_profile_legal_docs IS NULL OR jsonb_typeof(masjid_profile_legal_docs) = 'object'),
  CONSTRAINT chk_mpp_counts_nonneg
    CHECK (
      (masjid_profile_worshipper_capacity IS NULL OR masjid_profile_worshipper_capacity >= 0) AND
      (masjid_profile_parking_capacity   IS NULL OR masjid_profile_parking_capacity   >= 0) AND
      (masjid_profile_wudu_spots        IS NULL OR masjid_profile_wudu_spots        >= 0) AND
      (masjid_profile_restrooms         IS NULL OR masjid_profile_restrooms         >= 0) AND
      (masjid_profile_photo_gallery_count IS NULL OR masjid_profile_photo_gallery_count >= 0) AND
      (masjid_profile_page_views        IS NULL OR masjid_profile_page_views >= 0)
    )
);

-- ---------- Indexes: MASJIDS_PROFILES ----------
CREATE INDEX IF NOT EXISTS idx_mpp_masjid_id
  ON masjid_profiles (masjid_profile_masjid_id);
CREATE INDEX IF NOT EXISTS idx_mpp_principal_user_id
  ON masjid_profiles (masjid_profile_school_principal_user_id);

-- Emails (CI)
CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_lower
  ON masjid_profiles (LOWER(masjid_profile_contact_email));
CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_alt_lower
  ON masjid_profiles (LOWER(masjid_profile_contact_email_alt));
CREATE INDEX IF NOT EXISTS idx_mpp_school_email_lower
  ON masjid_profiles (LOWER(masjid_profile_school_email));

-- Attribute indexes
CREATE INDEX IF NOT EXISTS idx_mpp_accreditation
  ON masjid_profiles (masjid_profile_school_accreditation);
CREATE INDEX IF NOT EXISTS idx_mpp_founded_year
  ON masjid_profiles (masjid_profile_founded_year);
CREATE INDEX IF NOT EXISTS idx_mpp_is_boarding
  ON masjid_profiles (masjid_profile_school_is_boarding);
CREATE INDEX IF NOT EXISTS idx_mpp_geohash
  ON masjid_profiles (masjid_profile_geohash);

-- Geospasial nearest-neighbor
CREATE INDEX IF NOT EXISTS idx_mpp_earth
  ON masjid_profiles USING gist (
    ll_to_earth(masjid_profile_latitude::float8, masjid_profile_longitude::float8)
  );

-- JSONB GIN
CREATE INDEX IF NOT EXISTS gin_mpp_opening_hours
  ON masjid_profiles USING gin (masjid_profile_opening_hours);
CREATE INDEX IF NOT EXISTS gin_mpp_facilities
  ON masjid_profiles USING gin (masjid_profile_facilities);
CREATE INDEX IF NOT EXISTS gin_mpp_services
  ON masjid_profiles USING gin (masjid_profile_services);
CREATE INDEX IF NOT EXISTS gin_mpp_geo_admin
  ON masjid_profiles USING gin (masjid_profile_geo_admin);
CREATE INDEX IF NOT EXISTS gin_mpp_translations
  ON masjid_profiles USING gin (masjid_profile_translations);
CREATE INDEX IF NOT EXISTS gin_mpp_social_handles
  ON masjid_profiles USING gin (masjid_profile_social_handles);
CREATE INDEX IF NOT EXISTS gin_mpp_legal_docs
  ON masjid_profiles USING gin (masjid_profile_legal_docs);

-- FTS & arsip waktu
CREATE INDEX IF NOT EXISTS idx_mpp_search
  ON masjid_profiles USING gin (masjid_profile_search);
CREATE INDEX IF NOT EXISTS brin_mpp_created_at
  ON masjid_profiles USING brin (masjid_profile_created_at);

-- Unik NPSN/NSS bila diisi
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_npsn
  ON masjid_profiles (masjid_profile_school_npsn);
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_nss
  ON masjid_profiles (masjid_profile_school_nss);

COMMIT;
