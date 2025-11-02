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
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'school_subscription_status_enum') THEN
    CREATE TYPE school_subscription_status_enum AS ENUM (
      'trial','active','grace','canceled','expired'
    );
  END IF;
END$$;

-- ============================ --
-- TABLE MASJID SERVICE PLANS
-- (katalog paket school)
-- ============================ --
CREATE TABLE IF NOT EXISTS school_service_plans (
  school_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  school_service_plan_code VARCHAR(30)  NOT NULL,
  school_service_plan_name VARCHAR(100) NOT NULL,
  school_service_plan_description TEXT,

  -- Gambar (2-slot + retensi 30 hari)
  school_service_plan_image_url                  TEXT,
  school_service_plan_image_object_key           TEXT,
  school_service_plan_image_url_old              TEXT,
  school_service_plan_image_object_key_old       TEXT,
  school_service_plan_image_delete_pending_until TIMESTAMPTZ,
  CONSTRAINT chk_msp_image_old_pair CHECK (
    (school_service_plan_image_url_old IS NULL     AND school_service_plan_image_object_key_old IS NULL)
    OR
    (school_service_plan_image_url_old IS NOT NULL AND school_service_plan_image_object_key_old IS NOT NULL)
  ),

  -- Kuota/limit
  school_service_plan_max_teachers   INT,
  school_service_plan_max_students   INT,
  school_service_plan_max_storage_mb INT,

  school_service_plan_price_monthly NUMERIC(12,2),
  school_service_plan_price_yearly  NUMERIC(12,2),

  school_service_plan_allow_custom_theme BOOLEAN NOT NULL DEFAULT FALSE,
  school_service_plan_max_custom_themes  INT,

  school_service_plan_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  school_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_service_plan_deleted_at TIMESTAMPTZ,

  -- Uniqueness & checks
  CONSTRAINT ux_msp_code UNIQUE (school_service_plan_code),
  CONSTRAINT chk_msp_code_format CHECK (
    school_service_plan_code ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  ),
  CONSTRAINT chk_msp_prices_nonneg CHECK (
    (school_service_plan_price_monthly IS NULL OR school_service_plan_price_monthly >= 0) AND
    (school_service_plan_price_yearly  IS NULL OR school_service_plan_price_yearly  >= 0)
  ),
  CONSTRAINT chk_msp_limits_nonneg CHECK (
    (school_service_plan_max_teachers      IS NULL OR school_service_plan_max_teachers      >= 0) AND
    (school_service_plan_max_students      IS NULL OR school_service_plan_max_students      >= 0) AND
    (school_service_plan_max_storage_mb    IS NULL OR school_service_plan_max_storage_mb    >= 0) AND
    (school_service_plan_max_custom_themes IS NULL OR school_service_plan_max_custom_themes >= 0)
  )
);

-- Indexes: plans
CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_lower
  ON school_service_plans (lower(school_service_plan_code))
  WHERE school_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON school_service_plans (school_service_plan_is_active)
  WHERE school_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON school_service_plans (school_service_plan_is_active, school_service_plan_price_monthly)
  WHERE school_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_image_purge_due
  ON school_service_plans (school_service_plan_image_delete_pending_until)
  WHERE school_service_plan_image_object_key_old IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON school_service_plans USING brin (school_service_plan_created_at);

CREATE INDEX IF NOT EXISTS brin_msp_updated_at
  ON school_service_plans USING brin (school_service_plan_updated_at);



-- ============================ --
-- SEED DEFAULT PLANS
-- ============================ --
INSERT INTO school_service_plans (
  school_service_plan_code, school_service_plan_name, school_service_plan_description,
  school_service_plan_max_teachers, school_service_plan_max_students, school_service_plan_max_storage_mb,
  school_service_plan_price_monthly, school_service_plan_price_yearly,
  school_service_plan_allow_custom_theme, school_service_plan_max_custom_themes,
  school_service_plan_is_active
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan',
    5, 200, 1024, 0, 0, FALSE, NULL, TRUE),
  ('premium','Premium','Fitur menengah + domain custom',
    20, 2000, 10240, 299000, 2990000, TRUE, 3, TRUE),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas',
    999, 999999, 102400, 999000, 9990000, TRUE, 20, TRUE)
ON CONFLICT (school_service_plan_code) DO NOTHING;

COMMIT;