BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ============================ --
-- ENUM STATUS LANGGANAN MASJID
-- ============================ --
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'masjid_subscription_status_enum') THEN
    CREATE TYPE masjid_subscription_status_enum AS ENUM (
      'trial','active','grace','canceled','expired'
    );
  END IF;
END$$;

-- ============================ --
-- TABLE MASJID SERVICE PLANS
-- (katalog paket masjid)
-- ============================ --
CREATE TABLE IF NOT EXISTS masjid_service_plans (
  masjid_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_service_plan_code VARCHAR(30)  NOT NULL,
  masjid_service_plan_name VARCHAR(100) NOT NULL,
  masjid_service_plan_description TEXT,

  -- Gambar (2-slot + retensi 30 hari)
  masjid_service_plan_image_url                  TEXT,
  masjid_service_plan_image_object_key           TEXT,
  masjid_service_plan_image_url_old              TEXT,
  masjid_service_plan_image_object_key_old       TEXT,
  masjid_service_plan_image_delete_pending_until TIMESTAMPTZ,
  CONSTRAINT chk_msp_image_old_pair CHECK (
    (masjid_service_plan_image_url_old IS NULL     AND masjid_service_plan_image_object_key_old IS NULL)
    OR
    (masjid_service_plan_image_url_old IS NOT NULL AND masjid_service_plan_image_object_key_old IS NOT NULL)
  ),

  -- Kuota/limit
  masjid_service_plan_max_teachers   INT,
  masjid_service_plan_max_students   INT,
  masjid_service_plan_max_storage_mb INT,

  masjid_service_plan_price_monthly NUMERIC(12,2),
  masjid_service_plan_price_yearly  NUMERIC(12,2),

  masjid_service_plan_allow_custom_theme BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_max_custom_themes  INT,

  masjid_service_plan_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  masjid_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_deleted_at TIMESTAMPTZ,

  -- Uniqueness & checks
  CONSTRAINT ux_msp_code UNIQUE (masjid_service_plan_code),
  CONSTRAINT chk_msp_code_format CHECK (
    masjid_service_plan_code ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  ),
  CONSTRAINT chk_msp_prices_nonneg CHECK (
    (masjid_service_plan_price_monthly IS NULL OR masjid_service_plan_price_monthly >= 0) AND
    (masjid_service_plan_price_yearly  IS NULL OR masjid_service_plan_price_yearly  >= 0)
  ),
  CONSTRAINT chk_msp_limits_nonneg CHECK (
    (masjid_service_plan_max_teachers      IS NULL OR masjid_service_plan_max_teachers      >= 0) AND
    (masjid_service_plan_max_students      IS NULL OR masjid_service_plan_max_students      >= 0) AND
    (masjid_service_plan_max_storage_mb    IS NULL OR masjid_service_plan_max_storage_mb    >= 0) AND
    (masjid_service_plan_max_custom_themes IS NULL OR masjid_service_plan_max_custom_themes >= 0)
  )
);

-- Indexes: plans
CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_lower
  ON masjid_service_plans (lower(masjid_service_plan_code))
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON masjid_service_plans (masjid_service_plan_is_active)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON masjid_service_plans (masjid_service_plan_is_active, masjid_service_plan_price_monthly)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_image_purge_due
  ON masjid_service_plans (masjid_service_plan_image_delete_pending_until)
  WHERE masjid_service_plan_image_object_key_old IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON masjid_service_plans USING brin (masjid_service_plan_created_at);

CREATE INDEX IF NOT EXISTS brin_msp_updated_at
  ON masjid_service_plans USING brin (masjid_service_plan_updated_at);




-- ============================ --
-- TABLE MASJID SERVICE SUBSCRIPTIONS
-- ============================ --
CREATE TABLE IF NOT EXISTS masjid_service_subscriptions (
  masjid_service_subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_service_subscription_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  masjid_service_subscription_plan_id UUID NOT NULL
    REFERENCES masjid_service_plans(masjid_service_plan_id) ON DELETE RESTRICT,

  masjid_service_subscription_status masjid_subscription_status_enum NOT NULL DEFAULT 'active',
  masjid_service_subscription_is_auto_renew BOOLEAN NOT NULL DEFAULT FALSE,

  masjid_service_subscription_start_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_end_at   TIMESTAMPTZ,
  masjid_service_subscription_trial_end_at TIMESTAMPTZ,

  -- Snapshot harga saat checkout
  masjid_service_subscription_price_monthly NUMERIC(12,2),
  masjid_service_subscription_price_yearly  NUMERIC(12,2),

  -- Metadata billing
  masjid_service_subscription_provider        VARCHAR(40),
  masjid_service_subscription_provider_ref_id VARCHAR(100),
  masjid_service_subscription_canceled_at     TIMESTAMPTZ,

  -- Override kuota (NULL = ikut plan)
  masjid_service_subscription_max_teachers_override      INT,
  masjid_service_subscription_max_students_override      INT,
  masjid_service_subscription_max_storage_mb_override    INT,
  masjid_service_subscription_max_custom_themes_override INT,

  masjid_service_subscription_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_deleted_at TIMESTAMPTZ,

  -- Konsistensi
  CONSTRAINT ck_mss_time_order CHECK (
    masjid_service_subscription_end_at IS NULL
    OR masjid_service_subscription_end_at >= masjid_service_subscription_start_at
  ),
  CONSTRAINT ck_mss_overrides_nonneg CHECK (
    (masjid_service_subscription_max_teachers_override      IS NULL OR masjid_service_subscription_max_teachers_override      >= 0) AND
    (masjid_service_subscription_max_students_override      IS NULL OR masjid_service_subscription_max_students_override      >= 0) AND
    (masjid_service_subscription_max_storage_mb_override    IS NULL OR masjid_service_subscription_max_storage_mb_override    >= 0) AND
    (masjid_service_subscription_max_custom_themes_override IS NULL OR masjid_service_subscription_max_custom_themes_override >= 0)
  ),

  -- Generated column: window periode (untuk query overlap/now())
  masjid_service_subscription_period tstzrange
    GENERATED ALWAYS AS (
      tstzrange(
        masjid_service_subscription_start_at,
        COALESCE(masjid_service_subscription_end_at, 'infinity'::timestamptz),
        '[)'
      )
    ) STORED
);

-- Indexes: subscriptions
-- Maks. 1 langganan "current" (end_at IS NULL) per masjid
CREATE UNIQUE INDEX IF NOT EXISTS uq_mss_masjid_current_alive
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id)
  WHERE masjid_service_subscription_deleted_at IS NULL
    AND masjid_service_subscription_end_at IS NULL;

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_mss_masjid_alive
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id)
  WHERE masjid_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mss_plan_alive
  ON masjid_service_subscriptions (masjid_service_subscription_plan_id)
  WHERE masjid_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mss_status_alive
  ON masjid_service_subscriptions (masjid_service_subscription_status)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Performa query waktu-kini / window
CREATE INDEX IF NOT EXISTS idx_mss_current_window
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id, masjid_service_subscription_start_at DESC)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Range index untuk period (bisa dipakai cek "NOW() ∈ period")
CREATE INDEX IF NOT EXISTS gist_mss_period
  ON masjid_service_subscriptions
  USING gist (masjid_service_subscription_period)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Yang mau/sudah habis (monitoring/cron)
CREATE INDEX IF NOT EXISTS idx_mss_end_at_alive
  ON masjid_service_subscriptions (masjid_service_subscription_end_at)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Provider ref unik (hindari duplikasi webhook)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mss_provider_ref_alive
  ON masjid_service_subscriptions (masjid_service_subscription_provider, masjid_service_subscription_provider_ref_id)
  WHERE masjid_service_subscription_deleted_at IS NULL
    AND masjid_service_subscription_provider IS NOT NULL
    AND masjid_service_subscription_provider_ref_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_mss_created_at
  ON masjid_service_subscriptions USING brin (masjid_service_subscription_created_at);

CREATE INDEX IF NOT EXISTS brin_mss_updated_at
  ON masjid_service_subscriptions USING brin (masjid_service_subscription_updated_at);

-- ============================ --
-- SEED DEFAULT PLANS
-- ============================ --
INSERT INTO masjid_service_plans (
  masjid_service_plan_code, masjid_service_plan_name, masjid_service_plan_description,
  masjid_service_plan_max_teachers, masjid_service_plan_max_students, masjid_service_plan_max_storage_mb,
  masjid_service_plan_price_monthly, masjid_service_plan_price_yearly,
  masjid_service_plan_allow_custom_theme, masjid_service_plan_max_custom_themes,
  masjid_service_plan_is_active
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan',
    5, 200, 1024, 0, 0, FALSE, NULL, TRUE),
  ('premium','Premium','Fitur menengah + domain custom',
    20, 2000, 10240, 299000, 2990000, TRUE, 3, TRUE),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas',
    999, 999999, 102400, 999000, 9990000, TRUE, 20, TRUE)
ON CONFLICT (masjid_service_plan_code) DO NOTHING;

COMMIT;