-- +migrate Up
BEGIN;

-- =========================================
-- Extensions (idempotent)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- =========================================
-- Enums (idempotent)
-- =========================================
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status') THEN
    CREATE TYPE payment_status AS ENUM (
      'initiated','pending','awaiting_callback',
      'paid','partially_refunded','refunded',
      'failed','canceled','expired'
    );
  END IF;
END$$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method') THEN
    CREATE TYPE payment_method AS ENUM ('gateway','bank_transfer','cash','qris','other');
  END IF;
END$$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_gateway_provider') THEN
    CREATE TYPE payment_gateway_provider AS ENUM (
      'midtrans','xendit','tripay','duitku','nicepay','stripe','paypal','other'
    );
  END IF;
END$$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'gateway_event_status') THEN
    CREATE TYPE gateway_event_status AS ENUM ('received','processed','ignored','duplicated','failed');
  END IF;
END$$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_entry_type') THEN
    CREATE TYPE payment_entry_type AS ENUM ('charge','payment','refund','adjustment');
  END IF;
END$$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fee_scope') THEN
    CREATE TYPE fee_scope AS ENUM ('tenant','class_parent','class','section','student','term');
  END IF;
END$$;

-- =========================================
-- TABLE: payments  (kolom snapshot => *_snapshot)
-- =========================================
CREATE TABLE IF NOT EXISTS payments (
  payment_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & actor
  payment_school_id                UUID REFERENCES schools(school_id) ON DELETE SET NULL,
  payment_user_id                  UUID REFERENCES users(id)          ON DELETE SET NULL,
  payment_number BIGINT,

  -- Target (salah satu wajib)
  payment_student_bill_id          UUID REFERENCES student_bills(student_bill_id)               ON DELETE SET NULL,
  payment_general_billing_id       UUID REFERENCES general_billings(general_billing_id)        ON DELETE SET NULL,
  payment_general_billing_kind_id  UUID REFERENCES general_billing_kinds(general_billing_kind_id) ON DELETE SET NULL,

  -- Context (opsional)
  payment_bill_batch_id            UUID REFERENCES bill_batches(bill_batch_id) ON DELETE SET NULL,

  -- Nominal
  payment_amount_idr               INT NOT NULL CHECK (payment_amount_idr >= 0),
  payment_currency                 VARCHAR(8) NOT NULL DEFAULT 'IDR' CHECK (payment_currency IN ('IDR')),

  -- Status & metode
  payment_status                   payment_status NOT NULL DEFAULT 'initiated',
  payment_method                   payment_method NOT NULL DEFAULT 'gateway',

  -- Info gateway (NULL jika manual)
  payment_gateway_provider         payment_gateway_provider,
  payment_external_id              TEXT,
  payment_gateway_reference        TEXT,
  payment_checkout_url             TEXT,
  payment_qr_string                TEXT,
  payment_signature                TEXT,
  payment_idempotency_key          TEXT,

  -- Timestamps status
  payment_requested_at             TIMESTAMPTZ DEFAULT NOW(),
  payment_expires_at               TIMESTAMPTZ,
  payment_paid_at                  TIMESTAMPTZ,
  payment_canceled_at              TIMESTAMPTZ,
  payment_failed_at                TIMESTAMPTZ,
  payment_refunded_at              TIMESTAMPTZ,

  -- Manual ops (kasir/admin)
  payment_manual_channel           VARCHAR(32),
  payment_manual_reference         VARCHAR(120),
  payment_manual_received_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_manual_verified_at       TIMESTAMPTZ,

  -- Ledger & invoice
  payment_entry_type               payment_entry_type NOT NULL DEFAULT 'payment',
  payment_invoice_number           TEXT,
  payment_invoice_due_date         DATE,
  payment_invoice_title            TEXT,

  -- Subjek (opsional)
  payment_subject_user_id          UUID REFERENCES users(id) ON DELETE SET NULL,
  payment_subject_student_id       UUID REFERENCES school_students(school_student_id) ON DELETE SET NULL,

  -- ===== Fee rule snapshots =====
  payment_fee_rule_id                     UUID REFERENCES fee_rules(fee_rule_id) ON DELETE SET NULL,
  payment_fee_rule_option_code_snapshot   VARCHAR(20),
  payment_fee_rule_option_index_snapshot  SMALLINT,
  payment_fee_rule_amount_snapshot        INT CHECK (payment_fee_rule_amount_snapshot IS NULL OR payment_fee_rule_amount_snapshot >= 0),
  payment_fee_rule_gbk_id_snapshot        UUID,
  payment_fee_rule_scope_snapshot         fee_scope,
  payment_fee_rule_note_snapshot          TEXT,

  -- ===== User snapshots (payer) =====
  payment_user_name_snapshot       TEXT,   -- users.user_name
  payment_full_name_snapshot       TEXT,   -- users.full_name (atau fallback profile)
  payment_email_snapshot           TEXT,   -- users.email
  payment_donation_name_snapshot   TEXT,   -- user_profiles.user_profile_donation_name

  -- Meta
  payment_description              TEXT,
  payment_note                     TEXT,
  payment_meta                     JSONB,
  payment_attachments              JSONB,

  -- Audit
  payment_created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_deleted_at               TIMESTAMPTZ,

  -- CHECKs
  CONSTRAINT ck_payments_method_provider CHECK (
    (payment_method = 'gateway' AND payment_gateway_provider IS NOT NULL)
    OR
    (payment_method IN ('cash','bank_transfer','qris','other') AND payment_gateway_provider IS NULL)
  ),
  CONSTRAINT ck_payment_target_any CHECK (
    payment_student_bill_id IS NOT NULL
    OR payment_general_billing_id IS NOT NULL
    OR payment_general_billing_kind_id IS NOT NULL
  ),
  CONSTRAINT ck_payment_fee_rule_option_index_snapshot CHECK (
    payment_fee_rule_option_index_snapshot IS NULL OR payment_fee_rule_option_index_snapshot >= 1
  )
);

-- =========================================
-- Upgrade helpers (rename kolom lama → *_snapshot & add if missing)
-- =========================================
DO $$
BEGIN
  -- fee_rule option code/index → *_snapshot
  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='payments' AND column_name='payment_fee_rule_option_code')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='payments' AND column_name='payment_fee_rule_option_code_snapshot')
  THEN
    EXECUTE 'ALTER TABLE payments RENAME COLUMN payment_fee_rule_option_code TO payment_fee_rule_option_code_snapshot';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='payments' AND column_name='payment_fee_rule_option_index')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='payments' AND column_name='payment_fee_rule_option_index_snapshot')
  THEN
    EXECUTE 'ALTER TABLE payments RENAME COLUMN payment_fee_rule_option_index TO payment_fee_rule_option_index_snapshot';
  END IF;

  -- Tambah kolom snapshot user bila belum ada
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_name='payments' AND column_name='payment_user_name_snapshot') THEN
    EXECUTE 'ALTER TABLE payments ADD COLUMN payment_user_name_snapshot TEXT';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_name='payments' AND column_name='payment_full_name_snapshot') THEN
    EXECUTE 'ALTER TABLE payments ADD COLUMN payment_full_name_snapshot TEXT';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_name='payments' AND column_name='payment_email_snapshot') THEN
    EXECUTE 'ALTER TABLE payments ADD COLUMN payment_email_snapshot TEXT';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_name='payments' AND column_name='payment_donation_name_snapshot') THEN
    EXECUTE 'ALTER TABLE payments ADD COLUMN payment_donation_name_snapshot TEXT';
  END IF;
END$$;

-- =========================================
-- Indexes (idempotent)
-- =========================================
-- Bersih-bersih index lama yang tidak dipakai
DROP INDEX IF EXISTS ix_payments_usb_live;
DROP INDEX IF EXISTS ix_payments_spp_header_live;

-- Idempotency per tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_idem_live
  ON payments (payment_school_id, COALESCE(payment_idempotency_key, ''))
  WHERE payment_deleted_at IS NULL AND payment_idempotency_key IS NOT NULL;

-- Unique order_id per provider (aktif)
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_provider_extid_live
  ON payments (payment_gateway_provider, COALESCE(payment_external_id,''))
  WHERE payment_deleted_at IS NULL
    AND payment_gateway_provider IS NOT NULL
    AND payment_external_id IS NOT NULL;

-- Umum
CREATE INDEX IF NOT EXISTS ix_payments_tenant_created_live
  ON payments (payment_school_id, payment_created_at DESC)
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

CREATE INDEX IF NOT EXISTS ix_payments_student_bill_live
  ON payments (payment_student_bill_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_bill_batch_live
  ON payments (payment_bill_batch_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_gb_header_live
  ON payments (payment_general_billing_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_gbk_live
  ON payments (payment_general_billing_kind_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_entrytype_live
  ON payments (payment_entry_type, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_subject_student_live
  ON payments (payment_subject_student_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

-- Fuzzy search helpers
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

-- Fee rule snapshot indexes
CREATE INDEX IF NOT EXISTS ix_payments_fee_rule_live
  ON payments (payment_fee_rule_id, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_fee_rule_option_code_snapshot_live
  ON payments (LOWER(payment_fee_rule_option_code_snapshot), payment_created_at DESC)
  WHERE payment_deleted_at IS NULL AND payment_fee_rule_option_code_snapshot IS NOT NULL;

-- Snapshot username lookup cepat
CREATE INDEX IF NOT EXISTS ix_payments_user_name_snapshot_live
  ON payments (LOWER(payment_user_name_snapshot))
  WHERE payment_deleted_at IS NULL;

  -- Unique per sekolah (hanya untuk row yang masih live)
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_school_number_live
  ON payments (payment_school_id, payment_number)
  WHERE payment_deleted_at IS NULL
    AND payment_number IS NOT NULL;

-- =========================================
-- TABLE: payment_gateway_events
-- =========================================
CREATE TABLE IF NOT EXISTS payment_gateway_events (
  gateway_event_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  gateway_event_school_id     UUID REFERENCES schools(school_id) ON DELETE SET NULL,
  gateway_event_payment_id    UUID REFERENCES payments(payment_id) ON DELETE SET NULL,

  gateway_event_provider      payment_gateway_provider NOT NULL,
  gateway_event_type          TEXT,
  gateway_event_external_id   TEXT,
  gateway_event_external_ref  TEXT,

  gateway_event_headers       JSONB,
  gateway_event_payload       JSONB,
  gateway_event_signature     TEXT,
  gateway_event_raw_query     TEXT,

  gateway_event_status        gateway_event_status NOT NULL DEFAULT 'received',
  gateway_event_error         TEXT,
  gateway_event_try_count     INT NOT NULL DEFAULT 0 CHECK (gateway_event_try_count >= 0),

  gateway_event_received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_processed_at  TIMESTAMPTZ,

  gateway_event_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  gateway_event_deleted_at    TIMESTAMPTZ
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

CREATE INDEX IF NOT EXISTS ix_gw_events_school_live
  ON payment_gateway_events (gateway_event_school_id, gateway_event_received_at DESC)
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
