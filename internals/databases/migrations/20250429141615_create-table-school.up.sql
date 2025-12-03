-- +migrate Up
BEGIN;

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

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_entry_mode_enum') THEN
    CREATE TYPE attendance_entry_mode_enum AS ENUM ('teacher_only', 'student_only', 'both');
  END IF;
END$$;

-- =========================================================
-- SEQUENCE: SCHOOL NUMBER
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_class
    WHERE relkind = 'S'
      AND relname = 'schools_school_number_seq'
  ) THEN
    CREATE SEQUENCE schools_school_number_seq;
  END IF;
END$$;

-- =========================================================
-- TABLE: SCHOOLS
-- =========================================================
CREATE TABLE IF NOT EXISTS schools (
  school_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Running number (auto-increment, global per table)
  school_number BIGINT NOT NULL DEFAULT nextval('schools_school_number_seq'::regclass),

  -- Relations
  school_yayasan_id       UUID REFERENCES yayasans (yayasan_id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_current_plan_id  UUID REFERENCES school_service_plans (school_service_plan_id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Identity & short location
  school_name      VARCHAR(100) NOT NULL,
  school_bio_short TEXT,
  school_location  TEXT,
  school_city      VARCHAR(80),

  -- Domain & slug
  school_domain VARCHAR(50),
  school_slug   VARCHAR(100) NOT NULL,

  -- Status & verification
  school_is_active           BOOLEAN NOT NULL DEFAULT TRUE,
  school_is_verified         BOOLEAN NOT NULL DEFAULT FALSE,
  school_verification_status verification_status_enum NOT NULL DEFAULT 'pending',
  school_verified_at         TIMESTAMPTZ,
  school_verification_notes  TEXT,

  -- Contact & admin
  school_contact_person_name  VARCHAR(100),
  school_contact_person_phone VARCHAR(30),

  -- Flag
  school_is_islamic_school BOOLEAN NOT NULL DEFAULT FALSE,

  -- Tenant profile
  school_tenant_profile tenant_profile_enum NOT NULL DEFAULT 'school_basic',

  -- Levels (tag array)
  school_levels JSONB,

  -- Teacher invite/join code
  school_teacher_code_hash   BYTEA,
  school_teacher_code_set_at TIMESTAMPTZ,

  -- Media: icon (2-slot + retention)
  school_icon_url                  TEXT,
  school_icon_object_key           TEXT,
  school_icon_url_old              TEXT,
  school_icon_object_key_old       TEXT,
  school_icon_delete_pending_until TIMESTAMPTZ,

  -- Media: logo (2-slot + retention)
  school_logo_url                  TEXT,
  school_logo_object_key           TEXT,
  school_logo_url_old              TEXT,
  school_logo_object_key_old       TEXT,
  school_logo_delete_pending_until TIMESTAMPTZ,

  -- Media: background (2-slot + retention)
  school_background_url                  TEXT,
  school_background_object_key           TEXT,
  school_background_url_old              TEXT,
  school_background_object_key_old       TEXT,
  school_background_delete_pending_until TIMESTAMPTZ,

  -- Default attendance mode: teacher_only / student_only / both
  school_default_attendance_entry_mode attendance_entry_mode_enum
    NOT NULL DEFAULT 'both',

  -- Global school settings
  school_timezone                  VARCHAR(50),
  school_default_min_passing_score INT,

  -- 🆕 Default number of students per class (school-wide setting)
  school_default_class_qouta       INT,

  school_settings JSONB NOT NULL DEFAULT '{}'::jsonb,

  -- Audit
  school_created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_last_activity_at TIMESTAMPTZ,
  school_deleted_at       TIMESTAMPTZ,

  -- Checks
  CONSTRAINT chk_school_levels_is_array
    CHECK (school_levels IS NULL OR jsonb_typeof(school_levels) = 'array'),

  CONSTRAINT chk_school_contact_phone
    CHECK (school_contact_person_phone IS NULL OR school_contact_person_phone ~ '^\+?[0-9]{7,20}$'),

  CONSTRAINT chk_school_slug_format
    CHECK (school_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),

  CONSTRAINT chk_school_domain_format
    CHECK (
      school_domain IS NULL
      OR school_domain ~* '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z]{2,})+$'
    ),

  CONSTRAINT chk_school_default_min_passing_score
    CHECK (
      school_default_min_passing_score IS NULL
      OR school_default_min_passing_score BETWEEN 0 AND 100
    ),

  -- 🆕 reasonable bound for default class quota
  CONSTRAINT chk_school_default_class_qouta
    CHECK (
      school_default_class_qouta IS NULL
      OR school_default_class_qouta BETWEEN 0 AND 1000
    ),

  CONSTRAINT chk_school_settings_is_object
    CHECK (
      school_settings IS NULL
      OR jsonb_typeof(school_settings) = 'object'
    )
);

-- =========================================================
-- INDEXES: SCHOOLS
-- =========================================================

-- Running number unique
CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_school_number
  ON schools (school_number);

CREATE INDEX IF NOT EXISTS idx_schools_name_trgm
  ON schools USING gin (school_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schools_location_trgm
  ON schools USING gin (school_location gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schools_name_lower
  ON schools (LOWER(school_name));

CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_domain_ci
  ON schools (LOWER(school_domain));

CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_slug_ci
  ON schools (LOWER(school_slug));

CREATE INDEX IF NOT EXISTS idx_schools_slug_lower
  ON schools (LOWER(school_slug));

CREATE INDEX IF NOT EXISTS idx_schools_yayasan
  ON schools (school_yayasan_id);

CREATE INDEX IF NOT EXISTS idx_schools_current_plan
  ON schools (school_current_plan_id);

CREATE INDEX IF NOT EXISTS gin_schools_levels
  ON schools USING gin (school_levels);

CREATE INDEX IF NOT EXISTS brin_schools_created_at
  ON schools USING brin (school_created_at);

CREATE INDEX IF NOT EXISTS idx_schools_active_alive
  ON schools (school_is_active)
  WHERE school_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_schools_tenant_profile
  ON schools (school_tenant_profile);

CREATE INDEX IF NOT EXISTS brin_schools_icon_delete_pending_until
  ON schools USING brin (school_icon_delete_pending_until);

CREATE INDEX IF NOT EXISTS brin_schools_logo_delete_pending_until
  ON schools USING brin (school_logo_delete_pending_until);

CREATE INDEX IF NOT EXISTS brin_schools_background_delete_pending_until
  ON schools USING brin (school_background_delete_pending_until);

CREATE INDEX IF NOT EXISTS idx_schools_city_alive
  ON schools (school_city)
  WHERE school_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_schools_default_attendance_entry_mode_alive
  ON schools (school_default_attendance_entry_mode)
  WHERE school_deleted_at IS NULL;

-- =========================================================
-- TRIGGERS: updated_at & is_verified sync
-- =========================================================

CREATE OR REPLACE FUNCTION set_school_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.school_updated_at := now();
  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'trg_schools_set_updated_at'
  ) THEN
    CREATE TRIGGER trg_schools_set_updated_at
    BEFORE UPDATE ON schools
    FOR EACH ROW
    EXECUTE FUNCTION set_school_updated_at();
  END IF;
END$$;

CREATE OR REPLACE FUNCTION sync_school_is_verified()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.school_is_verified := (NEW.school_verification_status = 'approved');
  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'trg_schools_sync_is_verified'
  ) THEN
    CREATE TRIGGER trg_schools_sync_is_verified
    BEFORE INSERT OR UPDATE ON schools
    FOR EACH ROW
    EXECUTE FUNCTION sync_school_is_verified();
  END IF;
END$$;


-- =====================================================================
-- SCHOOL PROFILES (1:1 ke schools)
-- =====================================================================
CREATE TABLE IF NOT EXISTS school_profiles (
  school_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi 1:1 ke school
  school_profile_school_id UUID NOT NULL UNIQUE
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Deskripsi & sejarah
  school_profile_description  TEXT,
  school_profile_founded_year INT,

  -- Alamat & kontak publikschool_profile_address
  school_profile_contact_phone VARCHAR(30),

  -- Sosial/link publik
  school_profile_google_maps_url           TEXT,
  school_profile_instagram_url             TEXT,
  school_profile_whatsapp_url              TEXT,
  school_profile_youtube_url               TEXT,
  school_profile_facebook_url              TEXT,
  school_profile_tiktok_url                TEXT,
  school_profile_whatsapp_group_ikhwan_url TEXT,
  school_profile_whatsapp_group_akhwat_url TEXT,
  school_profile_website_url               TEXT,

  -- Profil sekolah (opsional)—tanpa school_type
  school_profile_school_npsn              VARCHAR(20),
  school_profile_school_nss               VARCHAR(20),
  school_profile_school_accreditation     VARCHAR(10),
  school_profile_school_principal_user_id UUID
    REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  school_profile_school_student_capacity  INT,
  school_profile_school_is_boarding       BOOLEAN NOT NULL DEFAULT FALSE,

  -- Lokasi koordinat
  school_profile_latitude  DOUBLE PRECISION,
  school_profile_longitude DOUBLE PRECISION,

  -- Atribut tambahan
  school_profile_school_email   VARCHAR(120),
  school_profile_school_address TEXT,

  -- Audit
  school_profile_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_profile_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_profile_deleted_at TIMESTAMPTZ,

  -- Checks
  CONSTRAINT chk_mpp_founded_year CHECK (
    school_profile_founded_year IS NULL
    OR school_profile_founded_year BETWEEN 1800 AND EXTRACT(YEAR FROM now())::int
  ),
  CONSTRAINT chk_mpp_latlon_pair CHECK (
    (school_profile_latitude IS NULL AND school_profile_longitude IS NULL)
    OR (school_profile_latitude BETWEEN -90 AND 90 AND school_profile_longitude BETWEEN -180 AND 180)
  ),
  CONSTRAINT chk_mpp_school_email CHECK (
    school_profile_school_email IS NULL
    OR school_profile_school_email ~* $$^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$$
  ),
  CONSTRAINT chk_mpp_student_capacity CHECK (
    school_profile_school_student_capacity IS NULL
    OR school_profile_school_student_capacity >= 0
  ),
  CONSTRAINT chk_mpp_school_accreditation CHECK (
    school_profile_school_accreditation IS NULL
    OR school_profile_school_accreditation IN ('A','B','C','Ungraded','-')
  ),
  CONSTRAINT chk_mpp_phone CHECK (
    school_profile_contact_phone IS NULL
    OR school_profile_contact_phone ~ '^\+?[0-9]{7,20}$'
  )
);

-- =========================
-- INDEXES: SCHOOL PROFILES
-- =========================
CREATE INDEX IF NOT EXISTS idx_mpp_principal_user_id_alive
  ON school_profiles (school_profile_school_principal_user_id)
  WHERE school_profile_deleted_at IS NULL;



CREATE INDEX IF NOT EXISTS idx_mpp_school_email_lower_alive
  ON school_profiles (LOWER(school_profile_school_email))
  WHERE school_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_accreditation_alive
  ON school_profiles (school_profile_school_accreditation)
  WHERE school_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_founded_year_alive
  ON school_profiles (school_profile_founded_year)
  WHERE school_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mpp_is_boarding_alive
  ON school_profiles (school_profile_school_is_boarding)
  WHERE school_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gist_mpp_earth_alive
  ON school_profiles
  USING gist (ll_to_earth(school_profile_latitude::float8, school_profile_longitude::float8))
  WHERE school_profile_deleted_at IS NULL
    AND school_profile_latitude IS NOT NULL
    AND school_profile_longitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS trgm_mpp_description_alive
  ON school_profiles
  USING gin (school_profile_description gin_trgm_ops)
  WHERE school_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_mpp_created_at
  ON school_profiles USING brin (school_profile_created_at);

CREATE INDEX IF NOT EXISTS brin_mpp_updated_at
  ON school_profiles USING brin (school_profile_updated_at);

CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_npsn_alive
  ON school_profiles (school_profile_school_npsn)
  WHERE school_profile_deleted_at IS NULL
    AND school_profile_school_npsn IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_mpp_nss_alive
  ON school_profiles (school_profile_school_nss)
  WHERE school_profile_deleted_at IS NULL
    AND school_profile_school_nss IS NOT NULL;
