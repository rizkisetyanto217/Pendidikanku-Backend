-- Intent/sesi pembayaran
CREATE TABLE IF NOT EXISTS payment_intents (
  payment_intent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_intent_bill_id UUID NOT NULL REFERENCES student_bills(student_bill_id) ON DELETE CASCADE,
  payment_intent_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  payment_intent_amount NUMERIC(12,2) NOT NULL CHECK (payment_intent_amount >= 0),
  payment_intent_currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
  payment_intent_provider TEXT NOT NULL DEFAULT 'midtrans',
  payment_intent_channel  TEXT,
  payment_intent_status   TEXT NOT NULL DEFAULT 'pending'
    CHECK (payment_intent_status IN ('pending','requires_action','succeeded','failed','canceled','expired')),
  payment_intent_expires_at TIMESTAMPTZ,
  payment_intent_redirect_url TEXT,
  payment_intent_client_meta JSONB,
  payment_intent_idempotency_key VARCHAR(120),
  payment_intent_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_intent_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_intent_bill   ON payment_intents(payment_intent_bill_id);
CREATE INDEX IF NOT EXISTS idx_payment_intent_masjid ON payment_intents(payment_intent_masjid_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_intent_idempotency
  ON payment_intents(payment_intent_provider, payment_intent_idempotency_key)
  WHERE payment_intent_idempotency_key IS NOT NULL;

-- Semua notifikasi/event dari Midtrans
CREATE TABLE IF NOT EXISTS payment_transactions (
  payment_transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_transaction_intent_id UUID NOT NULL REFERENCES payment_intents(payment_intent_id) ON DELETE CASCADE,
  payment_transaction_bill_id   UUID NOT NULL REFERENCES student_bills(student_bill_id) ON DELETE CASCADE,

  provider TEXT NOT NULL DEFAULT 'midtrans',
  provider_order_id VARCHAR(64) NOT NULL,
  provider_transaction_id VARCHAR(64),
  provider_payment_type TEXT,
  provider_transaction_status TEXT,
  provider_fraud_status TEXT,

  provider_gross_amount NUMERIC(12,2) NOT NULL CHECK (provider_gross_amount >= 0),
  provider_currency VARCHAR(10) NOT NULL DEFAULT 'IDR',

  provider_transaction_time TIMESTAMPTZ,
  provider_settlement_time TIMESTAMPTZ,
  provider_expiry_time TIMESTAMPTZ,

  provider_signature_key VARCHAR(200),
  provider_channel_meta JSONB,
  provider_raw_notification JSONB,
  provider_event_source TEXT,           -- notification / poll / return
  webhook_idempotency_key VARCHAR(200),

  local_handled BOOLEAN NOT NULL DEFAULT FALSE,
  local_notes TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_provider_order
  ON payment_transactions(provider, provider_order_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_webhook_idem
  ON payment_transactions(provider, webhook_idempotency_key)
  WHERE webhook_idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_txn_bill   ON payment_transactions(payment_transaction_bill_id);
CREATE INDEX IF NOT EXISTS idx_payment_txn_intent ON payment_transactions(payment_transaction_intent_id);
CREATE INDEX IF NOT EXISTS idx_payment_txn_provider_order_time
  ON payment_transactions(provider, provider_order_id, created_at DESC);
