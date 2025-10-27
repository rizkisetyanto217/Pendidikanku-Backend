-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops
CREATE EXTENSION IF NOT EXISTS btree_gist; -- EXCLUDE constraint (range)

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
DO $$ BEGIN
  CREATE TYPE fee_scope AS ENUM ('tenant','class_parent','class','section','student');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- MASTER: general_billing_kinds (katalog tunggal, bisa GLOBAL/tenant)
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billing_kinds (
  general_billing_kind_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  general_billing_kind_masjid_id    UUID
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  general_billing_kind_code         VARCHAR(60) NOT NULL,
  general_billing_kind_name         TEXT NOT NULL,
  general_billing_kind_desc         TEXT,
  general_billing_kind_is_active    BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_kind_default_amount_idr INT CHECK (general_billing_kind_default_amount_idr >= 0),

  general_billing_kind_category     VARCHAR(20)
    CHECK (general_billing_kind_category IN ('billing','campaign')) DEFAULT 'billing',
  general_billing_kind_is_global    BOOLEAN NOT NULL DEFAULT FALSE,
  general_billing_kind_visibility   VARCHAR(20)
    CHECK (general_billing_kind_visibility IN ('public','internal')),

  -- flags untuk pipeline per-siswa
  general_billing_kind_is_recurring          BOOLEAN NOT NULL DEFAULT FALSE, -- SPP=TRUE
  general_billing_kind_requires_month_year   BOOLEAN NOT NULL DEFAULT FALSE, -- SPP=TRUE
  general_billing_kind_requires_option_code  BOOLEAN NOT NULL DEFAULT FALSE, -- UNIFORM/BOOK/TRIP=TRUE

  general_billing_kind_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_kind_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_kind_deleted_at TIMESTAMPTZ
);

-- Pastikan kolom masjid_id boleh NULL (GLOBAL)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name='general_billing_kinds'
      AND column_name='general_billing_kind_masjid_id'
      AND is_nullable='NO'
  ) THEN
    ALTER TABLE general_billing_kinds
      ALTER COLUMN general_billing_kind_masjid_id DROP NOT NULL;
  END IF;
END$$;

-- Unik per tenant (alive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_gbk_code_per_tenant_alive
  ON general_billing_kinds (general_billing_kind_masjid_id, LOWER(general_billing_kind_code))
  WHERE general_billing_kind_deleted_at IS NULL;

-- Unik GLOBAL (tanpa masjid)
CREATE UNIQUE INDEX IF NOT EXISTS uq_gbk_code_global_alive
  ON general_billing_kinds (LOWER(general_billing_kind_code))
  WHERE general_billing_kind_deleted_at IS NULL
    AND general_billing_kind_masjid_id IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_tenant_active
  ON general_billing_kinds (general_billing_kind_masjid_id, general_billing_kind_is_active)
  WHERE general_billing_kind_deleted_at IS NULL;

-- SEED defaults (aman diulang)
INSERT INTO general_billing_kinds
  (general_billing_kind_masjid_id, general_billing_kind_code, general_billing_kind_name,
   general_billing_kind_is_recurring, general_billing_kind_requires_month_year, general_billing_kind_requires_option_code,
   general_billing_kind_is_global, general_billing_kind_category)
VALUES
  (NULL,'SPP','SPP Bulanan',         TRUE,  TRUE,  FALSE, TRUE, 'billing'),
  (NULL,'BOOK','Buku',               FALSE, FALSE, TRUE,  TRUE, 'billing'),
  (NULL,'UNIFORM','Seragam',         FALSE, FALSE, TRUE,  TRUE, 'billing'),
  (NULL,'REG','Pendaftaran',         FALSE, FALSE, FALSE, TRUE, 'billing'),
  (NULL,'TRIP','Karya Wisata/Trip',  FALSE, FALSE, TRUE,  TRUE, 'billing')
ON CONFLICT DO NOTHING;

-- =========================================================
-- FEE RULES (generik)
-- =========================================================
CREATE TABLE IF NOT EXISTS fee_rules (
  fee_rule_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fee_rule_masjid_id         UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  fee_rule_scope             fee_scope NOT NULL,
  fee_rule_class_parent_id   UUID,
  fee_rule_class_id          UUID,
  fee_rule_section_id        UUID,
  fee_rule_masjid_student_id UUID,

  -- Periode (salah satu: term_id ATAU year+month)
  fee_rule_term_id           UUID REFERENCES academic_terms(academic_term_id) ON DELETE SET NULL,
  fee_rule_month             SMALLINT CHECK (fee_rule_month BETWEEN 1 AND 12),
  fee_rule_year              SMALLINT CHECK (fee_rule_year BETWEEN 2000 AND 2100),

  -- Jenis rule (link ke katalog + denorm code)
  fee_rule_general_billing_kind_id UUID
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  fee_rule_bill_code         VARCHAR(60) NOT NULL DEFAULT 'SPP',

  -- Opsi/label
  fee_rule_option_code       VARCHAR(20) NOT NULL DEFAULT 'T1',
  fee_rule_option_label      VARCHAR(60),
  fee_rule_is_default        BOOLEAN NOT NULL DEFAULT FALSE,

  -- Nominal
  fee_rule_amount_idr        INT NOT NULL CHECK (fee_rule_amount_idr >= 0),

  -- Effective window
  fee_rule_effective_from    DATE,
  fee_rule_effective_to      DATE,

  fee_rule_note              TEXT,

  fee_rule_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_rule_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  fee_rule_deleted_at        TIMESTAMPTZ,

  CONSTRAINT ck_fee_rules_scope_target CHECK (
    (fee_rule_scope = 'tenant'       AND fee_rule_class_parent_id IS NULL AND fee_rule_class_id IS NULL AND fee_rule_section_id IS NULL AND fee_rule_masjid_student_id IS NULL)
 OR (fee_rule_scope = 'class_parent' AND fee_rule_class_parent_id IS NOT NULL AND fee_rule_class_id IS NULL AND fee_rule_section_id IS NULL AND fee_rule_masjid_student_id IS NULL)
 OR (fee_rule_scope = 'class'        AND fee_rule_class_id        IS NOT NULL AND fee_rule_class_parent_id IS NULL AND fee_rule_section_id IS NULL AND fee_rule_masjid_student_id IS NULL)
 OR (fee_rule_scope = 'section'      AND fee_rule_section_id      IS NOT NULL AND fee_rule_class_parent_id IS NULL AND fee_rule_class_id IS NULL AND fee_rule_masjid_student_id IS NULL)
 OR (fee_rule_scope = 'student'      AND fee_rule_masjid_student_id IS NOT NULL AND fee_rule_class_parent_id IS NULL AND fee_rule_class_id IS NULL AND fee_rule_section_id IS NULL)
  ),
  CONSTRAINT ck_fee_rules_period CHECK (
    fee_rule_term_id IS NOT NULL
    OR (fee_rule_month IS NOT NULL AND fee_rule_year IS NOT NULL)
  ),
  CONSTRAINT ck_fee_rules_effective_window CHECK (
    fee_rule_effective_from IS NULL
    OR fee_rule_effective_to IS NULL
    OR fee_rule_effective_to >= fee_rule_effective_from
  ),

  -- Generated daterange utk EXCLUDE
  fee_rule_effective_daterange daterange
    GENERATED ALWAYS AS (
      daterange(
        COALESCE(fee_rule_effective_from, '-infinity'::date),
        COALESCE(fee_rule_effective_to,   'infinity'::date),
        '[]'
      )
    ) STORED
);

-- No-overlap guard (pakai btree_gist)
DO $$
BEGIN
  -- Overlap untuk mode TERM
  EXECUTE $X$
    ALTER TABLE fee_rules
    ADD CONSTRAINT ex_fee_rules_term_no_overlap
    EXCLUDE USING gist (
      fee_rule_masjid_id WITH =,
      fee_rule_bill_code WITH =,
      fee_rule_scope     WITH =,
      fee_rule_class_parent_id WITH =,
      fee_rule_class_id  WITH =,
      fee_rule_section_id WITH =,
      fee_rule_masjid_student_id WITH =,
      fee_rule_term_id   WITH =,
      fee_rule_effective_daterange WITH &&
    ) WHERE (fee_rule_deleted_at IS NULL AND fee_rule_term_id IS NOT NULL)
  $X$;
EXCEPTION WHEN duplicate_table THEN NULL; END $$;

DO $$
BEGIN
  -- Overlap untuk mode YM (tanpa TERM)
  EXECUTE $Y$
    ALTER TABLE fee_rules
    ADD CONSTRAINT ex_fee_rules_ym_no_overlap
    EXCLUDE USING gist (
      fee_rule_masjid_id WITH =,
      fee_rule_bill_code WITH =,
      fee_rule_scope     WITH =,
      fee_rule_class_parent_id WITH =,
      fee_rule_class_id  WITH =,
      fee_rule_section_id WITH =,
      fee_rule_masjid_student_id WITH =,
      fee_rule_year      WITH =,
      fee_rule_month     WITH =,
      fee_rule_effective_daterange WITH &&
    ) WHERE (fee_rule_deleted_at IS NULL AND fee_rule_term_id IS NULL
             AND fee_rule_year IS NOT NULL AND fee_rule_month IS NOT NULL)
  $Y$;
EXCEPTION WHEN duplicate_table THEN NULL; END $$;

-- Index bantu fee_rules
CREATE INDEX IF NOT EXISTS idx_fee_rules_tenant_scope
  ON fee_rules (fee_rule_masjid_id, fee_rule_scope);
CREATE INDEX IF NOT EXISTS idx_fee_rules_term
  ON fee_rules (fee_rule_term_id);
CREATE INDEX IF NOT EXISTS idx_fee_rules_month_year
  ON fee_rules (fee_rule_year, fee_rule_month);
CREATE INDEX IF NOT EXISTS idx_fee_rules_amount
  ON fee_rules (fee_rule_amount_idr);
CREATE INDEX IF NOT EXISTS idx_fee_rules_option_code
  ON fee_rules (LOWER(fee_rule_option_code));
CREATE INDEX IF NOT EXISTS idx_fee_rules_is_default
  ON fee_rules (fee_rule_is_default);
CREATE INDEX IF NOT EXISTS ix_fee_rules_gbk
  ON fee_rules (fee_rule_general_billing_kind_id);
CREATE INDEX IF NOT EXISTS ix_fee_rules_billcode
  ON fee_rules (fee_rule_bill_code, fee_rule_scope);

-- =========================================================
-- BILL BATCHES (generik; class/section XOR)
-- =========================================================
CREATE TABLE IF NOT EXISTS bill_batches (
  bill_batch_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  bill_batch_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  bill_batch_class_id   UUID REFERENCES classes(class_id) ON DELETE SET NULL,
  bill_batch_section_id UUID REFERENCES class_sections(class_section_id) ON DELETE SET NULL,

  -- Periode (untuk recurring seperti SPP)
  bill_batch_month      SMALLINT CHECK (bill_batch_month BETWEEN 1 AND 12),
  bill_batch_year       SMALLINT CHECK (bill_batch_year BETWEEN 2000 AND 2100),
  bill_batch_term_id    UUID REFERENCES academic_terms(academic_term_id) ON UPDATE CASCADE ON DELETE SET NULL,

  -- Katalog jenis + denorm code + option untuk one-off
  bill_batch_general_billing_kind_id UUID
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  bill_batch_bill_code  VARCHAR(60) NOT NULL DEFAULT 'SPP',
  bill_batch_option_code VARCHAR(60), -- wajib untuk one-off (UNIFORM/BOOK/TRIP/REG)

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

  CONSTRAINT ck_bill_batches_xor_class_section
    CHECK (
      (bill_batch_class_id IS NOT NULL AND bill_batch_section_id IS NULL)
      OR
      (bill_batch_class_id IS NULL AND bill_batch_section_id IS NOT NULL)
    )
);

-- Idempotensi BATCH:
-- Periodik (SPP) → option_code NULL, unik per YM
CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_periodic_section
  ON bill_batches (bill_batch_masjid_id, bill_batch_bill_code, bill_batch_section_id, bill_batch_term_id, bill_batch_year, bill_batch_month)
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_section_id IS NOT NULL
    AND bill_batch_option_code IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_periodic_class
  ON bill_batches (bill_batch_masjid_id, bill_batch_bill_code, bill_batch_class_id, bill_batch_term_id, bill_batch_year, bill_batch_month)
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_class_id IS NOT NULL
    AND bill_batch_option_code IS NULL;

-- One-off (UNIFORM/BOOK/TRIP/REG) → option_code WAJIB, tanpa YM
CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_oneoff_section
  ON bill_batches (bill_batch_masjid_id, bill_batch_bill_code, bill_batch_section_id, bill_batch_term_id, bill_batch_option_code)
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_section_id IS NOT NULL
    AND bill_batch_option_code IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_oneoff_class
  ON bill_batches (bill_batch_masjid_id, bill_batch_bill_code, bill_batch_class_id, bill_batch_term_id, bill_batch_option_code)
  WHERE bill_batch_deleted_at IS NULL
    AND bill_batch_class_id IS NOT NULL
    AND bill_batch_option_code IS NOT NULL;

-- Index bantu query batch
CREATE INDEX IF NOT EXISTS ix_bill_batches_tenant_ym_alive
  ON bill_batches (bill_batch_masjid_id, bill_batch_year, bill_batch_month)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_due_date_alive
  ON bill_batches (bill_batch_due_date)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_masjid_created_at_alive
  ON bill_batches (bill_batch_masjid_id, bill_batch_created_at DESC)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_masjid_updated_at_alive
  ON bill_batches (bill_batch_masjid_id, bill_batch_updated_at DESC)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_term_alive
  ON bill_batches (bill_batch_term_id)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_masjid_term_alive
  ON bill_batches (bill_batch_masjid_id, bill_batch_term_id)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_class_alive
  ON bill_batches (bill_batch_class_id)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_section_alive
  ON bill_batches (bill_batch_section_id)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_masjid_due_date_alive
  ON bill_batches (bill_batch_masjid_id, bill_batch_due_date)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_bill_batches_title_trgm_alive
  ON bill_batches USING GIN (LOWER(bill_batch_title) gin_trgm_ops)
  WHERE bill_batch_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_batch_gbk
  ON bill_batches (bill_batch_general_billing_kind_id);

-- =========================================================
-- STUDENT BILLS (generik per siswa)
-- =========================================================
CREATE TABLE IF NOT EXISTS student_bills (
  student_bill_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_bill_batch_id          UUID NOT NULL REFERENCES bill_batches(bill_batch_id) ON DELETE CASCADE,

  student_bill_masjid_id         UUID NOT NULL,
  student_bill_masjid_student_id UUID,
  CONSTRAINT fk_student_bill_student_tenant FOREIGN KEY (student_bill_masjid_student_id, student_bill_masjid_id)
    REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  student_bill_payer_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,

  -- Denorm jenis + periode (ikut batch)
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

  student_bill_amount_idr        INT NOT NULL CHECK (student_bill_amount_idr >= 0),

  student_bill_status            VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                                 CHECK (student_bill_status IN ('unpaid','paid','canceled')),
  student_bill_paid_at           TIMESTAMPTZ,
  student_bill_note              TEXT,

  student_bill_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_bill_deleted_at        TIMESTAMPTZ,

  -- Idempotensi per-batch
  CONSTRAINT uq_student_bill_per_student UNIQUE (student_bill_batch_id, student_bill_masjid_student_id)
);

CREATE INDEX IF NOT EXISTS ix_student_bills_gbk ON student_bills (student_bill_general_billing_kind_id);
CREATE INDEX IF NOT EXISTS ix_student_bill_amount ON student_bills (student_bill_amount_idr);
CREATE INDEX IF NOT EXISTS ix_student_bill_status ON student_bills (student_bill_status);
CREATE INDEX IF NOT EXISTS ix_student_bill_created_at ON student_bills (student_bill_created_at);

-- Idempotensi baris siswa:
-- Periodik (SPP) → tanpa option_code
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_periodic
  ON student_bills (student_bill_masjid_id, student_bill_masjid_student_id, student_bill_bill_code, student_bill_term_id, student_bill_year, student_bill_month)
  WHERE student_bill_deleted_at IS NULL
    AND student_bill_option_code IS NULL;

-- One-off (UNIFORM/BOOK/TRIP/REG) → wajib option_code
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_oneoff
  ON student_bills (student_bill_masjid_id, student_bill_masjid_student_id, student_bill_bill_code, student_bill_option_code)
  WHERE student_bill_deleted_at IS NULL
    AND student_bill_option_code IS NOT NULL;

-- =========================================================
-- GENERAL billings (tetap, untuk non-per-siswa/campaign)
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billings (
  general_billing_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  general_billing_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  general_billing_kind_id    UUID NOT NULL
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  general_billing_code       VARCHAR(60),
  general_billing_title      TEXT NOT NULL,
  general_billing_desc       TEXT,

  -- cakupan akademik (opsional)
  general_billing_class_id   UUID REFERENCES classes(class_id) ON DELETE SET NULL,
  general_billing_section_id UUID REFERENCES class_sections(class_section_id) ON DELETE SET NULL,
  general_billing_term_id    UUID REFERENCES academic_terms(academic_term_id) ON DELETE SET NULL,

  general_billing_due_date   DATE,
  general_billing_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_default_amount_idr INT CHECK (general_billing_default_amount_idr >= 0),

  -- snapshots (MINIMAL)
  general_billing_kind_snapshot    JSONB,  -- {id, code, name}
  general_billing_class_snapshot   JSONB,  -- {id, name, slug}
  general_billing_section_snapshot JSONB,  -- {id, name, code}
  general_billing_term_snapshot    JSONB,  -- {id, academic_year, name, slug}

  general_billing_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_general_billings_code_per_tenant_alive
  ON general_billings (general_billing_masjid_id, LOWER(general_billing_code))
  WHERE general_billing_deleted_at IS NULL AND general_billing_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_gb_tenant_kind_active_created
  ON general_billings (general_billing_masjid_id, general_billing_kind_id, general_billing_is_active, general_billing_created_at DESC)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_due_alive
  ON general_billings (general_billing_due_date)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_kind_alive
  ON general_billings (general_billing_kind_id)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_term_alive
  ON general_billings (general_billing_term_id)
  WHERE general_billing_deleted_at IS NULL;

COMMIT;