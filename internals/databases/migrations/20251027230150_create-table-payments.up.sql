-- +migrate Up
BEGIN;

-- =========================================
-- Extensions (kalau belum ada)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- =========================================
-- ENUMS (mirror dengan enum di Go)
-- =========================================

-- payment_status
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status') THEN
    CREATE TYPE payment_status AS ENUM (
      'initiated',
      'pending',
      'awaiting_callback',
      'paid',
      'partially_refunded',
      'refunded',
      'failed',
      'canceled',
      'expired'
    );
  END IF;
END$$;

-- payment_method
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method') THEN
    CREATE TYPE payment_method AS ENUM (
      'gateway',
      'bank_transfer',
      'cash',
      'qris',
      'other'
    );
  END IF;
END$$;

-- payment_gateway_provider
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_gateway_provider') THEN
    CREATE TYPE payment_gateway_provider AS ENUM (
      'midtrans',
      'xendit',
      'tripay',
      'duitku',
      'nicepay',
      'stripe',
      'paypal',
      'other'
    );
  END IF;
END$$;

-- payment_entry_type
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_entry_type') THEN
    CREATE TYPE payment_entry_type AS ENUM (
      'charge',
      'payment',
      'refund',
      'adjustment'
    );
  END IF;
END$$;

-- fee_scope (kalau SUDAH ada dari migration lain, blok ini boleh dihapus)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fee_scope') THEN
    CREATE TYPE fee_scope AS ENUM (
      'tenant',
      'class_parent',
      'class',
      'section',
      'student',
      'term'
    );
  END IF;
END$$;

-- gateway_event_status (untuk payment_gateway_events)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'gateway_event_status') THEN
    CREATE TYPE gateway_event_status AS ENUM (
      'received',
      'processing',
      'success',
      'failed'
    );
  END IF;
END$$;

-- =========================================
-- TABLE: payments (HEADER transaksi / VA)
-- =========================================
CREATE TABLE IF NOT EXISTS payments (
  payment_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & actor
  payment_school_id                UUID REFERENCES schools(school_id) ON DELETE SET NULL,
  payment_user_id                  UUID REFERENCES users(id)          ON DELETE SET NULL,
  payment_number                   BIGINT,

  -- Nominal TOTAL transaksi (sum of items)
  payment_amount_idr               INT NOT NULL CHECK (payment_amount_idr >= 0),
  payment_currency                 VARCHAR(8) NOT NULL DEFAULT 'IDR'
    CHECK (payment_currency IN ('IDR')),

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

  -- Snapshot channel/bank/VA (hasil dari provider)
  payment_channel_snapshot         VARCHAR(40),
  payment_bank_snapshot            VARCHAR(80),
  payment_va_number_snapshot       VARCHAR(80),
  payment_va_name_snapshot         VARCHAR(160),

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

  -- Ledger / tipe entry
  payment_entry_type               payment_entry_type NOT NULL DEFAULT 'payment',

  -- Subjek pembayaran (payer di level user)
  payment_subject_user_id          UUID REFERENCES users(id) ON DELETE SET NULL,

  -- ===== User snapshots (payer) =====
  payment_user_name_snapshot       TEXT,
  payment_full_name_snapshot       TEXT,
  payment_email_snapshot           TEXT,
  payment_donation_name_snapshot   TEXT,

  -- Meta (header level / bundle)
  payment_description              TEXT,
  payment_note                     TEXT,
  payment_meta                     JSONB,
  payment_attachments              JSONB,

  -- Audit
  payment_created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_deleted_at               TIMESTAMPTZ,

  -- ===== CHECKs =====
  CONSTRAINT ck_payments_method_provider CHECK (
    (payment_method = 'gateway' AND payment_gateway_provider IS NOT NULL)
    OR
    (payment_method IN ('cash','bank_transfer','qris','other') AND payment_gateway_provider IS NULL)
  )
);

-- =========================================
-- Indexes: payments (HEADER)
-- =========================================

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_idem_live
  ON payments (payment_school_id, COALESCE(payment_idempotency_key, ''))
  WHERE payment_deleted_at IS NULL
    AND payment_idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_provider_extid_live
  ON payments (payment_gateway_provider, COALESCE(payment_external_id,''))
  WHERE payment_deleted_at IS NULL
    AND payment_gateway_provider IS NOT NULL
    AND payment_external_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_school_number_live
  ON payments (payment_school_id, payment_number)
  WHERE payment_deleted_at IS NULL
    AND payment_number IS NOT NULL;

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

CREATE INDEX IF NOT EXISTS ix_payments_entrytype_live
  ON payments (payment_entry_type, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

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

CREATE INDEX IF NOT EXISTS ix_payments_channel_live
  ON payments (payment_channel_snapshot, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_bank_live
  ON payments (payment_bank_snapshot, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payments_va_number_live
  ON payments (payment_va_number_snapshot, payment_created_at DESC)
  WHERE payment_deleted_at IS NULL
    AND payment_va_number_snapshot IS NOT NULL;



-- =========================================
-- TABLE: payment_items (DETAIL / SNAPSHOT per kebutuhan)
-- =========================================
CREATE TABLE IF NOT EXISTS payment_items (
  payment_item_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi ke header
  payment_item_school_id                UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,
  payment_item_payment_id               UUID NOT NULL
    REFERENCES payments(payment_id) ON DELETE CASCADE,

  -- Urutan line item dalam 1 payment
  payment_item_index                    SMALLINT NOT NULL,

  -- === Target per item ===
  payment_item_student_bill_id          UUID REFERENCES student_bills(student_bill_id) ON DELETE SET NULL,
  payment_item_general_billing_id       UUID REFERENCES general_billings(general_billing_id) ON DELETE SET NULL,
  payment_item_general_billing_kind_id  UUID REFERENCES general_billing_kinds(general_billing_kind_id) ON DELETE SET NULL,
  payment_item_bill_batch_id            UUID REFERENCES bill_batches(bill_batch_id) ON DELETE SET NULL,

  -- Subjek murid per item
  payment_item_school_student_id        UUID REFERENCES school_students(school_student_id) ON DELETE SET NULL,

  -- Context kelas/enrollment (opsional)
  payment_item_class_id                 UUID,
  payment_item_enrollment_id            UUID,

  -- Nominal per item
  payment_item_amount_idr               INT NOT NULL CHECK (payment_item_amount_idr >= 0),

  -- === Fee rule snapshots per item ===
  payment_item_fee_rule_id                     UUID REFERENCES fee_rules(fee_rule_id) ON DELETE SET NULL,
  payment_item_fee_rule_option_code_snapshot   VARCHAR(20),
  payment_item_fee_rule_option_index_snapshot  SMALLINT,
  payment_item_fee_rule_amount_snapshot        INT CHECK (payment_item_fee_rule_amount_snapshot IS NULL OR payment_item_fee_rule_amount_snapshot >= 0),
  payment_item_fee_rule_gbk_id_snapshot        UUID,
  payment_item_fee_rule_scope_snapshot         fee_scope,
  payment_item_fee_rule_note_snapshot          TEXT,

  -- === Academic term snapshots per item ===
  payment_item_academic_term_id                UUID,
  payment_item_academic_term_academic_year_cache VARCHAR(40),
  payment_item_academic_term_name_cache          VARCHAR(100),
  payment_item_academic_term_slug_cache          VARCHAR(160),
  payment_item_academic_term_angkatan_cache      VARCHAR(40),

  CONSTRAINT fk_payment_items_term_school_pair
    FOREIGN KEY (payment_item_academic_term_id, payment_item_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT,

  -- 🧾 Invoice per item
  payment_item_invoice_number      TEXT,
  payment_item_invoice_title       TEXT,
  payment_item_invoice_due_date    DATE,

  -- Line title / deskripsi buat tampilan
  payment_item_title               TEXT,
  payment_item_description         TEXT,
  payment_item_meta                JSONB,

  payment_item_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_item_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_item_deleted_at          TIMESTAMPTZ,

  CONSTRAINT ck_payment_item_fee_rule_option_index_snapshot CHECK (
    payment_item_fee_rule_option_index_snapshot IS NULL
    OR payment_item_fee_rule_option_index_snapshot >= 1
  ),

  CONSTRAINT ck_payment_item_target_any CHECK (
    payment_item_student_bill_id IS NOT NULL
    OR payment_item_general_billing_id IS NOT NULL
    OR payment_item_general_billing_kind_id IS NOT NULL
    OR payment_item_school_student_id IS NOT NULL
  )
);

-- Indexes: payment_items
CREATE INDEX IF NOT EXISTS ix_payment_items_payment_live
  ON payment_items (payment_item_payment_id, payment_item_index)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_tenant_created_live
  ON payment_items (payment_item_school_id, payment_item_created_at DESC)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_student_bill_live
  ON payment_items (payment_item_student_bill_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_gb_live
  ON payment_items (payment_item_general_billing_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_gbk_live
  ON payment_items (payment_item_general_billing_kind_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_school_student_live
  ON payment_items (payment_item_school_student_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_fee_rule_live
  ON payment_items (payment_item_fee_rule_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_term_live
  ON payment_items (payment_item_school_id, payment_item_academic_term_id)
  WHERE payment_item_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_payment_items_invoice_live
  ON payment_items (payment_item_school_id, payment_item_invoice_number)
  WHERE payment_item_deleted_at IS NULL
    AND payment_item_invoice_number IS NOT NULL;


-- =========================================
-- TABLE: payment_gateway_events (LOG WEBHOOK)
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
  WHERE gateway_event_deleted_at IS NULL
    AND gateway_event_external_id IS NOT NULL;

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
