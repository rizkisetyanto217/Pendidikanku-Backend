-- =========================================================
-- FULL UP MIGRATION (from scratch, NO ALTER TABLE)
-- Schools core + follows + public profiles
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
CREATE TABLE IF NOT EXISTS schools (
  school_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi
  school_yayasan_id       UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_current_plan_id  UUID REFERENCES school_service_plans (school_service_plan_id),
  school_verified_by_user_id  UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_created_by_user_id   UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_updated_by_user_id   UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Identitas & lokasi ringkas
  school_name      VARCHAR(100) NOT NULL,
  school_bio_short TEXT,
  school_location  TEXT,                    -- "Kota, Provinsi"
  school_city      VARCHAR(80),
  school_province  VARCHAR(80),

  school_timezone  VARCHAR(50),             -- ex: Asia/Jakarta

  -- Domain & slug
  school_domain VARCHAR(50),
  school_slug   VARCHAR(100) NOT NULL,

  -- Status & verifikasi
  school_is_active            BOOLEAN NOT NULL DEFAULT TRUE,
  school_is_verified          BOOLEAN NOT NULL DEFAULT FALSE,
  school_verification_status  verification_status_enum NOT NULL DEFAULT 'pending',
  school_verified_at          TIMESTAMPTZ,
  school_verification_notes   TEXT,


  -- Branding/SEO
  school_tagline              VARCHAR(120),
  school_logo_url             TEXT,
  school_banner_url           TEXT,
  school_profile_cover_url    TEXT,

  -- Kontak & admin
  school_official_email       CITEXT,
  school_official_phone       VARCHAR(30),
  school_contact_person_name  VARCHAR(100),
  school_contact_person_phone VARCHAR(30),

  school_domain_verified_at   TIMESTAMPTZ,

  -- Flag & levels
  school_is_islamic_school BOOLEAN NOT NULL DEFAULT FALSE,

  school_theme_preset_code VARCHAR(64),             -- refer ui_theme_presets.code
  school_theme_custom JSONB,                        -- { 


  -- Audit waktu & IP
  school_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_deleted_at TIMESTAMPTZ,
  school_last_activity_at     TIMESTAMPTZ,
  school_ip_created           INET,                 -- NEW
  school_ip_updated           INET,                 -- NEW

  -- Search (generated)
  school_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(school_name,'')), 'A')
    || setweight(to_tsvector('simple', array_to_string(coalesce(school_alt_names,'{}'::text[]),' ')), 'A')
    || setweight(to_tsvector('simple', coalesce(school_location,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(school_bio_short,'')), 'C')
    || setweight(to_tsvector('simple', coalesce(school_levels::text,'')), 'D')
  ) STORED,

  -- Validasi
  CONSTRAINT chk_school_levels_is_array
    CHECK (school_levels IS NULL OR jsonb_typeof(school_levels) = 'array'),
  CONSTRAINT chk_school_theme_custom_is_object
    CHECK (school_theme_custom IS NULL OR jsonb_typeof(school_theme_custom) = 'object'),
  CONSTRAINT chk_school_feature_flags_is_object
    CHECK (school_feature_flags IS NULL OR jsonb_typeof(school_feature_flags) = 'object'),
  CONSTRAINT chk_school_translations_is_object
    CHECK (school_translations IS NULL OR jsonb_typeof(school_translations) = 'object'),
  CONSTRAINT chk_school_external_ids_is_object
    CHECK (school_external_ids IS NULL OR jsonb_typeof(school_external_ids) = 'object'),
  CONSTRAINT chk_school_qibla_bearing_valid
    CHECK (school_qibla_bearing_deg IS NULL OR (school_qibla_bearing_deg >= 0 AND school_qibla_bearing_deg <= 360)),
  CONSTRAINT chk_school_capacity_nonneg
    CHECK (
      (school_capacity_men IS NULL OR school_capacity_men >= 0) AND
      (school_capacity_women IS NULL OR school_capacity_women >= 0)
    ),
  CONSTRAINT chk_school_donation_min_amount_nonneg
    CHECK (school_donation_min_amount IS NULL OR school_donation_min_amount >= 0)
);

-- ---------- Indexes: MASJIDS ----------
-- CI uniques
CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_domain_ci
  ON schools (LOWER(school_domain));
CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_slug_ci
  ON schools (LOWER(school_slug));

-- FTS & trigram
CREATE INDEX IF NOT EXISTS idx_schools_name_trgm
  ON schools USING gin (school_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_schools_location_trgm
  ON schools USING gin (school_location gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_schools_search
  ON schools USING gin (school_search);

-- FK helpers & attributes
CREATE INDEX IF NOT EXISTS idx_schools_yayasan
  ON schools (school_yayasan_id);
CREATE INDEX IF NOT EXISTS idx_schools_current_plan
  ON schools (school_current_plan_id);
CREATE INDEX IF NOT EXISTS idx_schools_verified_by
  ON schools (school_verified_by_user_id);
CREATE INDEX IF NOT EXISTS idx_schools_created_by
  ON schools (school_created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_schools_updated_by
  ON schools (school_updated_by_user_id);

-- JSONB GIN
CREATE INDEX IF NOT EXISTS gin_schools_levels
  ON schools USING gin (school_levels);
CREATE INDEX IF NOT EXISTS gin_schools_feature_flags
  ON schools USING gin (school_feature_flags);
CREATE INDEX IF NOT EXISTS gin_schools_theme_custom
  ON schools USING gin (school_theme_custom);
CREATE INDEX IF NOT EXISTS gin_schools_translations
  ON schools USING gin (school_translations);
CREATE INDEX IF NOT EXISTS gin_schools_external_ids
  ON schools USING gin (school_external_ids);

-- Operational & housekeeping
CREATE INDEX IF NOT EXISTS idx_schools_active_alive
  ON schools(school_is_active)
  WHERE school_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schools_billing_status
  ON schools (school_billing_status);
CREATE INDEX IF NOT EXISTS idx_schools_domain_dns_status
  ON schools (school_domain_dns_status);
CREATE INDEX IF NOT EXISTS idx_schools_country_code
  ON schools (school_country_code);
CREATE INDEX IF NOT EXISTS idx_schools_is_flagged
  ON schools (school_is_flagged);
CREATE INDEX IF NOT EXISTS brin_schools_created_at
  ON schools USING brin (school_created_at);
CREATE INDEX IF NOT EXISTS brin_schools_last_activity_at
  ON schools USING brin (school_last_activity_at);
CREATE INDEX IF NOT EXISTS brin_schools_last_indexed_at
  ON schools USING brin (school_last_indexed_at);
CREATE INDEX IF NOT EXISTS brin_schools_last_engagement_at
  ON schools USING brin (school_last_engagement_at);

-- Emails (CI)
CREATE INDEX IF NOT EXISTS idx_schools_official_email_lower
  ON schools (LOWER(school_official_email));

-- =========================================================
-- USER_FOLLOW_MASJID (relasi + preferensi notifikasi)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_follow_school (
  user_follow_school_user_id    UUID NOT NULL,
  user_follow_school_school_id  UUID NOT NULL,
  user_follow_school_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- preferensi & housekeeping
  user_follow_school_notify_new_post BOOLEAN NOT NULL DEFAULT TRUE,
  user_follow_school_notify_event    BOOLEAN NOT NULL DEFAULT TRUE,
  user_follow_school_digest_frequency VARCHAR(10),             -- off|daily|weekly
  user_follow_school_mute_until       TIMESTAMPTZ,
  user_follow_school_last_notified_at TIMESTAMPTZ,
  user_follow_school_source           VARCHAR(20),             -- web|android|ios|import
  user_follow_school_tags             TEXT[],                  -- minat user

  CONSTRAINT pk_user_follow_school
    PRIMARY KEY (user_follow_school_user_id, user_follow_school_school_id),
  CONSTRAINT fk_user_follow_school_user
    FOREIGN KEY (user_follow_school_user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_user_follow_school_school
    FOREIGN KEY (user_follow_school_school_id) REFERENCES schools(school_id) ON DELETE CASCADE
);

-- Indexes: USER_FOLLOW_MASJID
CREATE INDEX IF NOT EXISTS idx_user_follow_school_user_id
  ON user_follow_school (user_follow_school_user_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_school_id
  ON user_follow_school (user_follow_school_school_id);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_created_at
  ON user_follow_school (user_follow_school_created_at);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_digest_freq
  ON user_follow_school (user_follow_school_digest_frequency);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_mute_until
  ON user_follow_school (user_follow_school_mute_until);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_last_notified_at
  ON user_follow_school (user_follow_school_last_notified_at);
CREATE INDEX IF NOT EXISTS idx_user_follow_school_source
  ON user_follow_school (user_follow_school_source);

-- =========================================================
-- MASJIDS_PROFILES (profil publik + sekolah) + extras
-- =========================================================
CREATE TABLE IF NOT EXISTS school_profiles (
  school_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi 1:1 ke school
  school_profile_school_id UUID NOT NULL UNIQUE REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Deskripsi & sejarah
  school_profile_description  TEXT,
  school_profile_about_short  VARCHAR(280),
  school_profile_visi         TEXT,
  school_profile_misi         TEXT,
  school_profile_founded_year INT,

  -- Alamat & kontak publik
  school_profile_address       TEXT,
  school_profile_contact_phone VARCHAR(30),
  school_profile_contact_email_alt VARCHAR(120),

  -- Lokasi detail & geospasial
  school_profile_google_place_id VARCHAR(64),
  school_profile_postal_code     VARCHAR(20),
  school_profile_geo_admin       JSONB,         -- {kelurahan, kecamatan, kab_kota, provinsi}

  -- Profil sekolah (opsional)
  school_profile_school_npsn              VARCHAR(20),
  school_profile_school_nss               VARCHAR(20),
  school_profile_school_accreditation     VARCHAR(10),
  school_profile_school_principal_user_id UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_profile_school_phone             VARCHAR(30),
  school_profile_school_email             VARCHAR(120),
  school_profile_school_address           TEXT,
  school_profile_school_student_capacity  INT,
  school_profile_school_is_boarding       BOOLEAN NOT NULL DEFAULT FALSE,

  -- Donasi & media
  school_profile_donation_bank_name    VARCHAR(60),
  school_profile_donation_account_no   VARCHAR(60),
  school_profile_donation_account_name VARCHAR(120),
  school_profile_donation_min_amount   BIGINT,
  school_profile_donation_url          TEXT,
  school_profile_qris_image_url        TEXT,

  -- Branding tambahan (opsional)
  school_profile_logo_url       TEXT,
  school_profile_stempel_url    TEXT,
  school_profile_ttd_ketua_url  TEXT,

  -- Konten, layanan, legal, analitik
  school_profile_services        JSONB,     -- katalog layanan (filter)
  school_profile_photo_gallery_count INT,

  -- Search (generated)
  school_profile_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(school_profile_description,'')), 'A')
    || setweight(to_tsvector('simple', coalesce(school_profile_visi,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(school_profile_misi,'')), 'B')
    || setweight(to_tsvector('simple', coalesce(school_profile_address, school_profile_school_address, '')), 'C')
    || setweight(
         to_tsvector('simple',
           coalesce(school_profile_instagram_url,'') || ' ' ||
           coalesce(school_profile_youtube_url,'')   || ' ' ||
           coalesce(school_profile_facebook_url,'')  || ' ' ||
           coalesce(school_profile_tiktok_url,'')    || ' ' ||
           coalesce(school_profile_website_url,'')   || ' ' ||
           coalesce(school_profile_google_maps_url,'')
         ), 'D'
       )
  ) STORED,

  -- Audit & IP
  school_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_profile_deleted_at TIMESTAMPTZ,
  school_profile_ip_created INET,                -- NEW
  school_profile_ip_updated INET,                -- NEW

  -- Validasi
  CONSTRAINT chk_mpp_founded_year
    CHECK (school_profile_founded_year IS NULL OR school_profile_founded_year BETWEEN 1800 AND EXTRACT(YEAR FROM now())::int),
  CONSTRAINT chk_mpp_latlon_pair
    CHECK (
      (school_profile_latitude IS NULL AND school_profile_longitude IS NULL)
      OR (school_profile_latitude BETWEEN -90 AND 90 AND school_profile_longitude BETWEEN -180 AND 180)
    ),
  CONSTRAINT chk_mpp_contact_email
    CHECK (
      school_profile_contact_email IS NULL
      OR school_profile_contact_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
    ),
  CONSTRAINT chk_mpp_school_email
    CHECK (
      school_profile_school_email IS NULL
      OR school_profile_school_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
    ),
  CONSTRAINT chk_mpp_student_capacity
    CHECK (school_profile_school_student_capacity IS NULL OR school_profile_school_student_capacity >= 0),
  CONSTRAINT chk_mpp_school_accreditation
    CHECK (school_profile_school_accreditation IS NULL OR school_profile_school_accreditation IN ('A','B','C','Ungraded','-')),
  CONSTRAINT chk_mpp_opening_hours_is_object
    CHECK (school_profile_opening_hours IS NULL OR jsonb_typeof(school_profile_opening_hours) = 'object'),
  CONSTRAINT chk_mpp_facilities_is_object
    CHECK (school_profile_facilities IS NULL OR jsonb_typeof(school_profile_facilities) = 'object'),
  CONSTRAINT chk_mpp_services_is_object
    CHECK (school_profile_services IS NULL OR jsonb_typeof(school_profile_services) = 'object'),
  CONSTRAINT chk_mpp_geo_admin_is_object
    CHECK (school_profile_geo_admin IS NULL OR jsonb_typeof(school_profile_geo_admin) = 'object'),
  CONSTRAINT chk_mpp_translations_is_object
    CHECK (school_profile_translations IS NULL OR jsonb_typeof(school_profile_translations) = 'object'),
  CONSTRAINT chk_mpp_legal_docs_is_object
    CHECK (school_profile_legal_docs IS NULL OR jsonb_typeof(school_profile_legal_docs) = 'object'),
  CONSTRAINT chk_mpp_counts_nonneg
    CHECK (
      (school_profile_worshipper_capacity IS NULL OR school_profile_worshipper_capacity >= 0) AND
      (school_profile_parking_capacity   IS NULL OR school_profile_parking_capacity   >= 0) AND
      (school_profile_wudu_spots        IS NULL OR school_profile_wudu_spots        >= 0) AND
      (school_profile_restrooms         IS NULL OR school_profile_restrooms         >= 0) AND
      (school_profile_photo_gallery_count IS NULL OR school_profile_photo_gallery_count >= 0) AND
      (school_profile_page_views        IS NULL OR school_profile_page_views >= 0)
    )
);

-- ---------- Indexes: MASJIDS_PROFILES ----------
CREATE INDEX IF NOT EXISTS idx_mpp_school_id
  ON school_profiles (school_profile_school_id);
CREATE INDEX IF NOT EXISTS idx_mpp_principal_user_id
  ON school_profiles (school_profile_school_principal_user_id);

-- Emails (CI)
CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_lower
  ON school_profiles (LOWER(school_profile_contact_email));
CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_alt_lower
  ON school_profiles (LOWER(school_profile_contact_email_alt));
CREATE INDEX IF NOT EXISTS idx_mpp_school_email_lower
  ON school_profiles (LOWER(school_profile_school_email));

-- Attribute indexes
CREATE INDEX IF NOT EXISTS idx_mpp_accreditation
  ON school_profiles (school_profile_school_accreditation);
CREATE INDEX IF NOT EXISTS idx_mpp_founded_year
  ON school_profiles (school_profile_founded_year);
CREATE INDEX IF NOT EXISTS idx_mpp_is_boarding
  ON school_profiles (school_profile_school_is_boarding);
CREATE INDEX IF NOT EXISTS idx_mpp_geohash
  ON school_profiles (school_profile_geohash);

-- Geospasial nearest-neighbor
CREATE INDEX IF NOT EXISTS idx_mpp_earth
  ON school_profiles USING gist (
    ll_to_earth(school_profile_latitude::float8, school_profile_longitude::float8)
  );

-- JSONB GIN
CREATE INDEX IF NOT EXISTS gin_mpp_opening_hours
  ON school_profiles USING gin (school_profile_opening_hours);
CREATE INDEX IF NOT EXISTS gin_mpp_facilities
  ON school_profiles USING gin (school_profile_facilities);
CREATE INDEX IF NOT EXISTS gin_mpp_services
  ON school_profiles USING gin (school_profile_services);
CREATE INDEX IF NOT EXISTS gin_mpp_geo_admin
  ON school_profiles USING gin (school_profile_geo_admin);
CREATE INDEX IF NOT EXISTS gin_mpp_translations
  ON school_profiles USING gin (school_profile_translations);
CREATE INDEX IF NOT EXISTS gin_mpp_social_handles
  ON school_profiles USING gin (school_profile_social_handles);
CREATE INDEX IF NOT EXISTS gin_mpp_legal_docs
  ON school_profiles USING gin (school_profile_legal_docs);

-- FTS & arsip waktu
CREATE INDEX IF NOT EXISTS idx_mpp_search
  ON school_profiles USING gin (school_profile_search);
CREATE INDEX IF NOT EXISTS brin_mpp_created_at
  ON school_profiles USING brin (school_profile_created_at);

-- Unik NPSN/NSS bila diisi
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_npsn
  ON school_profiles (school_profile_school_npsn);
CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_nss
  ON school_profiles (school_profile_school_nss);

COMMIT;
