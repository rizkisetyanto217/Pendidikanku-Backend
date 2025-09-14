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
  masjid_service_plan_deleted_reason TEXT,

  -- ==============================
  -- Kuota & batas detail
  -- ==============================
  masjid_service_plan_max_admins           INT,
  masjid_service_plan_max_classes          INT,
  masjid_service_plan_max_lectures         INT,
  masjid_service_plan_max_lecture_sessions INT,
  masjid_service_plan_max_posts            INT,
  masjid_service_plan_max_media_assets     INT,
  masjid_service_plan_max_custom_domains   INT,
  masjid_service_plan_max_api_keys         INT,
  masjid_service_plan_max_webhooks         INT,
  masjid_service_plan_max_automations      INT,

  -- ==============================
  -- Fitur granular
  -- ==============================
  masjid_service_plan_allow_api_access        BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_webhooks          BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_automations       BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_branding_removal  BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_sso               BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_multisite         BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_allow_backup_restore    BOOLEAN NOT NULL DEFAULT FALSE,

  -- ==============================
  -- Billing & pricing detail
  -- ==============================
  masjid_service_plan_currency               VARCHAR(10)  DEFAULT 'IDR',
  masjid_service_plan_tax_inclusive          BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_service_plan_vat_rate               NUMERIC(5,2),
  masjid_service_plan_trial_days             INT,
  masjid_service_plan_grace_days_on_past_due INT,
  masjid_service_plan_proration_mode         VARCHAR(20),
  masjid_service_plan_min_term_months        INT,
  masjid_service_plan_cancellation_fee       NUMERIC(12,2),

  -- ==============================
  -- Overage & rate limit
  -- ==============================
  masjid_service_plan_overage_enabled                  BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_service_plan_overage_storage_price_per_gb     NUMERIC(12,2),
  masjid_service_plan_overage_user_price               NUMERIC(12,2),
  masjid_service_plan_rate_limit_rpm                   INT,
  masjid_service_plan_rate_limit_rpd                   INT,

  -- ==============================
  -- Support / SLA
  -- ==============================
  masjid_service_plan_support_channel   VARCHAR(30),   -- 'email'|'chat'|'phone'
  masjid_service_plan_support_sla_hours INT,
  masjid_service_plan_success_manager   BOOLEAN NOT NULL DEFAULT FALSE,

  -- ==============================
  -- Marketing & katalog
  -- ==============================
  masjid_service_plan_is_public           BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_service_plan_visibility_scope    VARCHAR(20), -- 'public'|'internal'|'hidden'
  masjid_service_plan_sort_order          INT,
  masjid_service_plan_badge_label         VARCHAR(30),
  masjid_service_plan_marketing_highlight TEXT,
  masjid_service_plan_display_note        VARCHAR(200),

  -- ==============================
  -- Fleksibel: fitur/limit JSON
  -- ==============================
  masjid_service_plan_features JSONB,
  masjid_service_plan_limits   JSONB,

  -- ==============================
  -- Per-seat & usage pricing
  -- ==============================
  masjid_service_plan_price_per_teacher NUMERIC(12,2),
  masjid_service_plan_price_per_student NUMERIC(12,2),
  masjid_service_plan_price_per_admin   NUMERIC(12,2),
  masjid_service_plan_min_seats         INT,
  masjid_service_plan_max_seats         INT,

  -- ==============================
  -- Regional / External mapping
  -- ==============================
  masjid_service_plan_region              VARCHAR(40),  -- 'ID','MY','GLOBAL'
  masjid_service_plan_data_residency      VARCHAR(40),  -- 'ap-southeast-1', etc.
  masjid_service_plan_external_product_id VARCHAR(120), -- payment catalog id
  masjid_service_plan_appstore_product_id VARCHAR(120),

  -- ==============================
  -- Communication quotas
  -- ==============================
  masjid_service_plan_email_quota_month    INT,
  masjid_service_plan_sms_quota_month      INT,
  masjid_service_plan_whatsapp_quota_month INT,

  -- ==============================
  -- Media/streaming
  -- ==============================
  masjid_service_plan_max_upload_size_mb      INT,
  masjid_service_plan_max_concurrent_streams  INT,
  masjid_service_plan_storage_overage_cap_gb  INT,

  -- ==============================
  -- Integrations / Modules / AI
  -- ==============================
  masjid_service_plan_integrations JSONB,  -- {"midtrans":true,"xendit":true}
  masjid_service_plan_modules      JSONB,  -- {"exam":true,"streaming":false}
  masjid_service_plan_ai_features  JSONB,  -- {"summarize":true,"ocr":false}

  -- ==============================
  -- Eligibility & targeting
  -- ==============================
  masjid_service_plan_min_org_size      INT,
  masjid_service_plan_max_org_size      INT,
  masjid_service_plan_allowed_countries TEXT[],
  masjid_service_plan_denied_countries  TEXT[],
  masjid_service_plan_eligibility_notes TEXT,

  -- ==============================
  -- Billing lifecycle & renewal
  -- ==============================
  masjid_service_plan_auto_renew           BOOLEAN DEFAULT TRUE,
  masjid_service_plan_renewal_type         VARCHAR(20), -- 'manual'|'auto'
  masjid_service_plan_cancel_at_period_end BOOLEAN DEFAULT FALSE,
  masjid_service_plan_free_trial_type      VARCHAR(20), -- 'full'|'limited'
  masjid_service_plan_discount_allowed     BOOLEAN DEFAULT TRUE,
  masjid_service_plan_coupon_allowed       BOOLEAN DEFAULT TRUE,

  -- ==============================
  -- Suspension / grace
  -- ==============================
  masjid_service_plan_past_due_suspension_days INT,
  masjid_service_plan_soft_limit_behavior      VARCHAR(20), -- 'throttle'|'block'|'allow'
  masjid_service_plan_hard_limit_behavior      VARCHAR(20), -- 'block'|'allow_paid'

  -- ==============================
  -- A/B rollout & versioning
  -- ==============================
  masjid_service_plan_rollout_percent   NUMERIC(5,2),  -- 0..100
  masjid_service_plan_experiment_bucket VARCHAR(40),   -- 'A','B','beta1'
  masjid_service_plan_version           VARCHAR(20),   -- 'v1'
  masjid_service_plan_replaced_by_code  VARCHAR(30),

  -- ==============================
  -- Analytics & retention
  -- ==============================
  masjid_service_plan_analytics_retention_days INT,
  masjid_service_plan_backup_retention_days    INT,

  -- ==============================
  -- Accessibility / branding
  -- ==============================
  masjid_service_plan_accessibility_features JSONB, -- {"high_contrast":true}
  masjid_service_plan_brand_assets_allowed  BOOLEAN DEFAULT FALSE,

  -- ==============================
  -- Lokalisasi (multi-bahasa)
  -- ==============================
  masjid_service_plan_name_i18n        JSONB, -- {"id":"Premium","en":"Premium"}
  masjid_service_plan_description_i18n JSONB, -- {"id":"...","en":"..."}

  -- ==============================
  -- Pajak & regional prices lanjutan
  -- ==============================
  masjid_service_plan_tax_code        VARCHAR(40),  -- PPN11, dsb
  masjid_service_plan_tax_behavior    VARCHAR(20),  -- 'inclusive'|'exclusive'|'exempt'
  masjid_service_plan_regional_prices JSONB,        -- {"IDR":{"monthly":299000,"yearly":2990000}, ...}

  -- ==============================
  -- Promo/marketing
  -- ==============================
  masjid_service_plan_compare_at_monthly NUMERIC(12,2), -- harga coret
  masjid_service_plan_promo_starts_at     TIMESTAMPTZ,
  masjid_service_plan_promo_ends_at       TIMESTAMPTZ,

  -- ==============================
  -- Bundling & add-ons
  -- ==============================
  masjid_service_plan_includes_addons   JSONB, -- {"sms_pack":true}
  masjid_service_plan_compatible_addons JSONB, -- {"ai_pack":true,"storage_plus":true}

  -- ==============================
  -- Metered billing (definisi metrik)
  -- ==============================
  masjid_service_plan_metered_metrics JSONB, -- [{"key":"emails","unit":"count","rate_per_unit":50}, ...]

  -- ==============================
  -- Aturan lifecycle upgrade/downgrade
  -- ==============================
  masjid_service_plan_upgrade_paths JSONB, -- {"to":["premium","exclusive"]}
  masjid_service_plan_downgrade_paths JSONB, -- {"to":["basic"]}
  masjid_service_plan_change_fee NUMERIC(12,2),

  -- ==============================
  -- Kepatuhan & perjanjian
  -- ==============================
  masjid_service_plan_terms_url     TEXT,
  masjid_service_plan_privacy_url   TEXT,
  masjid_service_plan_data_regions  TEXT[], -- ['ID','SG']

  -- ==============================
  -- Akuntansi
  -- ==============================
  masjid_service_plan_gl_code     VARCHAR(40),  -- kode akun pendapatan
  masjid_service_plan_revrec_rule VARCHAR(30),  -- 'ratable_monthly'|'on_purchase'

  -- ==============================
  -- Catatan publik & deprecation
  -- ==============================
  masjid_service_plan_public_notes     TEXT,
  masjid_service_plan_deprecated_at    TIMESTAMPTZ,
  masjid_service_plan_deprecated_reason TEXT,

  -- Audit user
  masjid_service_plan_created_by_user_id UUID,
  masjid_service_plan_updated_by_user_id UUID,

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
