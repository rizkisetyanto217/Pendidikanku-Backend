-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================
-- TABLE 1: fee_categories  (+ deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS fee_categories (
  fee_category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fee_category_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  fee_category_code VARCHAR(40)  NOT NULL,
  fee_category_name VARCHAR(120) NOT NULL,
  fee_category_cycle TEXT NOT NULL DEFAULT 'one_time'
    CHECK (fee_category_cycle IN ('one_time','monthly','term','ad_hoc')),
  fee_category_is_mandatory  BOOLEAN NOT NULL DEFAULT TRUE,
  fee_category_discountable  BOOLEAN NOT NULL DEFAULT TRUE,
  fee_category_refundable    BOOLEAN NOT NULL DEFAULT FALSE,
  fee_category_meta JSONB,
  fee_category_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  fee_category_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_category_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_category_deleted_at TIMESTAMPTZ
);

-- Unik per masjid hanya untuk row aktif (belum terhapus)
CREATE UNIQUE INDEX IF NOT EXISTS uq_fee_category_code_per_masjid
  ON fee_categories (fee_category_masjid_id, fee_category_code)
  WHERE fee_category_deleted_at IS NULL;

-- Query umum: by masjid + belum terhapus
CREATE INDEX IF NOT EXISTS idx_fee_category_masjid_not_deleted
  ON fee_categories (fee_category_masjid_id, fee_category_deleted_at);

-- Aktif per masjid (opsional, filter tambahan)
CREATE INDEX IF NOT EXISTS idx_fee_category_masjid_active
  ON fee_categories (fee_category_masjid_id)
  WHERE fee_category_is_active = TRUE AND fee_category_deleted_at IS NULL;

-- =========================================
-- TABLE 2: school_fee_settings  (dengan *_snapshot + deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS school_fee_settings (
  fee_setting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fee_setting_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  fee_setting_class_id   UUID,
  fee_setting_section_id UUID,
  fee_setting_category_id UUID NOT NULL REFERENCES fee_categories(fee_category_id) ON DELETE RESTRICT,

  -- snapshots dari fee_categories
  fee_setting_category_code_fee_category_snapshot  VARCHAR(40),
  fee_setting_category_name_fee_category_snapshot  VARCHAR(120),
  fee_setting_category_cycle_fee_category_snapshot TEXT,

  fee_setting_amount   NUMERIC(12,2) NOT NULL CHECK (fee_setting_amount >= 0),
  fee_setting_currency VARCHAR(10)   NOT NULL DEFAULT 'IDR',
  fee_setting_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  fee_setting_note TEXT,

  fee_setting_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_setting_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_setting_deleted_at TIMESTAMPTZ,

  CONSTRAINT ck_fee_scope CHECK (
    (fee_setting_class_id IS NOT NULL)::int +
    (fee_setting_section_id IS NOT NULL)::int IN (0,1)
  )
);

-- Unik scope aktif & belum terhapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_fee_setting_scope
  ON school_fee_settings (
    fee_setting_masjid_id,
    fee_setting_category_id,
    COALESCE(fee_setting_section_id, fee_setting_class_id)
  )
  WHERE fee_setting_is_active = TRUE
    AND fee_setting_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fee_setting_masjid_not_deleted
  ON school_fee_settings(fee_setting_masjid_id, fee_setting_deleted_at);

CREATE INDEX IF NOT EXISTS idx_fee_setting_active_by_category
  ON school_fee_settings (fee_setting_masjid_id, fee_setting_category_id)
  WHERE fee_setting_is_active = TRUE
    AND fee_setting_deleted_at IS NULL;

-- =========================================
-- TABLE 3: student_bills (dengan *_snapshot + deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS student_bills (
  student_bill_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_bill_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  student_bill_user_class_section_id UUID NOT NULL
    REFERENCES user_class_sections(user_class_section_id) ON DELETE CASCADE,

  student_bill_fee_setting_id UUID REFERENCES school_fee_settings(fee_setting_id),
  student_bill_category_id UUID REFERENCES fee_categories(fee_category_id) ON DELETE SET NULL,

  -- snapshots kategori
  student_bill_category_code_fee_category_snapshot  VARCHAR(40),
  student_bill_category_name_fee_category_snapshot  VARCHAR(120),
  student_bill_category_cycle_fee_category_snapshot TEXT,

  -- snapshots siswa/kelas (opsional)
  student_bill_name_user_snapshot                  VARCHAR(120),
  student_bill_class_name_class_snapshot           VARCHAR(80),
  student_bill_section_label_section_snapshot      VARCHAR(80),

  -- data tagihan
  student_bill_period DATE,
  student_bill_period_yyyymm INT
    GENERATED ALWAYS AS (
      EXTRACT(YEAR FROM student_bill_period)::int * 100 + EXTRACT(MONTH FROM student_bill_period)::int
    ) STORED,

  student_bill_amount   NUMERIC(12,2) NOT NULL CHECK (student_bill_amount >= 0),
  student_bill_currency VARCHAR(10)   NOT NULL DEFAULT 'IDR',

  student_bill_status TEXT NOT NULL DEFAULT 'unpaid'
    CHECK (student_bill_status IN ('unpaid','partial','paid','canceled')),

  -- denorm pembayaran (diisi backend)
  student_bill_paid_total NUMERIC(12,2) NOT NULL DEFAULT 0,
  student_bill_remaining  NUMERIC(12,2)
    GENERATED ALWAYS AS (GREATEST(student_bill_amount - student_bill_paid_total, 0)) STORED,
  student_bill_last_paid_at TIMESTAMPTZ,

  -- tambahan
  student_bill_title TEXT,
  student_bill_note  TEXT,
  student_bill_due_date DATE,
  student_bill_is_overdue BOOLEAN
    GENERATED ALWAYS AS (
      student_bill_status IN ('unpaid','partial')
      AND student_bill_due_date IS NOT NULL
      AND (CURRENT_DATE > student_bill_due_date)
    ) STORED,

  student_bill_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_deleted_at TIMESTAMPTZ
);

-- Unik hanya untuk bill yang belum terhapus + belum canceled
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_bill_unique
  ON student_bills (student_bill_user_class_section_id, student_bill_category_id, student_bill_period)
  WHERE student_bill_status <> 'canceled'
    AND student_bill_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_bill_user_not_deleted
  ON student_bills(student_bill_user_class_section_id, student_bill_deleted_at);
CREATE INDEX IF NOT EXISTS idx_student_bill_period_not_deleted
  ON student_bills(student_bill_period)
  WHERE student_bill_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_student_bill_masjid_not_deleted
  ON student_bills(student_bill_masjid_id, student_bill_deleted_at);

CREATE INDEX IF NOT EXISTS idx_bill_masjid_period_active
  ON student_bills (student_bill_masjid_id, student_bill_period, student_bill_status)
  INCLUDE (student_bill_amount, student_bill_currency, student_bill_paid_total, student_bill_remaining)
  WHERE student_bill_status <> 'canceled'
    AND student_bill_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_bill_user_due
  ON student_bills (student_bill_user_class_section_id, student_bill_category_id, student_bill_period)
  WHERE student_bill_status IN ('unpaid','partial')
    AND student_bill_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_bill_yyyymm_masjid_not_deleted
  ON student_bills (student_bill_masjid_id, student_bill_period_yyyymm)
  WHERE student_bill_deleted_at IS NULL;

-- =========================================
-- TABLE 4: student_payments (manual ATAU gateway + deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS student_payments (
  student_payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_payment_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  student_payment_bill_id   UUID NOT NULL REFERENCES student_bills(student_bill_id) ON DELETE CASCADE,

  -- snapshots kategori dari fee_categories (via bill)
  student_payment_category_code_fee_category_snapshot VARCHAR(40),
  student_payment_category_name_fee_category_snapshot VARCHAR(120),

  -- jumlah
  student_payment_amount   NUMERIC(12,2) NOT NULL CHECK (student_payment_amount > 0),
  student_payment_currency VARCHAR(10)   NOT NULL DEFAULT 'IDR',

  -- metode high-level
  student_payment_method TEXT,

  -- ================= MANUAL FIELDS (opsional) =================
  student_payment_manual_receipt_number   VARCHAR(64),
  student_payment_manual_channel          TEXT,
  student_payment_manual_reference_number VARCHAR(64),
  student_payment_manual_received_by      VARCHAR(120),
  student_payment_manual_received_at      TIMESTAMPTZ,

  -- ================= PROVIDER/GATEWAY FIELDS (opsional) =================
  student_payment_provider                      TEXT,        -- 'midtrans' (nullable kalau manual)
  student_payment_provider_order_id             VARCHAR(64),
  student_payment_provider_transaction_id       VARCHAR(64),
  student_payment_provider_payment_type         TEXT,
  student_payment_provider_fee_amount           NUMERIC(12,2),
  student_payment_provider_net_amount           NUMERIC(12,2),
  student_payment_provider_meta                 JSONB,

  -- umum
  student_payment_note   TEXT,
  student_payment_paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_payment_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_payment_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_payment_deleted_at TIMESTAMPTZ
);

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_student_payment_bill_not_deleted
  ON student_payments(student_payment_bill_id, student_payment_deleted_at);
CREATE INDEX IF NOT EXISTS idx_student_payment_masjid_not_deleted
  ON student_payments(student_payment_masjid_id, student_payment_deleted_at);
CREATE INDEX IF NOT EXISTS brin_payment_paid_at
  ON student_payments USING BRIN (student_payment_paid_at)
  WITH (pages_per_range = 32);

-- Anti duplikasi settlement gateway: hanya untuk row belum terhapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_payment_midtrans_txn
  ON student_payments (student_payment_provider, student_payment_provider_transaction_id)
  WHERE student_payment_provider IS NOT NULL
    AND student_payment_deleted_at IS NULL;