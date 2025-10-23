-- +migrate Up
BEGIN;

-- =========================================
-- Extensions
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram ops

-- =========================================
-- Enums (idempotent)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status') THEN
    CREATE TYPE payment_status AS ENUM (
      'initiated','pending','awaiting_callback',
      'paid','partially_refunded','refunded',
      'failed','canceled','expired'
    );
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method') THEN
    CREATE TYPE payment_method AS ENUM ('gateway','bank_transfer','cash','qris','other');
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_gateway_provider') THEN
    CREATE TYPE payment_gateway_provider AS ENUM (
      'midtrans','xendit','tripay','duitku','nicepay','stripe','paypal','other'
    );
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'gateway_event_status') THEN
    CREATE TYPE gateway_event_status AS ENUM ('received','processed','ignored','duplicated','failed');
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_entry_type') THEN
    CREATE TYPE payment_entry_type AS ENUM ('charge','payment','refund','adjustment');
  END IF;
END$$;

-- =========================================
-- TABLE: payments  (ledger tunggal; TANPA UGB)
-- =========================================
CREATE TABLE IF NOT EXISTS payments (
  payment_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  payment_masjid_id            UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  payment_user_id              UUID REFERENCES users(id)          ON DELETE SET NULL,

  -- FK eksplisit (by-instance)
  payment_bill_batch_id           UUID REFERENCES bill_batches(bill_batch_id)                   ON DELETE SET NULL,
  payment_general_billing_id      UUID REFERENCES general_billings(general_billing_id)         ON DELETE SET NULL,
  payment_general_billing_kind_id UUID REFERENCES general_billing_kinds(general_billing_kind_id) ON DELETE SET NULL,

  -- nominal
  payment_amount_idr           INT NOT NULL CHECK (payment_amount_idr >= 0),
  payment_currency             VARCHAR(8) NOT NULL DEFAULT 'IDR' CHECK (payment_currency IN ('IDR')),

  -- status/metode
  payment_status               payment_status NOT NULL DEFAULT 'initiated',
  payment_method               payment_method NOT NULL DEFAULT 'gateway',

  -- info gateway (opsional jika manual)
  payment_gateway_provider     payment_gateway_provider,  -- NULL jika manual
  payment_external_id          TEXT,
  payment_gateway_reference    TEXT,
  payment_checkout_url         TEXT,
  payment_qr_string            TEXT,
  payment_signature            TEXT,
  payment_idempotency_key      TEXT,

  -- timestamps status
  payment_requested_at         TIMESTAMPTZ DEFAULT NOW(),
  payment_expires_at           TIMESTAMPTZ,
  payment_paid_at              TIMESTAMPTZ,
  payment_canceled_at          TIMESTAMPTZ,
  payment_failed_at            TIMESTAMPTZ,
  payment_refunded_at          TIMESTAMPTZ,

  -- manual ops (kasir/admin)
  payment_manual_channel       VARCHAR(32),   -- 'cash','bank_transfer','qris'
  payment_manual_reference     VARCHAR(120),
  payment_manual_received_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_at   TIMESTAMPTZ,

  -- ledger & invoice fields
  payment_entry_type           payment_entry_type NOT NULL DEFAULT 'payment',
  invoice_number               TEXT,
  invoice_due_date             DATE,
  invoice_title                TEXT,
  payment_subject_user_id      UUID REFERENCES users(id) ON DELETE SET NULL, -- subjek tagihan (user)
  payment_subject_student_id   UUID,                                         -- subjek tagihan (masjid_student)

  -- meta
  payment_description          TEXT,
  payment_note                 TEXT,
  payment_meta                 JSONB,
  payment_attachments          JSONB,

  -- audit
  payment_created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_deleted_at           TIMESTAMPTZ,

  -- Konsistensi method vs provider
  CONSTRAINT ck_payments_method_provider CHECK (
    (payment_method = 'gateway' AND payment_gateway_provider IS NOT NULL)
    OR
    (payment_method IN ('cash','bank_transfer','qris','other') AND payment_gateway_provider IS NULL)
  ),

  -- Minimal salah satu target FK eksplisit harus terisi
  CONSTRAINT ck_payment_target_any CHECK (
    payment_bill_batch_id IS NOT NULL
    OR payment_general_billing_id IS NOT NULL
    OR payment_general_billing_kind_id IS NOT NULL
  )
);

-- =========================================
-- Indexes: payments
-- =========================================

-- Idempotent DROP untuk index yang salah/typo (kalau pernah terbuat)
DROP INDEX IF EXISTS ix_payments_usb_live;
DROP INDEX IF EXISTS ix_payments_spp_header_live;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_idem_live
  ON payments (payment_masjid_id, COALESCE(payment_idempotency_key, ''))
  WHERE payment_deleted_at IS NULL AND payment_idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_provider_extid_live
  ON payments (payment_gateway_provider, COALESCE(payment_external_id,''))
  WHERE payment_deleted_at IS NULL AND payment_gateway_provider IS NOT NULL AND payment_external_id IS NOT NULL;

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

-- header indexes yang benar
CREATE INDEX IF NOT EXISTS ix_payments_bill_batch_live
  ON payments (payment_bill_batch_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_gb_header_live
  ON payments (payment_general_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_gbk_live
  ON payments (payment_general_billing_kind_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

-- ledger/invoice & subject
CREATE INDEX IF NOT EXISTS ix_payments_entrytype_live
  ON payments (payment_entry_type, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_subject_billing_live
  ON payments (payment_subject_student_id, payment_general_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

-- nomor invoice unik per tenant (untuk charge)
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoice_per_tenant
  ON payments (payment_masjid_id, LOWER(invoice_number))
  WHERE payment_deleted_at IS NULL
    AND payment_entry_type = 'charge'
    AND invoice_number IS NOT NULL;

-- anti double-charge per siswa × billing (opsional)
CREATE UNIQUE INDEX IF NOT EXISTS uq_charge_once_per_student_billing
  ON payments (payment_subject_student_id, payment_general_billing_id)
  WHERE payment_deleted_at IS NULL
    AND payment_entry_type = 'charge'
    AND payment_subject_student_id IS NOT NULL
    AND payment_general_billing_id IS NOT NULL;

-- trigram (live only)
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

-- =========================================
-- VIEW: v_invoices (invoice = baris CHARGE)
-- (Tetap fokus ke general_billing; bill_batches/student_bills bisa dibuat view terpisah jika dibutuhkan)
-- =========================================
CREATE OR REPLACE VIEW v_invoices AS
SELECT
  c.payment_id                    AS invoice_id,
  c.payment_masjid_id,
  c.payment_general_billing_id,
  c.payment_general_billing_kind_id,
  c.payment_subject_user_id,
  c.payment_subject_student_id,
  c.invoice_number,
  c.invoice_title,
  c.invoice_due_date,
  c.payment_description,
  c.payment_meta,
  c.payment_created_at           AS invoice_created_at,
  c.payment_updated_at           AS invoice_updated_at,
  c.payment_amount_idr           AS amount_idr,

  COALESCE(SUM(
    CASE
      WHEN p.payment_entry_type IN ('payment','refund')
       AND p.payment_status = 'paid' THEN p.payment_amount_idr
      ELSE 0
    END
  ),0) AS paid_idr,

  (c.payment_amount_idr
   - COALESCE(SUM(
       CASE WHEN p.payment_entry_type IN ('payment','refund')
             AND p.payment_status='paid'
            THEN p.payment_amount_idr ELSE 0 END
     ),0)
  ) AS balance_idr,

  CASE
    WHEN (c.payment_amount_idr
          - COALESCE(SUM(CASE WHEN p.payment_entry_type IN ('payment','refund')
                                 AND p.payment_status='paid'
                              THEN p.payment_amount_idr ELSE 0 END),0)
         ) <= 0
    THEN 'paid'
    WHEN c.invoice_due_date IS NOT NULL AND CURRENT_DATE > c.invoice_due_date
    THEN 'overdue'
    ELSE 'unpaid'
  END AS invoice_status

FROM payments c
LEFT JOIN payments p
  ON p.payment_subject_student_id IS NOT DISTINCT FROM c.payment_subject_student_id
 AND p.payment_general_billing_id IS NOT DISTINCT FROM c.payment_general_billing_id
 AND p.payment_entry_type IN ('payment','refund')
 AND p.payment_deleted_at IS NULL
WHERE c.payment_entry_type = 'charge'
  AND c.payment_deleted_at IS NULL
GROUP BY c.payment_id;
