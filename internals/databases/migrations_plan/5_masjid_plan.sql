-- =========================================================
-- UP Migration — masjid_service_plans (FINAL ALL-IN-ONE)
-- =========================================================
BEGIN;

-- Prasyarat
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()

-- ---------------------------------------------------------
-- 1) Tabel: MASJID_SERVICE_PLANS (super lengkap)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS masjid_service_plans (
  masjid_service_plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Identitas
  masjid_service_plan_code        VARCHAR(30)  NOT NULL,
  masjid_service_plan_name        VARCHAR(100) NOT NULL,
  masjid_service_plan_description TEXT,

  -- Kuota dasar
  masjid_service_plan_max_teachers      INT,
  masjid_service_plan_max_students      INT,
  masjid_service_plan_max_storage_mb    INT,

  -- Fitur dasar
  masjid_service_plan_allow_custom_domain    BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_certificates     BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_priority_support BOOLEAN NOT NULL DEFAULT FALSE,

  -- Harga flat
  masjid_service_plan_price_monthly NUMERIC(12,2),
  masjid_service_plan_price_yearly  NUMERIC(12,2),

  -- Tema
  masjid_service_plan_allow_custom_theme BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_max_custom_themes  INT,


  -- Status & audit waktu
  masjid_service_plan_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_service_plan_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_plan_deleted_at TIMESTAMPTZ,

  -- Guard dasar non-negatif ringkas
  CONSTRAINT chk_msp_nonnegatives CHECK (
    COALESCE(masjid_service_plan_max_teachers,0)   >= 0 AND
    COALESCE(masjid_service_plan_max_students,0)   >= 0 AND
    COALESCE(masjid_service_plan_max_storage_mb,0) >= 0
  )
);

-- ---------------------------------------------------------
-- 2) Constraints tambahan (idempotent)
-- ---------------------------------------------------------
-- Non-negatif untuk kolom tambahan
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_nonnegatives_extra') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_nonnegatives_extra CHECK (
        COALESCE(masjid_service_plan_max_admins,0)            >= 0 AND
        COALESCE(masjid_service_plan_max_classes,0)           >= 0 AND
        COALESCE(masjid_service_plan_max_lectures,0)          >= 0 AND
        COALESCE(masjid_service_plan_max_lecture_sessions,0)  >= 0 AND
        COALESCE(masjid_service_plan_max_posts,0)             >= 0 AND
        COALESCE(masjid_service_plan_max_media_assets,0)      >= 0 AND
        COALESCE(masjid_service_plan_max_custom_domains,0)    >= 0 AND
        COALESCE(masjid_service_plan_max_api_keys,0)          >= 0 AND
        COALESCE(masjid_service_plan_max_webhooks,0)          >= 0 AND
        COALESCE(masjid_service_plan_max_automations,0)       >= 0 AND
        COALESCE(masjid_service_plan_trial_days,0)            >= 0 AND
        COALESCE(masjid_service_plan_grace_days_on_past_due,0)>= 0 AND
        COALESCE(masjid_service_plan_min_term_months,0)       >= 0 AND
        COALESCE(masjid_service_plan_support_sla_hours,0)     >= 0 AND
        COALESCE(masjid_service_plan_rate_limit_rpm,0)        >= 0 AND
        COALESCE(masjid_service_plan_rate_limit_rpd,0)        >= 0 AND
        COALESCE(masjid_service_plan_overage_storage_price_per_gb,0) >= 0 AND
        COALESCE(masjid_service_plan_overage_user_price,0)           >= 0 AND
        COALESCE(masjid_service_plan_cancellation_fee,0)             >= 0 AND
        COALESCE(masjid_service_plan_vat_rate,0)                     >= 0 AND
        COALESCE(masjid_service_plan_compare_at_monthly,0)           >= 0 AND
        COALESCE(masjid_service_plan_price_per_teacher,0)            >= 0 AND
        COALESCE(masjid_service_plan_price_per_student,0)            >= 0 AND
        COALESCE(masjid_service_plan_price_per_admin,0)              >= 0 AND
        COALESCE(masjid_service_plan_min_seats,0)                    >= 0 AND
        COALESCE(masjid_service_plan_max_seats,0)                    >= 0 AND
        COALESCE(masjid_service_plan_email_quota_month,0)            >= 0 AND
        COALESCE(masjid_service_plan_sms_quota_month,0)              >= 0 AND
        COALESCE(masjid_service_plan_whatsapp_quota_month,0)         >= 0 AND
        COALESCE(masjid_service_plan_max_upload_size_mb,0)           >= 0 AND
        COALESCE(masjid_service_plan_max_concurrent_streams,0)       >= 0 AND
        COALESCE(masjid_service_plan_storage_overage_cap_gb,0)       >= 0 AND
        COALESCE(masjid_service_plan_analytics_retention_days,0)     >= 0 AND
        COALESCE(masjid_service_plan_backup_retention_days,0)        >= 0 AND
        COALESCE(masjid_service_plan_rollout_percent,0)              BETWEEN 0 AND 100
      );
  END IF;
END$$;

-- Format code
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_code_format') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_code_format CHECK (masjid_service_plan_code ~ '^[a-zA-Z0-9_-]+$');
  END IF;
END$$;

-- Enum-like checks
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_proration_mode') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_proration_mode CHECK (
        masjid_service_plan_proration_mode IS NULL OR masjid_service_plan_proration_mode IN ('none','daily')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_visibility_scope') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_visibility_scope CHECK (
        masjid_service_plan_visibility_scope IS NULL OR masjid_service_plan_visibility_scope IN ('public','internal','hidden')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_support_channel') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_support_channel CHECK (
        masjid_service_plan_support_channel IS NULL OR masjid_service_plan_support_channel IN ('email','chat','phone')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_tax_behavior') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_tax_behavior CHECK (
        masjid_service_plan_tax_behavior IS NULL OR masjid_service_plan_tax_behavior IN ('inclusive','exclusive','exempt')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_renewal_type') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_renewal_type CHECK (
        masjid_service_plan_renewal_type IS NULL OR masjid_service_plan_renewal_type IN ('manual','auto')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_free_trial_type') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_free_trial_type CHECK (
        masjid_service_plan_free_trial_type IS NULL OR masjid_service_plan_free_trial_type IN ('full','limited')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_soft_limit_behavior') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_soft_limit_behavior CHECK (
        masjid_service_plan_soft_limit_behavior IS NULL OR masjid_service_plan_soft_limit_behavior IN ('throttle','block','allow')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_hard_limit_behavior') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_hard_limit_behavior CHECK (
        masjid_service_plan_hard_limit_behavior IS NULL OR masjid_service_plan_hard_limit_behavior IN ('block','allow_paid')
      );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_msp_revrec_rule') THEN
    ALTER TABLE masjid_service_plans
      ADD CONSTRAINT chk_msp_revrec_rule CHECK (
        masjid_service_plan_revrec_rule IS NULL OR masjid_service_plan_revrec_rule IN ('ratable_monthly','on_purchase')
      );
  END IF;
END$$;

-- ---------------------------------------------------------
-- 3) Index penting
-- ---------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_lower
  ON masjid_service_plans (LOWER(masjid_service_plan_code));

CREATE INDEX IF NOT EXISTS idx_msp_active_alive
  ON masjid_service_plans (masjid_service_plan_is_active)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_msp_active_price_monthly_alive
  ON masjid_service_plans (masjid_service_plan_is_active, masjid_service_plan_price_monthly)
  WHERE masjid_service_plan_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_msp_created_at
  ON masjid_service_plans USING brin (masjid_service_plan_created_at);

-- ---------------------------------------------------------
-- 4) Trigger auto update updated_at
-- ---------------------------------------------------------
CREATE OR REPLACE FUNCTION set_msp_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.masjid_service_plan_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_msp_set_updated_at ON masjid_service_plans;
CREATE TRIGGER trg_msp_set_updated_at
BEFORE UPDATE ON masjid_service_plans
FOR EACH ROW
EXECUTE FUNCTION set_msp_updated_at();

-- ---------------------------------------------------------
-- 5) Seed data idempotent (ringkas, bisa kamu perluas)
-- ---------------------------------------------------------
INSERT INTO masjid_service_plans (
  masjid_service_plan_code, masjid_service_plan_name, masjid_service_plan_description,
  masjid_service_plan_max_teachers, masjid_service_plan_max_students, masjid_service_plan_max_storage_mb,
  masjid_service_plan_allow_custom_domain, masjid_service_plan_allow_certificates, masjid_service_plan_allow_priority_support,
  masjid_service_plan_price_monthly, masjid_service_plan_price_yearly,
  masjid_service_plan_allow_custom_theme, masjid_service_plan_max_custom_themes,
  masjid_service_plan_is_active,
  masjid_service_plan_support_channel, masjid_service_plan_support_sla_hours, masjid_service_plan_success_manager,
  masjid_service_plan_features, masjid_service_plan_limits,
  masjid_service_plan_currency, masjid_service_plan_tax_behavior, masjid_service_plan_vat_rate
)
VALUES
  ('basic','Basic','Fitur dasar untuk mulai jalan',
    5, 200, 1024,
    FALSE, FALSE, FALSE,
    0, 0,
    FALSE, NULL,
    TRUE,
    'email', 72, FALSE,
    '{"attendance":true,"quiz":false}'::jsonb,
    '{"api_rpm":0,"custom_domains":0}'::jsonb,
    'IDR','inclusive',0
  ),
  ('premium','Premium','Fitur menengah + domain custom',
    20, 2000, 10240,
    TRUE, TRUE, TRUE,
    299000, 2990000,
    TRUE, 3,
    TRUE,
    'chat', 24, FALSE,
    '{"attendance":true,"quiz":true,"certificate":true,"custom_domain":true,"api":true,"webhooks":true}'::jsonb,
    '{"api_rpm":600,"custom_domains":3}'::jsonb,
    'IDR','inclusive',11.00
  ),
  ('exclusive','Eksklusif','Fitur penuh & dukungan prioritas',
    999, 999999, 102400,
    TRUE, TRUE, TRUE,
    999000, 9990000,
    TRUE, 20,
    TRUE,
    'phone', 4, TRUE,
    '{"attendance":true,"quiz":true,"certificate":true,"custom_domain":true,"sso":true,"backup_restore":true,"multisite":true}'::jsonb,
    '{"api_rpm":3000,"custom_domains":20}'::jsonb,
    'IDR','inclusive',11.00
  )
ON CONFLICT ((lower(masjid_service_plan_code))) DO NOTHING;

COMMIT;
