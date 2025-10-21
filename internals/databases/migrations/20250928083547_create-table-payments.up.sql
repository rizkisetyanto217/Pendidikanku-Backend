-- +migrate Up
BEGIN;

-- =========================================
-- Extensions (idempotent)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- Enums (idempotent)
-- =========================================
DO $$ BEGIN
  CREATE TYPE payment_status AS ENUM (
    'initiated','pending','awaiting_callback',
    'paid','partially_refunded','refunded',
    'failed','canceled','expired'
  );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE payment_method AS ENUM ('gateway','bank_transfer','cash','qris','other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE payment_gateway_provider AS ENUM (
    'midtrans','xendit','tripay','duitku','nicepay','stripe','paypal','other'
  );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE gateway_event_status AS ENUM ('received','processed','ignored','duplicated','failed');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================
-- TABLE: payments  (sudah support FK eksplisit + subject polimorfik)
-- =========================================
CREATE TABLE IF NOT EXISTS payments (
  payment_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  payment_masjid_id           UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  payment_user_id             UUID REFERENCES users(id)          ON DELETE SET NULL,

  -- ===== FK EKSPILISIT (by-instance) =====
  -- target billing (SPP atau General)
  payment_user_spp_billing_id     UUID REFERENCES user_spp_billings(user_spp_billing_id) ON DELETE SET NULL,
  payment_spp_billing_id          UUID REFERENCES spp_billings(spp_billing_id)           ON DELETE SET NULL,
  payment_user_general_billing_id UUID REFERENCES user_general_billings(user_general_billing_id) ON DELETE SET NULL,
  payment_general_billing_id      UUID REFERENCES general_billings(general_billing_id)           ON DELETE SET NULL,

  -- ===== SUBJECT POLIMORFIK (tanpa instance) =====
  -- contoh: general_billing_kind (campaign global/per-masjid), user_subscription
  payment_subject_type        VARCHAR(40),
  payment_subject_ref_id      UUID,

  -- nominal
  payment_amount_idr          INT NOT NULL CHECK (payment_amount_idr >= 0),
  payment_currency            VARCHAR(8) NOT NULL DEFAULT 'IDR' CHECK (payment_currency IN ('IDR')),

  -- status/metode
  payment_status              payment_status NOT NULL DEFAULT 'initiated',
  payment_method              payment_method NOT NULL DEFAULT 'gateway',

  -- info gateway (opsional jika manual)
  payment_gateway_provider    payment_gateway_provider,  -- NULL jika manual
  payment_external_id         TEXT,
  payment_gateway_reference   TEXT,
  payment_checkout_url        TEXT,
  payment_qr_string           TEXT,
  payment_signature           TEXT,
  payment_idempotency_key     TEXT,

  -- timestamps status
  payment_requested_at        TIMESTAMPTZ DEFAULT NOW(),
  payment_expires_at          TIMESTAMPTZ,
  payment_paid_at             TIMESTAMPTZ,
  payment_canceled_at         TIMESTAMPTZ,
  payment_failed_at           TIMESTAMPTZ,
  payment_refunded_at         TIMESTAMPTZ,

  -- manual ops (kasir/admin)
  payment_manual_channel      VARCHAR(32),   -- 'cash','bank_transfer','qris'
  payment_manual_reference    VARCHAR(120),
  payment_manual_received_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_at  TIMESTAMPTZ,

  -- meta
  payment_description         TEXT,
  payment_note                TEXT,
  payment_meta                JSONB,
  payment_attachments         JSONB,

  -- audit
  payment_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_deleted_at          TIMESTAMPTZ,

  -- Konsistensi method vs provider
  CONSTRAINT ck_payments_method_provider CHECK (
    (payment_method = 'gateway' AND payment_gateway_provider IS NOT NULL)
    OR
    (payment_method IN ('cash','bank_transfer','qris','other') AND payment_gateway_provider IS NULL)
  ),

  -- Whitelist subject type polimorfik
  CONSTRAINT chk_payment_subject_type CHECK (
    payment_subject_type IS NULL
    OR payment_subject_type IN ('general_billing_kind','general_billing','user_subscription')
  ),

  -- Pilih salah satu: (FK eksplisit) XOR (subject polimorfik)
  CONSTRAINT ck_payment_target_xor CHECK (
    (
      (payment_user_spp_billing_id IS NOT NULL OR payment_spp_billing_id IS NOT NULL
       OR payment_user_general_billing_id IS NOT NULL OR payment_general_billing_id IS NOT NULL)
      AND payment_subject_type IS NULL AND payment_subject_ref_id IS NULL
    )
    OR
    (
      (payment_user_spp_billing_id IS NULL AND payment_spp_billing_id IS NULL
       AND payment_user_general_billing_id IS NULL AND payment_general_billing_id IS NULL)
      AND payment_subject_type IS NOT NULL AND payment_subject_ref_id IS NOT NULL
    )
  )
);

-- Indexes: payments
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_idem_live
  ON payments (payment_masjid_id, COALESCE(payment_idempotency_key, ''))
  WHERE payment_deleted_at IS NULL AND payment_idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_provider_extid_live
  ON payments (payment_gateway_provider, COALESCE(payment_external_id,''))
  WHERE payment_deleted_at IS NULL AND payment_gateway_provider IS NOT NULL AND payment_external_id IS NOT NULL;

-- NOTE: kalau ingin 1x paid saja per user billing, aktifkan dua index unik di bawah ini
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_paid_once_per_usb_live
--   ON payments (payment_user_spp_billing_id)
--   WHERE payment_deleted_at IS NULL AND payment_status = 'paid';
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_paid_once_per_ugb_live
--   ON payments (payment_user_general_billing_id)
--   WHERE payment_deleted_at IS NULL AND payment_status = 'paid';

CREATE INDEX IF NOT EXISTS ix_payments_tenant_created_live
  ON payments (payment_masjid_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_status_live
  ON payments (payment_status, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_provider_live
  ON payments (payment_gateway_provider, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_user_live
  ON payments (payment_user_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_usb_live
  ON payments (payment_user_spp_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_ugb_live
  ON payments (payment_user_general_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_spp_header_live
  ON payments (payment_spp_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_gb_header_live
  ON payments (payment_general_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

-- Lookup subject polimorfik
CREATE INDEX IF NOT EXISTS ix_payments_subject
  ON payments (payment_subject_type, payment_subject_ref_id);

-- GIN trigram (live only)
CREATE INDEX IF NOT EXISTS gin_payments_extid_trgm_live
  ON payments USING GIN ( (COALESCE(payment_external_id,'')) gin_trgm_ops )
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_payments_gwref_trgm_live
  ON payments USING GIN ( (COALESCE(payment_gateway_reference,'')) gin_trgm_ops )
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_payments_manualref_trgm_live
  ON payments USING GIN ( (COALESCE(payment_manual_reference,'')) gin_trgm_ops )
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_payments_desc_trgm_live
  ON payments USING GIN ( (COALESCE(payment_description,'')) gin_trgm_ops )
  WHERE payment_deleted_at IS NULL;



-- =========================================
-- TABLE: payment_gateway_events
-- =========================================
CREATE TABLE IF NOT EXISTS payment_gateway_events (
  gateway_event_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  gateway_event_masjid_id   UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  gateway_event_payment_id  UUID REFERENCES payments(payment_id) ON DELETE SET NULL,

  gateway_event_provider    payment_gateway_provider NOT NULL,
  gateway_event_type        TEXT,
  gateway_event_external_id   TEXT,
  gateway_event_external_ref  TEXT,

  gateway_event_headers     JSONB,
  gateway_event_payload     JSONB,
  gateway_event_signature   TEXT,
  gateway_event_raw_query   TEXT,

  gateway_event_status      gateway_event_status NOT NULL DEFAULT 'received',
  gateway_event_error       TEXT,
  gateway_event_try_count   INT NOT NULL DEFAULT 0 CHECK (gateway_event_try_count >= 0),

  gateway_event_received_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_processed_at TIMESTAMPTZ,

  gateway_event_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_deleted_at  TIMESTAMPTZ
);

-- Indexes: payment_gateway_events
CREATE UNIQUE INDEX IF NOT EXISTS uq_gw_event_provider_extid_live
  ON payment_gateway_events (gateway_event_provider, COALESCE(gateway_event_external_id,''))
  WHERE gateway_event_deleted_at IS NULL AND gateway_event_external_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_gw_events_provider_status_live
  ON payment_gateway_events (gateway_event_provider, gateway_event_status, gateway_event_received_at DESC)
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gw_events_payment_live
  ON payment_gateway_events (gateway_event_payment_id, gateway_event_received_at DESC)
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gw_events_masjid_live
  ON payment_gateway_events (gateway_event_masjid_id, gateway_event_received_at DESC)
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_gw_events_payload_live
  ON payment_gateway_events USING GIN (gateway_event_payload)
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_gw_events_headers_live
  ON payment_gateway_events USING GIN (gateway_event_headers)
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_gw_events_extref_trgm_live
  ON payment_gateway_events USING GIN ( (COALESCE(gateway_event_external_ref,'')) gin_trgm_ops )
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_gw_events_type_trgm_live
  ON payment_gateway_events USING GIN ( (COALESCE(gateway_event_type,'')) gin_trgm_ops )
  WHERE gateway_event_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_gw_events_sig_trgm_live
  ON payment_gateway_events USING GIN ( (COALESCE(gateway_event_signature,'')) gin_trgm_ops )
  WHERE gateway_event_deleted_at IS NULL;

COMMIT;
