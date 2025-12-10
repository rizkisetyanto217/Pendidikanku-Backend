-- +migrate Up
BEGIN;

-- =========================================
-- EXTENSIONS (idempotent)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops
CREATE EXTENSION IF NOT EXISTS btree_gist; -- EXCLUDE
CREATE EXTENSION IF NOT EXISTS unaccent;   -- search

-- =========================================
-- ENUMS (idempotent)
-- =========================================
DO $$ BEGIN
  CREATE TYPE fee_scope AS ENUM ('tenant','class_parent','class','section','student','term');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE general_billing_category AS ENUM ('registration','spp','mass_student','donation');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- TABLE: general_billings (header/event tagihan per sekolah)
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billings (
  general_billing_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  general_billing_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- kategori + kode jenis (SPP/UNIFORM/REG/DONASI dll.)
  general_billing_category   general_billing_category NOT NULL,
  general_billing_bill_code  VARCHAR(60) NOT NULL DEFAULT 'SPP',

  -- kode unik opsional (human friendly)
  general_billing_code       VARCHAR(60),

  general_billing_title      TEXT NOT NULL,
  general_billing_desc       TEXT,

  -- cakupan akademik (opsional)
  general_billing_class_id   UUID REFERENCES classes(class_id) ON DELETE SET NULL,
  general_billing_section_id UUID REFERENCES class_sections(class_section_id) ON DELETE SET NULL,
  general_billing_term_id    UUID REFERENCES academic_terms(academic_term_id) ON DELETE SET NULL,

  -- periode (opsional, penting untuk SPP)
  general_billing_month      SMALLINT,
  general_billing_year       SMALLINT,

  general_billing_due_date   DATE,
  general_billing_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_default_amount_idr INT CHECK (general_billing_default_amount_idr >= 0),

  general_billing_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_deleted_at TIMESTAMPTZ
);

-- INDEXES: general_billings
CREATE UNIQUE INDEX IF NOT EXISTS uq_general_billings_code_per_tenant_alive
  ON general_billings (general_billing_school_id, LOWER(general_billing_code))
  WHERE general_billing_deleted_at IS NULL AND general_billing_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_gb_tenant_cat_active_created
  ON general_billings (
    general_billing_school_id,
    general_billing_category,
    general_billing_is_active,
    general_billing_created_at DESC
  )
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_due_alive
  ON general_billings (general_billing_due_date)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_term_alive
  ON general_billings (general_billing_term_id)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_month_year_alive
  ON general_billings (general_billing_school_id, general_billing_year, general_billing_month)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_billcode_alive
  ON general_billings (general_billing_school_id, general_billing_bill_code)
  WHERE general_billing_deleted_at IS NULL;

-- =========================================================
-- TABLE: user_general_billings
--   Tagihan per user/siswa untuk general_billings
-- =========================================================
CREATE TABLE IF NOT EXISTS user_general_billings (
  user_general_billing_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_general_billing_school_id          UUID NOT NULL,

  -- relasi ke siswa (opsional) — composite FK (id, school_id)
  user_general_billing_school_student_id  UUID,
  CONSTRAINT fk_ugb_student_tenant FOREIGN KEY (user_general_billing_school_student_id, user_general_billing_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- payer (opsional)
  user_general_billing_payer_user_id      UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- referensi ke general_billing (wajib)
  user_general_billing_billing_id         UUID NOT NULL
    REFERENCES general_billings(general_billing_id) ON DELETE CASCADE,

  -- nilai & status
  user_general_billing_amount_idr         INT NOT NULL CHECK (user_general_billing_amount_idr >= 0),
  user_general_billing_status             VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                                          CHECK (user_general_billing_status IN ('unpaid','paid','canceled')),
  user_general_billing_paid_at            TIMESTAMPTZ,
  user_general_billing_note               TEXT,

  -- snapshots ringan
  user_general_billing_title_snapshot     TEXT,
  user_general_billing_category_snapshot  general_billing_category,
  user_general_billing_bill_code_snapshot VARCHAR(60),

  -- metadata fleksibel
  user_general_billing_meta               JSONB DEFAULT '{}'::jsonb,

  -- timestamps (soft delete)
  user_general_billing_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_deleted_at         TIMESTAMPTZ
);

-- INDEXES: user_general_billings
-- Unik per student untuk satu billing (abaikan baris yang soft-deleted)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ugb_per_student_alive
  ON user_general_billings (user_general_billing_billing_id, user_general_billing_school_student_id)
  WHERE user_general_billing_deleted_at IS NULL
    AND user_general_billing_school_student_id IS NOT NULL;

-- Unik per payer untuk satu billing (abaikan baris yang soft-deleted)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ugb_per_payer_alive
  ON user_general_billings (user_general_billing_billing_id, user_general_billing_payer_user_id)
  WHERE user_general_billing_deleted_at IS NULL
    AND user_general_billing_payer_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_school_alive
  ON user_general_billings (user_general_billing_school_id)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_billing_alive
  ON user_general_billings (user_general_billing_billing_id)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_status_alive
  ON user_general_billings (user_general_billing_status)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_created_at_alive
  ON user_general_billings (user_general_billing_created_at DESC)
  WHERE user_general_billing_deleted_at IS NULL;

-- =========================================================
-- TABLE: fee_rules (JSONB opsi harga) — TANPA kinds
-- =========================================================
CREATE TABLE IF NOT EXISTS fee_rules (
  fee_rule_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fee_rule_school_id         UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  fee_rule_scope             fee_scope NOT NULL,
  fee_rule_class_parent_id   UUID,
  fee_rule_class_id          UUID,
  fee_rule_section_id        UUID,
  fee_rule_school_student_id UUID,

  -- Periode (salah satu: term_id ATAU year+month)
  fee_rule_term_id           UUID,
  fee_rule_month             SMALLINT,
  fee_rule_year              SMALLINT,

  -- Jenis rule (kategori + bill_code)
  fee_rule_category          general_billing_category NOT NULL,
  fee_rule_bill_code         VARCHAR(60) NOT NULL DEFAULT 'SPP',

  -- Opsi/label default (single, denorm untuk penanda)
  fee_rule_option_code       VARCHAR(20) NOT NULL DEFAULT 'T1',
  fee_rule_option_label      VARCHAR(60),
  fee_rule_is_default        BOOLEAN NOT NULL DEFAULT FALSE,

  -- JSONB daftar opsi harga: [{code,label,amount}, ...]
  fee_rule_amount_options    JSONB NOT NULL,

  -- Effective window
  fee_rule_effective_from    DATE,
  fee_rule_effective_to      DATE,

  fee_rule_note              TEXT,

  fee_rule_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_rule_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_rule_deleted_at        TIMESTAMPTZ,

  -- TENANT-SAFE FK ke academic_terms
  CONSTRAINT fk_fee_rule_term_tenant
    FOREIGN KEY (fee_rule_term_id, fee_rule_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- CHECK minimal (tanpa subquery)
  CONSTRAINT chk_fee_rule_amounts_json_array
    CHECK (
      jsonb_typeof(fee_rule_amount_options) = 'array'
      AND jsonb_array_length(fee_rule_amount_options) >= 1
    )
);

-- INDEXES: fee_rules
CREATE INDEX IF NOT EXISTS idx_fee_rules_tenant_scope
  ON fee_rules (fee_rule_school_id, fee_rule_scope);

CREATE INDEX IF NOT EXISTS idx_fee_rules_term
  ON fee_rules (fee_rule_term_id);

CREATE INDEX IF NOT EXISTS idx_fee_rules_month_year
  ON fee_rules (fee_rule_year, fee_rule_month);

CREATE INDEX IF NOT EXISTS ix_fee_rules_amount_options_gin
  ON fee_rules USING GIN (fee_rule_amount_options jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_fee_rules_option_code
  ON fee_rules (LOWER(fee_rule_option_code));

CREATE INDEX IF NOT EXISTS idx_fee_rules_is_default
  ON fee_rules (fee_rule_is_default);

CREATE INDEX IF NOT EXISTS ix_fee_rules_billcode
  ON fee_rules (fee_rule_bill_code, fee_rule_scope);

CREATE INDEX IF NOT EXISTS ix_fee_rules_category
  ON fee_rules (fee_rule_category);

-- =========================================================
-- TABLE: bill_batches (generik; class/section XOR) — TANPA kinds
-- =========================================================
CREATE TABLE IF NOT EXISTS bill_batches (
  bill_batch_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  bill_batch_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  bill_batch_class_id   UUID,
  bill_batch_section_id UUID,

  -- Periode (untuk recurring seperti SPP)
  bill_batch_month      SMALLINT CHECK (bill_batch_month BETWEEN 1 AND 12),
  bill_batch_year       SMALLINT CHECK (bill_batch_year BETWEEN 2000 AND 2100),
  bill_batch_term_id    UUID,

  -- Kategori + kode + option untuk one-off
  bill_batch_category    general_billing_category NOT NULL,
  bill_batch_bill_code   VARCHAR(60) NOT NULL DEFAULT 'SPP',
  bill_batch_option_code VARCHAR(60),

  bill_batch_title      TEXT NOT NULL,
  bill_batch_due_date   DATE,
  bill_batch_note       TEXT,

  -- Denorm totals
  bill_batch_total_amount_idr    INT NOT NULL DEFAULT 0,
  bill_batch_total_paid_idr      INT NOT NULL DEFAULT 0,
  bill_batch_total_students      INT NOT NULL DEFAULT 0,
  bill_batch_total_students_paid INT NOT NULL DEFAULT 0,

  bill_batch_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  bill_batch_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  bill_batch_deleted_at TIMESTAMPTZ,

  -- TENANT-SAFE FK komposit
  CONSTRAINT fk_bill_batch_class_tenant
    FOREIGN KEY (bill_batch_class_id, bill_batch_school_id)
    REFERENCES classes (class_id, class_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_bill_batch_section_tenant
    FOREIGN KEY (bill_batch_section_id, bill_batch_school_id)
    REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_bill_batch_term_tenant
    FOREIGN KEY (bill_batch_term_id, bill_batch_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT ck_bill_batches_xor_class_section
    CHECK (
      (bill_batch_class_id IS NOT NULL AND bill_batch_section_id IS NULL)
      OR
      (bill_batch_class_id IS NULL AND bill_batch_section_id IS NOT NULL)
    )
);

-- INDEXES: bill_batches
CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_periodic_section
  ON bill_batches (
    bill_batch_school_id,
    bill_batch_bill_code,
    bill_batch_section_id,
    bill_batch_term_id,
    bill_batch_year,
    bill_batch_month
  )
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_section_id IS NOT NULL
    AND bill_batch_option_code IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_periodic_class
  ON bill_batches (
    bill_batch_school_id,
    bill_batch_bill_code,
    bill_batch_class_id,
    bill_batch_term_id,
    bill_batch_year,
    bill_batch_month
  )
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_class_id IS NOT NULL
    AND bill_batch_option_code IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_oneoff_section
  ON bill_batches (
    bill_batch_school_id,
    bill_batch_bill_code,
    bill_batch_section_id,
    bill_batch_term_id,
    bill_batch_option_code
  )
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_section_id IS NOT NULL
    AND bill_batch_option_code IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_oneoff_class
  ON bill_batches (
    bill_batch_school_id,
    bill_batch_bill_code,
    bill_batch_class_id,
    bill_batch_term_id,
    bill_batch_option_code
  )
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_class_id IS NOT NULL
    AND bill_batch_option_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_tenant_ym_alive
  ON bill_batches (bill_batch_school_id, bill_batch_year, bill_batch_month)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_due_date_alive
  ON bill_batches (bill_batch_due_date)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_school_created_at_alive
  ON bill_batches (bill_batch_school_id, bill_batch_created_at DESC)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_school_updated_at_alive
  ON bill_batches (bill_batch_school_id, bill_batch_updated_at DESC)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_term_alive
  ON bill_batches (bill_batch_term_id)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_school_term_alive
  ON bill_batches (bill_batch_school_id, bill_batch_term_id)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_class_alive
  ON bill_batches (bill_batch_class_id)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_section_alive
  ON bill_batches (bill_batch_section_id)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_school_due_date_alive
  ON bill_batches (bill_batch_school_id, bill_batch_due_date)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_bill_batches_title_trgm_alive
  ON bill_batches USING GIN (LOWER(bill_batch_title) gin_trgm_ops)
  WHERE bill_batch_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_batch_category
  ON bill_batches (bill_batch_category);

COMMIT;
