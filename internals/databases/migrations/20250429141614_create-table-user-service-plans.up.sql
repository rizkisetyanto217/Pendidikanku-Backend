BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist; -- untuk exclusion constraint overlap

-- ============================ --
-- ENUM STATUS LANGGANAN USER
-- ============================ --
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_subscription_status_enum') THEN
    CREATE TYPE user_subscription_status_enum AS ENUM (
      'trial','active','grace','canceled','expired'
    );
  END IF;
END$$;

-- ============================ --
-- TABLE USER SERVICE PLANS (katalog)
-- ============================ --
CREATE TABLE IF NOT EXISTS user_service_plans (
  user_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_service_plan_code VARCHAR(30)  NOT NULL,
  user_service_plan_name VARCHAR(100) NOT NULL,
  user_service_plan_description TEXT,

  -- Gambar (2-slot + retensi 30 hari)
  user_service_plan_image_url                  TEXT,
  user_service_plan_image_object_key           TEXT,
  user_service_plan_image_url_old              TEXT,
  user_service_plan_image_object_key_old       TEXT,
  user_service_plan_image_delete_pending_until TIMESTAMPTZ,
  CONSTRAINT chk_usp_image_old_pair CHECK (
    (user_service_plan_image_url_old IS NULL     AND user_service_plan_image_object_key_old IS NULL)
    OR
    (user_service_plan_image_url_old IS NOT NULL AND user_service_plan_image_object_key_old IS NOT NULL)
  ),

  -- Kuota/limit user-level
  user_service_plan_max_masjids_owned   INT,
  user_service_plan_max_storage_mb      INT,
  user_service_plan_max_custom_themes   INT,

  -- Harga (0 = gratis)
  user_service_plan_price_monthly NUMERIC(12,2),
  user_service_plan_price_yearly  NUMERIC(12,2),

  user_service_plan_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  user_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_service_plan_deleted_at TIMESTAMPTZ,

  -- Uniqueness & checks (paralel dengan masjid)
  CONSTRAINT ux_usp_code UNIQUE (user_service_plan_code),
  CONSTRAINT chk_usp_code_format CHECK (
    user_service_plan_code ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  ),
  CONSTRAINT chk_usp_prices_nonneg CHECK (
    (user_service_plan_price_monthly IS NULL OR user_service_plan_price_monthly >= 0) AND
    (user_service_plan_price_yearly  IS NULL OR user_service_plan_price_yearly  >= 0)
  ),
  CONSTRAINT chk_usp_limits_nonneg CHECK (
    (user_service_plan_max_masjids_owned IS NULL OR user_service_plan_max_masjids_owned >= 0) AND
    (user_service_plan_max_storage_mb    IS NULL OR user_service_plan_max_storage_mb    >= 0) AND
    (user_service_plan_max_custom_themes IS NULL OR user_service_plan_max_custom_themes >= 0)
  )
);

-- Index katalog (selaras dgn masjid)
CREATE UNIQUE INDEX IF NOT EXISTS ux_usp_code_lower
  ON user_service_plans (lower(user_service_plan_code))
  WHERE user_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usp_active_alive
  ON user_service_plans (user_service_plan_is_active)
  WHERE user_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usp_active_price_monthly_alive
  ON user_service_plans (user_service_plan_is_active, user_service_plan_price_monthly)
  WHERE user_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usp_image_purge_due
  ON user_service_plans (user_service_plan_image_delete_pending_until)
  WHERE user_service_plan_image_object_key_old IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_usp_created_at
  ON user_service_plans USING brin (user_service_plan_created_at);

CREATE INDEX IF NOT EXISTS brin_usp_updated_at
  ON user_service_plans USING brin (user_service_plan_updated_at);



-- ============================ --
-- SEED USER PLANS
-- ============================ --
INSERT INTO user_service_plans (
  user_service_plan_code, user_service_plan_name, user_service_plan_description,
  user_service_plan_max_masjids_owned, user_service_plan_max_storage_mb, user_service_plan_max_custom_themes,
  user_service_plan_price_monthly, user_service_plan_price_yearly,
  user_service_plan_is_active
) VALUES
  ('free', 'Free', 'Paket gratis untuk mulai pakai',
    1,   1024,   0,
    0,   0,
    TRUE),
  ('pro', 'Pro', 'Fitur menengah untuk power users',
    3,   10240,  3,
    49000,  490000,
    TRUE),
  ('premium', 'Premium', 'Fitur penuh & dukungan prioritas',
    10,  102400, 10,
    149000, 1490000,
    TRUE)
ON CONFLICT (user_service_plan_code) DO NOTHING;

COMMIT;