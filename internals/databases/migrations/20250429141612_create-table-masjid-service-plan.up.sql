BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1) Table (fresh, lengkap dgn image single 2-slot)
CREATE TABLE IF NOT EXISTS masjid_service_plans (
  masjid_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_service_plan_code VARCHAR(30)  NOT NULL,   -- 'basic' | 'premium' | 'exclusive'
  masjid_service_plan_name VARCHAR(100) NOT NULL,
  masjid_service_plan_description TEXT,

  -- Image (single file, 2-slot + retensi)
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

  -- Tema per-plan
  masjid_service_plan_allow_custom_theme BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_max_custom_themes  INT,

  masjid_service_plan_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  masjid_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_deleted_at TIMESTAMPTZ,

  -- Kolom generated untuk unique case-insensitive
  masjid_service_plan_code_ci TEXT GENERATED ALWAYS AS (lower(masjid_service_plan_code)) STORED
);

-- 2) Indexing
-- (hapus index lama functional kalau ada)
DROP INDEX IF EXISTS ux_msp_code_lower;

-- Unique index di kolom generated (bukan table constraint)
CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_ci
  ON masjid_service_plans (masjid_service_plan_code_ci);

-- Partial index utk query umum
CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON masjid_service_plans (masjid_service_plan_is_active)
  WHERE masjid_service_plan_deleted_at IS NULL;

-- Kombinasi aktif + price_monthly (katalog)
CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON masjid_service_plans (masjid_service_plan_is_active, masjid_service_plan_price_monthly)
  WHERE masjid_service_plan_deleted_at IS NULL;

-- BRIN untuk time-range scan
CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON masjid_service_plans USING brin (masjid_service_plan_created_at);

-- 3) Seed (pakai conflict target kolom generated agar case-insensitive)
INSERT INTO masjid_service_plans (
  masjid_service_plan_code, masjid_service_plan_name, masjid_service_plan_description,
  masjid_service_plan_max_teachers, masjid_service_plan_max_students, masjid_service_plan_max_storage_mb,
  masjid_service_plan_price_monthly, masjid_service_plan_price_yearly,
  masjid_service_plan_allow_custom_theme, masjid_service_plan_max_custom_themes,
  masjid_service_plan_is_active
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan',
    5, 200, 1024,
    0, 0,
    FALSE, NULL,
    TRUE
  ),
  ('premium','Premium','Fitur menengah + domain custom',
    20, 2000, 10240,
    299000, 2990000,
    TRUE, 3,
    TRUE
  ),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas',
    999, 999999, 102400,
    999000, 9990000,
    TRUE, 20,
    TRUE
  )
ON CONFLICT (masjid_service_plan_code_ci) DO NOTHING;

COMMIT;
