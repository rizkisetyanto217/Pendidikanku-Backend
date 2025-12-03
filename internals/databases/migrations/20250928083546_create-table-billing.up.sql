-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops
CREATE EXTENSION IF NOT EXISTS btree_gist; -- EXCLUDE
CREATE EXTENSION IF NOT EXISTS unaccent;   -- pencarian

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
DO $$ BEGIN
  CREATE TYPE fee_scope AS ENUM ('tenant','class_parent','class','section','student','term');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE general_billing_kind_category AS ENUM ('registration','spp','mass_student','donation');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- TABLE: general_billing_kinds (katalog; bisa GLOBAL/tenant)
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billing_kinds (
  general_billing_kind_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  general_billing_kind_school_id          UUID REFERENCES schools(school_id) ON DELETE CASCADE, -- NULL = GLOBAL

  general_billing_kind_code               VARCHAR(60) NOT NULL,
  general_billing_kind_name               TEXT NOT NULL,
  general_billing_kind_desc               TEXT,
  general_billing_kind_is_active          BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_kind_default_amount_idr INT,

  -- enum kategori
  general_billing_kind_category           general_billing_kind_category NOT NULL DEFAULT 'mass_student',
  general_billing_kind_is_global          BOOLEAN NOT NULL DEFAULT FALSE,
  general_billing_kind_visibility         VARCHAR(20),

  -- flags (logika back-end)
  general_billing_kind_is_recurring            BOOLEAN NOT NULL DEFAULT FALSE,
  general_billing_kind_requires_month_year     BOOLEAN NOT NULL DEFAULT FALSE,
  general_billing_kind_requires_option_code    BOOLEAN NOT NULL DEFAULT FALSE,

  general_billing_kind_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_kind_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_kind_deleted_at TIMESTAMPTZ,

  CONSTRAINT ck_gbk_flags_match_category CHECK (
    CASE general_billing_kind_category
      WHEN 'registration' THEN
        general_billing_kind_is_recurring = FALSE
        AND general_billing_kind_requires_month_year = FALSE
        AND general_billing_kind_requires_option_code = FALSE
      WHEN 'spp' THEN
        general_billing_kind_is_recurring = TRUE
        AND general_billing_kind_requires_month_year = TRUE
        AND general_billing_kind_requires_option_code = FALSE
      WHEN 'mass_student' THEN
        general_billing_kind_is_recurring = FALSE
        AND general_billing_kind_requires_month_year = FALSE
        AND general_billing_kind_requires_option_code = TRUE
      WHEN 'donation' THEN
        general_billing_kind_is_recurring = FALSE
        AND general_billing_kind_requires_month_year = FALSE
        AND (general_billing_kind_requires_option_code = FALSE)
      ELSE TRUE
    END
  )
);

-- =========================
-- INDEXES: general_billing_kinds
-- =========================
CREATE UNIQUE INDEX IF NOT EXISTS uq_gbk_code_per_tenant_alive
  ON general_billing_kinds (general_billing_kind_school_id, LOWER(general_billing_kind_code))
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_gbk_code_global_alive
  ON general_billing_kinds (LOWER(general_billing_kind_code))
  WHERE general_billing_kind_deleted_at IS NULL
    AND general_billing_kind_school_id IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_tenant_active
  ON general_billing_kinds (general_billing_kind_school_id, general_billing_kind_is_active)
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_created_at_alive
  ON general_billing_kinds (general_billing_kind_created_at DESC)
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_category_global_alive
  ON general_billing_kinds (general_billing_kind_category, general_billing_kind_is_global)
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_visibility_alive
  ON general_billing_kinds (general_billing_kind_visibility)
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_flags_alive
  ON general_billing_kinds (
    general_billing_kind_is_recurring,
    general_billing_kind_requires_month_year,
    general_billing_kind_requires_option_code
  )
  WHERE general_billing_kind_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_gbk_code_trgm
  ON general_billing_kinds USING GIN (general_billing_kind_code gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_gbk_name_trgm
  ON general_billing_kinds USING GIN (general_billing_kind_name gin_trgm_ops);

-- =========================================================
-- TABLE: fee_rules (JSONB opsi harga) — tanpa subquery di CHECK
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

  -- Jenis rule (link ke katalog + denorm code)
  fee_rule_general_billing_kind_id UUID
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
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

  -- SNAPSHOT GBK (diisi backend)
  fee_rule_gbk_code_snapshot                 VARCHAR(60),
  fee_rule_gbk_name_snapshot                 TEXT,
  fee_rule_gbk_category_snapshot             VARCHAR(20),
  fee_rule_gbk_is_global_snapshot            BOOLEAN,
  fee_rule_gbk_visibility_snapshot           VARCHAR(20),
  fee_rule_gbk_is_recurring_snapshot         BOOLEAN,
  fee_rule_gbk_requires_month_year_snapshot  BOOLEAN,
  fee_rule_gbk_requires_option_code_snapshot BOOLEAN,
  fee_rule_gbk_default_amount_idr_snapshot   INT,
  fee_rule_gbk_is_active_snapshot            BOOLEAN,

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

-- =========================
-- INDEXES: fee_rules
-- =========================
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

CREATE INDEX IF NOT EXISTS ix_fee_rules_gbk
  ON fee_rules (fee_rule_general_billing_kind_id);

CREATE INDEX IF NOT EXISTS ix_fee_rules_billcode
  ON fee_rules (fee_rule_bill_code, fee_rule_scope);

-- =========================================================
-- TABLE: bill_batches (generik; class/section XOR)
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

  -- Katalog jenis + denorm code + option untuk one-off
  bill_batch_general_billing_kind_id UUID
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
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

-- =========================
-- INDEXES: bill_batches
-- =========================
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

CREATE INDEX IF NOT EXISTS ix_batch_gbk
  ON bill_batches (bill_batch_general_billing_kind_id);

-- =========================================================
-- TABLE: student_bills (core + relasi kelas/section + snapshot label)
-- =========================================================
CREATE TABLE IF NOT EXISTS student_bills (
  student_bill_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_bill_batch_id          UUID NOT NULL REFERENCES bill_batches(bill_batch_id) ON DELETE CASCADE,

  -- Tenant & subject (composite FK ke school_students)
  student_bill_school_id         UUID NOT NULL,
  student_bill_school_student_id UUID,
  CONSTRAINT fk_student_bill_student_tenant
    FOREIGN KEY (student_bill_school_student_id, student_bill_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Payer (opsional)
  student_bill_payer_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,

  -- Jenis + periode
  student_bill_general_billing_kind_id UUID
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  student_bill_bill_code         VARCHAR(60) NOT NULL DEFAULT 'SPP',
  student_bill_year              SMALLINT,
  student_bill_month             SMALLINT,
  student_bill_term_id           UUID,

  -- Option untuk one-off (boleh NULL untuk SPP)
  student_bill_option_code       VARCHAR(60),
  student_bill_option_label      VARCHAR(60),

  -- Nominal final
  student_bill_amount_idr        INT NOT NULL CHECK (student_bill_amount_idr >= 0),

  -- Status ringan
  student_bill_status            VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                                 CHECK (student_bill_status IN ('unpaid','paid','canceled')),
  student_bill_paid_at           TIMESTAMPTZ,
  student_bill_note              TEXT,

  -- Relasi kelas & section + snapshot label
  student_bill_class_id          UUID,
  student_bill_section_id        UUID,

  student_bill_class_name_snapshot    TEXT,
  student_bill_class_slug_snapshot    VARCHAR(80),
  student_bill_section_name_snapshot  TEXT,
  student_bill_section_slug_snapshot  VARCHAR(80),

  -- Audit
  student_bill_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_deleted_at        TIMESTAMPTZ,

  -- TENANT-SAFE FK komposit
  CONSTRAINT fk_student_bill_class_tenant
    FOREIGN KEY (student_bill_class_id, student_bill_school_id)
    REFERENCES classes (class_id, class_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_student_bill_section_tenant
    FOREIGN KEY (student_bill_section_id, student_bill_school_id)
    REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_student_bill_term_tenant
    FOREIGN KEY (student_bill_term_id, student_bill_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Idempotensi per-batch
  CONSTRAINT uq_student_bill_per_student
    UNIQUE (student_bill_batch_id, student_bill_school_student_id)
);

-- =========================
-- INDEXES: student_bills
-- =========================
CREATE INDEX IF NOT EXISTS ix_student_bills_gbk
  ON student_bills (student_bill_general_billing_kind_id);

CREATE INDEX IF NOT EXISTS ix_student_bill_amount
  ON student_bills (student_bill_amount_idr);

CREATE INDEX IF NOT EXISTS ix_student_bill_status
  ON student_bills (student_bill_status);

CREATE INDEX IF NOT EXISTS ix_student_bill_created_at
  ON student_bills (student_bill_created_at);

CREATE INDEX IF NOT EXISTS ix_student_bills_class_section_alive
  ON student_bills (student_bill_class_id, student_bill_section_id)
  WHERE student_bill_deleted_at IS NULL;

-- Periodik (SPP) → option_code NULL
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_periodic
  ON student_bills (
    student_bill_school_id,
    student_bill_school_student_id,
    student_bill_bill_code,
    student_bill_term_id,
    student_bill_year,
    student_bill_month
  )
  WHERE student_bill_deleted_at IS NULL
    AND student_bill_option_code IS NULL;

-- One-off (UNIFORM/BOOK/TRIP/REG) → option_code WAJIB
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_oneoff
  ON student_bills (
    student_bill_school_id,
    student_bill_school_student_id,
    student_bill_bill_code,
    student_bill_option_code
  )
  WHERE student_bill_deleted_at IS NULL
    AND student_bill_option_code IS NOT NULL;

COMMIT;
