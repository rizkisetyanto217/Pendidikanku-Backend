BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------
-- 1) Tabel: MASJID_SERVICE_PLANS
--    - Unique code pakai UNIQUE INDEX pada LOWER(code)
--    - Tambahan flag tema (custom theme)
--    - Check nilai non-negatif
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS masjid_service_plans (
  masjid_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_service_plan_code VARCHAR(30) NOT NULL,  -- 'basic' | 'premium' | 'exclusive'
  masjid_service_plan_name VARCHAR(100) NOT NULL,
  masjid_service_plan_description TEXT,

  masjid_service_plan_max_teachers   INT,
  masjid_service_plan_max_students   INT,
  masjid_service_plan_max_storage_mb INT,

  masjid_service_plan_allow_custom_domain    BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_certificates     BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_priority_support BOOLEAN NOT NULL DEFAULT FALSE,

  masjid_service_plan_price_monthly NUMERIC(12,2),
  masjid_service_plan_price_yearly  NUMERIC(12,2),

  -- Tema per-plan
  masjid_service_plan_allow_custom_theme BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_max_custom_themes  INT,

  masjid_service_plan_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_deleted_at TIMESTAMPTZ NULL,

  -- Guard nilai non-negatif
  CONSTRAINT chk_msp_nonnegatives CHECK (
    COALESCE(masjid_service_plan_max_teachers,0)      >= 0 AND
    COALESCE(masjid_service_plan_max_students,0)      >= 0 AND
    COALESCE(masjid_service_plan_max_storage_mb,0)    >= 0 AND
    COALESCE(masjid_service_plan_price_monthly,0)     >= 0 AND
    COALESCE(masjid_service_plan_price_yearly,0)      >= 0 AND
    COALESCE(masjid_service_plan_max_custom_themes,0) >= 0
  )
);

-- ---------------------------------------------------------
-- 2) Indexing penting
-- ---------------------------------------------------------

-- Unique, case-insensitive untuk code
CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_lower
  ON masjid_service_plans (LOWER(masjid_service_plan_code));

-- Daftar plan aktif (soft-delete aware)
CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON masjid_service_plans (masjid_service_plan_is_active)
  WHERE masjid_service_plan_deleted_at IS NULL;

-- Listing aktif + urut harga bulanan (katalog/pricing page)
CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON masjid_service_plans (masjid_service_plan_is_active, masjid_service_plan_price_monthly)
  WHERE masjid_service_plan_deleted_at IS NULL;

-- Arsip/sort waktu hemat
CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON masjid_service_plans USING brin (masjid_service_plan_created_at);

-- ---------------------------------------------------------
-- 3) Seed data (idempotent via ON CONFLICT DO NOTHING)
-- ---------------------------------------------------------
INSERT INTO masjid_service_plans (
  masjid_service_plan_code, masjid_service_plan_name, masjid_service_plan_description,
  masjid_service_plan_max_teachers, masjid_service_plan_max_students, masjid_service_plan_max_storage_mb,
  masjid_service_plan_allow_custom_domain, masjid_service_plan_allow_certificates, masjid_service_plan_allow_priority_support,
  masjid_service_plan_price_monthly, masjid_service_plan_price_yearly,
  masjid_service_plan_allow_custom_theme, masjid_service_plan_max_custom_themes,
  masjid_service_plan_is_active
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan',
    5, 200, 1024,
    FALSE, FALSE, FALSE,
    0, 0,
    FALSE, NULL,
    TRUE
  ),
  ('premium','Premium','Fitur menengah + domain custom',
    20, 2000, 10240,
    TRUE, TRUE, TRUE,
    299000, 2990000,
    TRUE, 3,
    TRUE
  ),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas',
    999, 999999, 102400,
    TRUE, TRUE, TRUE,
    999000, 9990000,
    TRUE, 20,
    TRUE
  )
ON CONFLICT (masjid_service_plan_code) DO NOTHING;

COMMIT;
