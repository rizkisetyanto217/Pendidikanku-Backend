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
  yayasan_npwp                   VARCHAR(32),
  yayasan_tax_status             VARCHAR(20),    -- 'npwp_aktif'|'npwp_nonaktif'|'tidak_ada'
  yayasan_founded_year           SMALLINT,

  -- Compliance lokal (Indonesia)
  yayasan_oss_nib                VARCHAR(50),    -- Nomor Induk Berusaha
  yayasan_ahu_id                 VARCHAR(50),    -- ID AHU Kemenkumham
  yayasan_siop_status            VARCHAR(20),    -- 'ok'|'review'|'expired'
  yayasan_tax_pkp_status         VARCHAR(20),    -- 'pkp'|'non_pkp'

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
  yayasan_timezone               VARCHAR(64),

  -- PIC
  yayasan_contact_person_name    VARCHAR(100),
  yayasan_contact_person_role    VARCHAR(80),
  yayasan_contact_person_email   CITEXT,
  yayasan_contact_person_phone   VARCHAR(32),

  -- Hierarki / Organisasi
  yayasan_parent_id              UUID REFERENCES yayasans(yayasan_id) ON DELETE SET NULL,
  yayasan_code                   VARCHAR(32),    -- kode internal singkat
  yayasan_short_alias            VARCHAR(30),    -- alias pendek untuk URL/QR

  -- Governance & Organisasi
  yayasan_board_structure        JSONB,          -- struktur pengurus
  yayasan_focus_area             TEXT[],         -- {"pendidikan","sosial","kesehatan"}
  yayasan_mission                TEXT,
  yayasan_vision                 TEXT,
  yayasan_values                 TEXT[],
  yayasan_awards                 JSONB,

  -- Finansial & Donasi
  yayasan_bank_name              VARCHAR(80),
  yayasan_bank_account_name      VARCHAR(120),
  yayasan_bank_account_number    VARCHAR(50),
  yayasan_donation_min_amount    BIGINT,
  yayasan_donation_fee_mode      VARCHAR(20),    -- 'include'|'exclude'|'sponsor'
  yayasan_donation_split_ratio   NUMERIC(5,2),   -- 0..100
  yayasan_donation_notes         TEXT,
  yayasan_fundraising_goal       BIGINT,
  yayasan_fundraising_achieved   BIGINT,
  yayasan_last_donation_at       TIMESTAMPTZ,

  -- Operasional
  yayasan_operating_hours        JSONB,          -- { "mon":[["08:00","16:00"]], ... }
  yayasan_service_area           JSONB,          -- { "radius_km": 10, "cities": ["..."] }
  yayasan_emergency_hotline      VARCHAR(32),

  -- Integrasi & Eksternal
  yayasan_external_ids           JSONB,          -- {"ahu":"...","nib":"..."}
  yayasan_settings               JSONB,          -- preferensi/feature toggles
  yayasan_feature_flags          JSONB,          -- alternatif pemisahan flags
  yayasan_theme_code             VARCHAR(50),
  yayasan_api_key                TEXT,
  yayasan_api_secret             TEXT,
  yayasan_sso_provider           VARCHAR(30),    -- 'google'|'microsoft'|'none'

  -- Billing & Compliance Ops
  yayasan_billing_plan           VARCHAR(20),    -- 'free'|'pro'|'enterprise'
  yayasan_billing_status         VARCHAR(20),    -- 'active'|'past_due'|'canceled'
  yayasan_billing_cycle_anchor   TIMESTAMPTZ,
  yayasan_trial_ends_at          TIMESTAMPTZ,
  yayasan_invoice_email          CITEXT,
  yayasan_payment_provider       VARCHAR(30),    -- 'xendit'|'midtrans'|'stripe'|'manual'
  yayasan_payment_customer_id    VARCHAR(120),
  yayasan_compliance_status      VARCHAR(20),    -- 'ok'|'review'|'blocked'
  yayasan_compliance_notes       TEXT,
  yayasan_dpa_url                TEXT,

  -- Moderation / Suspensi / Delete
  yayasan_is_suspended           BOOLEAN NOT NULL DEFAULT FALSE,
  yayasan_suspended_at           TIMESTAMPTZ,
  yayasan_suspension_reason      TEXT,
  yayasan_deleted_reason         TEXT,
  yayasan_deleted_by_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,

  -- Webhook
  yayasan_webhook_url            TEXT,
  yayasan_webhook_secret         TEXT,
  yayasan_webhook_is_active      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Security & Tagging
  yayasan_ip_allowlist           CIDR[],
  yayasan_tags                   TEXT[],

  -- Preferensi & Privasi
  yayasan_default_currency       VARCHAR(10),
  yayasan_default_locale         VARCHAR(10),
  yayasan_support_tier           VARCHAR(20),
  yayasan_notify_email_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
  yayasan_notify_whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  yayasan_marketing_consent_at   TIMESTAMPTZ,
  yayasan_data_retention_days    INTEGER,

  -- Listing & SEO
  yayasan_is_listed              BOOLEAN NOT NULL DEFAULT TRUE,
  yayasan_tagline                VARCHAR(150),
  yayasan_meta_title             VARCHAR(150),
  yayasan_meta_description       VARCHAR(200),
  yayasan_description_short      VARCHAR(200),
  yayasan_description_long       TEXT,

  -- Statistik / Engagement (cache)
  yayasan_masjid_count_cache     INTEGER NOT NULL DEFAULT 0,
  yayasan_post_count_cache       INTEGER NOT NULL DEFAULT 0,
  yayasan_storage_usage_bytes    BIGINT  NOT NULL DEFAULT 0,
  yayasan_total_staff_cache      INTEGER NOT NULL DEFAULT 0,
  yayasan_total_students_cache   INTEGER NOT NULL DEFAULT 0,
  yayasan_total_units_cache      INTEGER NOT NULL DEFAULT 0,
  yayasan_total_visitors_cache   BIGINT  NOT NULL DEFAULT 0,
  yayasan_total_followers_cache  BIGINT  NOT NULL DEFAULT 0,
  yayasan_popularity_rank        INTEGER,

  -- Risk & Audit
  yayasan_risk_level             VARCHAR(20),    -- 'low'|'medium'|'high'
  yayasan_risk_score             SMALLINT,       -- 0..100
  yayasan_risk_flags             JSONB,
  yayasan_last_activity_at       TIMESTAMPTZ,
  yayasan_last_audit_at          TIMESTAMPTZ,
  yayasan_audit_notes            TEXT,
  yayasan_internal_notes         TEXT,

  -- Audit by
  yayasan_created_by_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
  yayasan_updated_by_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
  yayasan_verified_by_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,

  -- ===== URL/Dokumen Pendukung =====
  yayasan_logo_url               TEXT,
  yayasan_banner_url             TEXT,
  yayasan_profile_document_url   TEXT,
  yayasan_statute_url            TEXT,           -- AD/ART
  yayasan_akta_pendirian_url     TEXT,
  yayasan_akta_perubahan_url     TEXT,
  yayasan_sk_kemenkumham_url     TEXT,
  yayasan_siop_document_url      TEXT,
  yayasan_tax_certificate_url    TEXT,
  yayasan_audit_report_url       TEXT,
  yayasan_annual_report_url      TEXT,

  -- Audit waktu
  yayasan_created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  yayasan_deleted_at             TIMESTAMPTZ,

  -- Validasi koordinat sederhana
  CONSTRAINT yayasans_lat_chk CHECK (yayasan_latitude  BETWEEN -90  AND 90),
  CONSTRAINT yayasans_lon_chk CHECK (yayasan_longitude BETWEEN -180 AND 180)
);
