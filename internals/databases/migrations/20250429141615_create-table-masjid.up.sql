-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram search
CREATE EXTENSION IF NOT EXISTS cube;          -- earthdistance requirement
CREATE EXTENSION IF NOT EXISTS earthdistance; -- ll_to_earth()

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status_enum') THEN
    CREATE TYPE verification_status_enum AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tenant_profile_enum') THEN
    CREATE TYPE tenant_profile_enum AS ENUM (
      'student',
      'teacher_solo',
      'teacher_plus',
      'school_basic',
      'school_plus'
    );
  END IF;
END$$;

-- =========================================================
-- TABLE: MASJIDS
-- =========================================================
CREATE TABLE IF NOT EXISTS masjids (
  masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi
  masjid_yayasan_id       UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_current_plan_id  UUID,

  -- Identitas & lokasi ringkas
  masjid_name      VARCHAR(100) NOT NULL,
  masjid_bio_short TEXT,
  masjid_location  TEXT,
  masjid_city      VARCHAR(80),

  -- Domain & slug
  masjid_domain VARCHAR(50),
  masjid_slug   VARCHAR(100) NOT NULL,

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

  -- Peruntukan tenant
  masjid_tenant_profile tenant_profile_enum NOT NULL DEFAULT 'school_basic',

  -- Levels (tag array)
  masjid_levels JSONB,

  -- Teacher invite/join code
  masjid_teacher_code_hash BYTEA,
  masjid_teacher_code_set_at TIMESTAMPTZ,

  -- Media: icon (2-slot + retensi)
  masjid_icon_url                  TEXT,
  masjid_icon_object_key           TEXT,
  masjid_icon_url_old              TEXT,
  masjid_icon_object_key_old       TEXT,
  masjid_icon_delete_pending_until TIMESTAMPTZ,

  -- Media: logo (2-slot + retensi)
  masjid_logo_url                  TEXT,
  masjid_logo_object_key           TEXT,
  masjid_logo_url_old              TEXT,
  masjid_logo_object_key_old       TEXT,
  masjid_logo_delete_pending_until TIMESTAMPTZ,

  -- Media: background (2-slot + retensi)
  masjid_background_url                  TEXT,
  masjid_background_object_key           TEXT,
  masjid_background_url_old              TEXT,
  masjid_background_object_key_old       TEXT,
  masjid_background_delete_pending_until TIMESTAMPTZ,

  -- Audit
  masjid_created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_last_activity_at TIMESTAMPTZ,
  masjid_deleted_at       TIMESTAMPTZ,

  -- Checks
  CONSTRAINT chk_masjid_levels_is_array
    CHECK (masjid_levels IS NULL OR jsonb_typeof(masjid_levels) = 'array'),
  CONSTRAINT chk_masjid_contact_phone
    CHECK (masjid_contact_person_phone IS NULL OR masjid_contact_person_phone ~ '^\+?[0-9]{7,20}$'),
  CONSTRAINT chk_masjid_slug_format
    CHECK (masjid_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  CONSTRAINT chk_masjid_domain_format
    CHECK (masjid_domain IS NULL OR masjid_domain ~* '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z]{2,})+$')
);

-- FK plan (idempotent, toleran delete)
ALTER TABLE masjids
  DROP CONSTRAINT IF EXISTS masjids_masjid_current_plan_id_fkey;
ALTER TABLE masjids
  ADD CONSTRAINT masjids_masjid_current_plan_id_fkey
  FOREIGN KEY (masjid_current_plan_id)
  REFERENCES masjid_service_plans (masjid_service_plan_id)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Bersih-bersih FTS lama (jika pernah ada)
DROP INDEX IF EXISTS idx_masjids_search;
ALTER TABLE masjids DROP COLUMN IF EXISTS masjid_search;

-- =========================================================
-- INDEXES: MASJIDS
-- =========================================================
CREATE INDEX IF NOT EXISTS idx_masjids_name_trgm
  ON masjids USING gin (masjid_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_masjids_location_trgm
  ON masjids USING gin (masjid_location gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_masjids_name_lower
  ON masjids (LOWER(masjid_name));

CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_domain_ci
  ON masjids (LOWER(masjid_domain));

-- Drop unique lama di kolom slug (jika ada)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'masjids'::regclass
      AND contype = 'u'
      AND conname = 'masjids_masjid_slug_key'
  ) THEN
    EXECUTE 'ALTER TABLE masjids DROP CONSTRAINT masjids_masjid_slug_key';
  END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_slug_ci
  ON masjids (LOWER(masjid_slug));

CREATE INDEX IF NOT EXISTS idx_masjids_slug_lower
  ON masjids (LOWER(masjid_slug));

CREATE INDEX IF NOT EXISTS idx_masjids_yayasan
  ON masjids (masjid_yayasan_id);

CREATE INDEX IF NOT EXISTS idx_masjids_current_plan
  ON masjids (masjid_current_plan_id);

CREATE INDEX IF NOT EXISTS gin_masjids_levels
  ON masjids USING gin (masjid_levels);

CREATE INDEX IF NOT EXISTS brin_masjids_created_at
  ON masjids USING brin (masjid_created_at);

CREATE INDEX IF NOT EXISTS idx_masjids_active_alive
  ON masjids(masjid_is_active)
  WHERE masjid_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_masjids_tenant_profile
  ON masjids (masjid_tenant_profile);

CREATE INDEX IF NOT EXISTS brin_masjids_icon_delete_pending_until
  ON masjids USING brin (masjid_icon_delete_pending_until);

CREATE INDEX IF NOT EXISTS brin_masjids_logo_delete_pending_until
  ON masjids USING brin (masjid_logo_delete_pending_until);

CREATE INDEX IF NOT EXISTS brin_masjids_background_delete_pending_until
  ON masjids USING brin (masjid_background_delete_pending_until);

CREATE INDEX IF NOT EXISTS idx_masjids_city_alive
  ON masjids (masjid_city)
  WHERE masjid_deleted_at IS NULL;

-- =========================================================
-- TRIGGERS: updated_at sinkron dengan DB
-- =========================================================
CREATE OR REPLACE FUNCTION set_masjid_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.masjid_updated_at := now();
  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'trg_masjids_set_updated_at'
  ) THEN
    CREATE TRIGGER trg_masjids_set_updated_at
    BEFORE UPDATE ON masjids
    FOR EACH ROW
    EXECUTE FUNCTION set_masjid_updated_at();
  END IF;
END$$;

CREATE OR REPLACE FUNCTION sync_masjid_is_verified()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.masjid_is_verified := (NEW.masjid_verification_status = 'approved');
  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'trg_masjids_sync_is_verified'
  ) THEN
    CREATE TRIGGER trg_masjids_sync_is_verified
    BEFORE INSERT OR UPDATE ON masjids
    FOR EACH ROW
    EXECUTE FUNCTION sync_masjid_is_verified();
  END IF;
END$$;

-- =====================================================================
-- MASJID PROFILES (1:1 ke masjids)
-- =====================================================================
CREATE TABLE IF NOT EXISTS masjid_profiles (
  masjid_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi 1:1 ke masjid
  masjid_profile_masjid_id UUID NOT NULL UNIQUE
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Deskripsi & sejarah
  masjid_profile_description  TEXT,
  masjid_profile_founded_year INT,

  -- Alamat & kontak publik
  masjid_profile_address       TEXT,
  masjid_profile_contact_phone VARCHAR(30),
  masjid_profile_contact_email VARCHAR(120),

  -- Sosial/link publik
  masjid_profile_google_maps_url           TEXT,
  masjid_profile_instagram_url             TEXT,
  masjid_profile_whatsapp_url              TEXT,
  masjid_profile_youtube_url               TEXT,
  masjid_profile_facebook_url              TEXT,
  masjid_profile_tiktok_url                TEXT,
  masjid_profile_whatsapp_group_ikhwan_url TEXT,
  masjid_profile_whatsapp_group_akhwat_url TEXT,
  masjid_profile_website_url               TEXT,

  -- Profil sekolah (opsional)—tanpa school_type
  masjid_profile_school_npsn              VARCHAR(20),
  masjid_profile_school_nss               VARCHAR(20),
  masjid_profile_school_accreditation     VARCHAR(10),
  masjid_profile_school_principal_user_id UUID
    REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  masjid_profile_school_student_capacity  INT,
  masjid_profile_school_is_boarding       BOOLEAN NOT NULL DEFAULT FALSE,

  -- Lokasi koordinat
  masjid_profile_latitude  DOUBLE PRECISION,
  masjid_profile_longitude DOUBLE PRECISION,

  -- Atribut tambahan
  masjid_profile_school_email   VARCHAR(120),
  masjid_profile_school_address TEXT,

  -- Audit
  masjid_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_profile_deleted_at TIMESTAMPTZ,

  -- Checks
  CONSTRAINT chk_mpp_founded_year CHECK (
    masjid_profile_founded_year IS NULL
    OR masjid_profile_founded_year BETWEEN 1800 AND EXTRACT(YEAR FROM now())::int
  ),
  CONSTRAINT chk_mpp_latlon_pair CHECK (
    (masjid_profile_latitude IS NULL AND masjid_profile_longitude IS NULL)
    OR (masjid_profile_latitude BETWEEN -90 AND 90 AND masjid_profile_longitude BETWEEN -180 AND 180)
  ),
  CONSTRAINT chk_mpp_contact_email CHECK (
    masjid_profile_contact_email IS NULL
    OR masjid_profile_contact_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
  ),
  CONSTRAINT chk_mpp_school_email CHECK (
    masjid_profile_school_email IS NULL
    OR masjid_profile_school_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
  ),
  CONSTRAINT chk_mpp_student_capacity CHECK (
    masjid_profile_school_student_capacity IS NULL
    OR masjid_profile_school_student_capacity >= 0
  ),
  CONSTRAINT chk_mpp_school_accreditation CHECK (
    masjid_profile_school_accreditation IS NULL
    OR masjid_profile_school_accreditation IN ('A','B','C','Ungraded','-')
  ),
  CONSTRAINT chk_mpp_phone CHECK (
    masjid_profile_contact_phone IS NULL
    OR masjid_profile_contact_phone ~ '^\+?[0-9]{7,20}$'
  )
);

-- =========================
-- INDEXES: MASJID PROFILES
-- =========================
CREATE INDEX IF NOT EXISTS idx_mpp_principal_user_id_alive
  ON masjid_profiles (masjid_profile_school_principal_user_id)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_contact_email_lower_alive
  ON masjid_profiles (LOWER(masjid_profile_contact_email))
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_school_email_lower_alive
  ON masjid_profiles (LOWER(masjid_profile_school_email))
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_accreditation_alive
  ON masjid_profiles (masjid_profile_school_accreditation)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_founded_year_alive
  ON masjid_profiles (masjid_profile_founded_year)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_is_boarding_alive
  ON masjid_profiles (masjid_profile_school_is_boarding)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gist_mpp_earth_alive
  ON masjid_profiles
  USING gist (ll_to_earth(masjid_profile_latitude::float8, masjid_profile_longitude::float8))
  WHERE masjid_profile_deleted_at IS NULL
    AND masjid_profile_latitude IS NOT NULL
    AND masjid_profile_longitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS trgm_mpp_address_alive
  ON masjid_profiles
  USING gin (masjid_profile_address gin_trgm_ops)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS trgm_mpp_description_alive
  ON masjid_profiles
  USING gin (masjid_profile_description gin_trgm_ops)
  WHERE masjid_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_mpp_created_at
  ON masjid_profiles USING brin (masjid_profile_created_at);

CREATE INDEX IF NOT EXISTS brin_mpp_updated_at
  ON masjid_profiles USING brin (masjid_profile_updated_at);

CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_npsn_alive
  ON masjid_profiles (masjid_profile_school_npsn)
  WHERE masjid_profile_deleted_at IS NULL
    AND masjid_profile_school_npsn IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_nss_alive
  ON masjid_profiles (masjid_profile_school_nss)
  WHERE masjid_profile_deleted_at IS NULL
    AND masjid_profile_school_nss IS NOT NULL;
