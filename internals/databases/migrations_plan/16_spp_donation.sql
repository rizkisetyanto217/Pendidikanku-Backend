-- =========================================================
-- UP MIGRATION — spp_billings, user_spp_billings, donations
-- FINAL VERSION
-- =========================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

BEGIN;

-- =========================================================
-- TABLE: spp_billings
-- =========================================================
DROP TABLE IF EXISTS spp_billings CASCADE;
CREATE TABLE IF NOT EXISTS spp_billings (
  spp_billing_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- konteks
  spp_billing_masjid_id        UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  spp_billing_class_id         UUID REFERENCES classes(class_id)   ON DELETE SET NULL,
  spp_billing_term_id          UUID,
  spp_billing_academic_year_id UUID,

  -- identitas & penamaan
  spp_billing_code             VARCHAR(60),
  spp_billing_title            TEXT NOT NULL,
  spp_billing_slug             VARCHAR(160),

  -- periode penagihan
  spp_billing_month            SMALLINT NOT NULL CHECK (spp_billing_month BETWEEN 1 AND 12),
  spp_billing_year             SMALLINT  NOT NULL CHECK (spp_billing_year BETWEEN 2000 AND 2100),
  spp_billing_cycle            TEXT,
  spp_billing_period_start     DATE,
  spp_billing_period_end       DATE,

  -- nominal & mata uang
  spp_billing_amount_cents     BIGINT,
  spp_billing_min_amount_cents BIGINT,
  spp_billing_max_amount_cents BIGINT,

  -- diskon, beasiswa, potongan
  spp_billing_discount_percent NUMERIC(6,3),
  spp_billing_discount_cents   BIGINT,
  spp_billing_scholarship_cents BIGINT,

  -- denda & grace period
  spp_billing_due_date         DATE,
  spp_billing_grace_days       SMALLINT,

  -- penjadwalan & siklus
  spp_billing_status           TEXT,
  spp_billing_status_reason    TEXT,
  spp_billing_open_at          TIMESTAMPTZ,
  spp_billing_close_at         TIMESTAMPTZ,
  spp_billing_recurring_rule   TEXT,
  spp_billing_locked           BOOLEAN DEFAULT FALSE,

  -- pengingat
  spp_billing_reminder_days_before INT[],
  spp_billing_reminder_days_after  INT[],
  spp_billing_reminder_channels    TEXT[],

  -- closedown
  spp_billing_closed_by_user_id UUID,
  spp_billing_closed_at TIMESTAMPTZ,

  -- metadata
  spp_billing_category TEXT,
  spp_billing_source_system TEXT,
  spp_billing_import_batch_id TEXT,
  spp_billing_note             TEXT,
  spp_billing_tags             TEXT[],
  spp_billing_external_ref     TEXT,
  spp_billing_extra            JSONB,
  spp_billing_deleted_reason   TEXT,
  spp_billing_audit            JSONB,

  -- audit aktor
  spp_billing_created_by_user_id UUID,
  spp_billing_updated_by_user_id UUID,
  spp_billing_deleted_by_user_id UUID,

  -- concurrency
  spp_billing_row_version INT DEFAULT 1,
  spp_billing_etag TEXT,

  -- timestamps
  spp_billing_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_deleted_at       TIMESTAMPTZ
);

-- =========================================================
-- TABLE: user_spp_billings
-- =========================================================
DROP TABLE IF EXISTS user_spp_billings CASCADE;
CREATE TABLE IF NOT EXISTS user_spp_billings (
  user_spp_billing_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- relasi
  user_spp_billing_billing_id     UUID NOT NULL REFERENCES spp_billings(spp_billing_id) ON DELETE CASCADE,
  user_spp_billing_user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
  user_spp_billing_masjid_student_id UUID,
  user_spp_billing_guardian_id    UUID,
  user_spp_billing_user_class_id  UUID,
  user_spp_billing_masjid_id      UUID,

  -- identitas invoice
  user_spp_billing_invoice_no     VARCHAR(80),
  user_spp_billing_invoice_slug   VARCHAR(160),
  user_spp_billing_public_no      VARCHAR(40),
  user_spp_billing_dedup_key      TEXT,

  -- nominal
  user_spp_billing_amount_cents   BIGINT,
  user_spp_billing_discount_cents BIGINT,
  user_spp_billing_scholarship_cents BIGINT,
  user_spp_billing_penalty_cents  BIGINT,
  user_spp_billing_total_due_cents BIGINT,
  user_spp_billing_amount_paid_cents BIGINT,

  -- status & lifecycle
  user_spp_billing_status         VARCHAR(20) DEFAULT 'unpaid',
  user_spp_billing_status_reason  TEXT,
  user_spp_billing_is_installment BOOLEAN DEFAULT FALSE,
  user_spp_billing_due_date       DATE,
  user_spp_billing_paid_at        TIMESTAMPTZ,
  user_spp_billing_canceled_at    TIMESTAMPTZ,
  user_spp_billing_refunded_at    TIMESTAMPTZ,

  -- cicilan
  user_spp_billing_installment_count SMALLINT,
  user_spp_billing_installment_index SMALLINT,
  user_spp_billing_next_due_date     DATE,
  user_spp_billing_installment_plan  JSONB,

  -- aging & reminder
  user_spp_billing_last_status_at    TIMESTAMPTZ,
  user_spp_billing_overdue_days      INT,
  user_spp_billing_last_calculated_at TIMESTAMPTZ,
  user_spp_billing_reminder_count    INT DEFAULT 0,
  user_spp_billing_reminder_last_at  TIMESTAMPTZ,
  user_spp_billing_reminder_last_channel TEXT,

  -- metode pembayaran
  user_spp_billing_currency       VARCHAR(10) DEFAULT 'IDR',
  user_spp_billing_payment_gateway VARCHAR(50),
  user_spp_billing_payment_method  VARCHAR(50),
  user_spp_billing_va_number       VARCHAR(64),
  user_spp_billing_qr_url          TEXT,
  user_spp_billing_payment_token   TEXT,
  user_spp_billing_payment_reference TEXT,
  user_spp_billing_settlement_id   TEXT,
  user_spp_billing_receipt_url     TEXT,
  user_spp_billing_payment_attempts JSONB,

  -- komponen biaya
  user_spp_billing_fee_gateway_cents BIGINT,
  user_spp_billing_fee_platform_cents BIGINT,
  user_spp_billing_tax_cents        BIGINT,
  user_spp_billing_waiver_note      TEXT,

  -- dispute
  user_spp_billing_dispute_status   TEXT,
  user_spp_billing_dispute_opened_at TIMESTAMPTZ,
  user_spp_billing_dispute_note     TEXT,
  user_spp_billing_failed_attempts  INT DEFAULT 0,

  -- bukti & dokumen
  user_spp_billing_invoice_pdf_url TEXT,
  user_spp_billing_tax_invoice_no  VARCHAR(80),
  user_spp_billing_npwp            VARCHAR(32),
  user_spp_billing_address         TEXT,

  -- refund
  user_spp_billing_refunded_amount_cents BIGINT,
  user_spp_billing_refund_reason     TEXT,

  -- rekonsiliasi
  user_spp_billing_reconciled_at     TIMESTAMPTZ,
  user_spp_billing_reconciled_by_user_id UUID,
  user_spp_billing_reconcile_batch_id TEXT,

  -- payer info
  user_spp_billing_payer_name        VARCHAR(120),
  user_spp_billing_payer_email       VARCHAR(120),
  user_spp_billing_payer_phone       VARCHAR(40),

  -- auto debit
  user_spp_billing_auto_debit_consent BOOLEAN DEFAULT FALSE,
  user_spp_billing_auto_debit_mandate_id TEXT,

  -- catatan & metadata
  user_spp_billing_note            TEXT,
  user_spp_billing_tags            TEXT[],
  user_spp_billing_external_ref    TEXT,
  user_spp_billing_extra           JSONB,
  user_spp_billing_deleted_reason  TEXT,
  user_spp_billing_audit           JSONB,

  -- audit
  user_spp_billing_created_by_user_id UUID,
  user_spp_billing_updated_by_user_id UUID,
  user_spp_billing_deleted_by_user_id UUID,

  -- concurrency
  user_spp_billing_row_version INT DEFAULT 1,
  user_spp_billing_etag TEXT,

  -- timestamps
  user_spp_billing_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_deleted_at      TIMESTAMPTZ
);

-- =========================================================
-- TABLE: donations
-- =========================================================
DROP TABLE IF EXISTS donations CASCADE;
CREATE TABLE IF NOT EXISTS donations (
  donation_id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- donor & konteks
  donation_user_id               UUID REFERENCES users(id) ON DELETE SET NULL,
  donation_masjid_id             UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  donation_user_spp_billing_id   UUID REFERENCES user_spp_billings(user_spp_billing_id) ON DELETE SET NULL,

  -- campaign/tujuan
  donation_campaign_id           UUID,
  donation_campaign_code         VARCHAR(60),
  donation_target_type           INT,
  donation_target_id             UUID,
  donation_is_anonymous          BOOLEAN DEFAULT FALSE,

  -- identitas order
  donation_parent_order_id       VARCHAR(120),
  donation_order_id              VARCHAR(100) UNIQUE,
  donation_slug                  VARCHAR(160),
  donation_dedup_key             TEXT,
  donation_outbox_id             TEXT,

  -- nominal
  donation_currency              VARCHAR(10) DEFAULT 'IDR',
  donation_amount_cents          BIGINT NOT NULL,
  donation_min_amount_cents      BIGINT,
  donation_max_amount_cents      BIGINT,

  -- split distribusi
  donation_amount_masjid_cents           BIGINT,
  donation_amount_platform_cents         BIGINT,
  donation_amount_platform_to_masjid_cents BIGINT,
  donation_amount_platform_to_app_cents  BIGINT,

  -- status lifecycle
  donation_status                VARCHAR(20) DEFAULT 'pending',
  donation_status_reason         TEXT,
  donation_is_recurring          BOOLEAN DEFAULT FALSE,
  donation_recurrence_rule       TEXT,
  donation_next_charge_at        TIMESTAMPTZ,

  -- metode pembayaran
  donation_payment_gateway       VARCHAR(50) DEFAULT 'midtrans',
  donation_payment_method        VARCHAR(50),
  donation_payment_token         TEXT,
  donation_payment_reference     TEXT,
  donation_va_number             VARCHAR(64),
  donation_qr_url                TEXT,

  -- settlement & bukti
  donation_settlement_id         TEXT,
  donation_receipt_no            VARCHAR(80),
  donation_receipt_url           TEXT,
  donation_receipt_issued_at     TIMESTAMPTZ,
  donation_receipt_issued_by_user_id UUID,
  donation_receipt_timezone      TEXT,
  donation_receipt_locale        VARCHAR(20),

  -- waktu
  donation_paid_at               TIMESTAMPTZ,
  donation_refunded_at           TIMESTAMPTZ,

  -- informasi donor
  donation_donor_name            VARCHAR(120),
  donation_donor_email           VARCHAR(120),
  donation_donor_phone           VARCHAR(40),

  -- pesan & catatan
  donation_name                  VARCHAR(80) NOT NULL,
  donation_message               TEXT,
  donation_note_admin            TEXT,
  donation_wall_message          TEXT,

  -- honoring/tribute
  donation_in_honor_of           VARCHAR(160),
  donation_in_memory_of          VARCHAR(160),
  donation_notify_contact        JSONB,

  -- matching gift
  donation_is_matching           BOOLEAN DEFAULT FALSE,
  donation_matching_org          VARCHAR(160),
  donation_matching_amount_cents BIGINT,

  -- pledge
  donation_is_pledge             BOOLEAN DEFAULT FALSE,
  donation_pledged_at            TIMESTAMPTZ,
  donation_pledge_due_at         TIMESTAMPTZ,
  donation_fulfilled_at          TIMESTAMPTZ,

  -- anti-fraud
  donation_risk_score            NUMERIC(5,2),
  donation_risk_flags            TEXT[],

  -- webhooks
  donation_webhook_last_status   TEXT,
  donation_webhook_last_at       TIMESTAMPTZ,
  donation_webhook_payload       JSONB,

  -- akuntansi & rekonsiliasi
  donation_gl_account_code       VARCHAR(40),
  donation_cost_center_code      VARCHAR(40),
  donation_reconciled_at         TIMESTAMPTZ,
  donation_reconciled_by_user_id UUID,

  -- pelacakan kampanye
  donation_source                TEXT,
  donation_utm                   JSONB,
  donation_device_info           JSONB,
  donation_geo_info              JSONB,

  -- compliance
  donation_tax_receipt_required  BOOLEAN DEFAULT FALSE,
  donation_tax_receipt_no        VARCHAR(80),
  donation_privacy_consent       BOOLEAN DEFAULT TRUE,
  donation_data_retention_days   INT,

  -- visibilitas publik
  donation_show_public_name      BOOLEAN DEFAULT TRUE,
  donation_show_public_amount    BOOLEAN DEFAULT TRUE,

  -- FX & concurrency
  donation_fx_rate               NUMERIC(18,8),
  donation_row_version           INT DEFAULT 1,
  donation_etag                  TEXT,

  -- metadata
  donation_tags                  TEXT[],
  donation_external_ref          TEXT,
  donation_extra                 JSONB,
  donation_deleted_reason        TEXT,
  donation_audit                 JSONB,

  -- timestamps
  created_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at                     TIMESTAMPTZ,
  deleted_by_user_id             UUID
);

COMMIT;