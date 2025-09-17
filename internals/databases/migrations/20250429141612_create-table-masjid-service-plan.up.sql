BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS masjid_service_plans (
  masjid_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_service_plan_code VARCHAR(30)  NOT NULL,
  masjid_service_plan_name VARCHAR(100) NOT NULL,
  masjid_service_plan_description TEXT,

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
  masjid_service_plan_deleted_at TIMESTAMPTZ
);

-- Tambahkan UNIQUE (wajib agar ON CONFLICT (code) valid)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = current_schema()
      AND indexname  = 'ux_msp_code'
  ) THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT ux_msp_code UNIQUE (masjid_service_plan_code);
  END IF;
END$$;

-- Index lain (opsional)
DROP INDEX IF EXISTS ux_msp_code_lower;

CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON masjid_service_plans (masjid_service_plan_is_active)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON masjid_service_plans (masjid_service_plan_is_active, masjid_service_plan_price_monthly)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON masjid_service_plans USING brin (masjid_service_plan_created_at);

-- Seed
INSERT INTO masjid_service_plans (
  masjid_service_plan_code, masjid_service_plan_name, masjid_service_plan_description,
  masjid_service_plan_max_teachers, masjid_service_plan_max_students, masjid_service_plan_max_storage_mb,
  masjid_service_plan_price_monthly, masjid_service_plan_price_yearly,
  masjid_service_plan_allow_custom_theme, masjid_service_plan_max_custom_themes,
  masjid_service_plan_is_active
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan', 5, 200, 1024, 0, 0, FALSE, NULL, TRUE),
  ('premium','Premium','Fitur menengah + domain custom', 20, 2000, 10240, 299000, 2990000, TRUE, 3, TRUE),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas', 999, 999999, 102400, 999000, 9990000, TRUE, 20, TRUE)
ON CONFLICT (masjid_service_plan_code) DO NOTHING;

COMMIT;
