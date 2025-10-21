-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops (dipakai umum)
CREATE EXTENSION IF NOT EXISTS btree_gist; -- untuk EXCLUDE constraint

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
DO $$ BEGIN
  CREATE TYPE spp_fee_scope AS ENUM ('tenant','class_parent','class','section','student');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE gateway_event_status AS ENUM ('received','processed','ignored','duplicated','failed');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- MASTER: general_billing_kinds
--  - masjid_id NULL = GLOBAL kind (operasional aplikasi)
--  - category: 'billing' | 'campaign'
--  - visibility: 'public' | 'internal'
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billing_kinds (
  general_billing_kind_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  general_billing_kind_masjid_id UUID
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  general_billing_kind_code VARCHAR(60) NOT NULL,
  general_billing_kind_name TEXT NOT NULL,
  general_billing_kind_desc TEXT,
  general_billing_kind_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_kind_default_amount_idr INT CHECK (general_billing_kind_default_amount_idr >= 0),

  general_billing_kind_category   VARCHAR(20)
    CHECK (general_billing_kind_category IN ('billing','campaign')) DEFAULT 'billing',
  general_billing_kind_is_global  BOOLEAN NOT NULL DEFAULT FALSE,
  general_billing_kind_visibility VARCHAR(20)
    CHECK (general_billing_kind_visibility IN ('public','internal')),

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

-- Unik untuk GLOBAL kinds (tanpa masjid)
CREATE UNIQUE INDEX IF NOT EXISTS uq_gbk_code_global_alive
  ON general_billing_kinds (LOWER(general_billing_kind_code))
  WHERE general_billing_kind_deleted_at IS NULL
    AND general_billing_kind_masjid_id IS NULL;

CREATE INDEX IF NOT EXISTS ix_gbk_tenant_active
  ON general_billing_kinds (general_billing_kind_masjid_id, general_billing_kind_is_active)
  WHERE general_billing_kind_deleted_at IS NULL;

  

-- =========================================================
-- SPP fee rules (tiered, dengan overlap guard)
-- =========================================================
CREATE TABLE IF NOT EXISTS spp_fee_rules (
  spp_fee_rule_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  spp_fee_rule_masjid_id     UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  spp_fee_rule_scope         spp_fee_scope NOT NULL,
  spp_fee_rule_class_parent_id   UUID,
  spp_fee_rule_class_id          UUID,
  spp_fee_rule_section_id        UUID,
  spp_fee_rule_masjid_student_id UUID,

  spp_fee_rule_term_id       UUID REFERENCES academic_terms(academic_term_id) ON DELETE SET NULL,
  spp_fee_rule_month         SMALLINT CHECK (spp_fee_rule_month BETWEEN 1 AND 12),
  spp_fee_rule_year          SMALLINT CHECK (spp_fee_rule_year BETWEEN 2000 AND 2100),

  spp_fee_rule_option_code   VARCHAR(20) NOT NULL DEFAULT 'T1',
  spp_fee_rule_option_label  VARCHAR(60),
  spp_fee_rule_is_default    BOOLEAN NOT NULL DEFAULT FALSE,

  spp_fee_rule_amount_idr    INT NOT NULL CHECK (spp_fee_rule_amount_idr >= 0),

  spp_fee_rule_effective_from DATE,
  spp_fee_rule_effective_to   DATE,

  spp_fee_rule_note          TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,

  CONSTRAINT ck_fee_rules_scope_target CHECK (
    (spp_fee_rule_scope = 'tenant'       AND spp_fee_rule_class_parent_id IS NULL AND spp_fee_rule_class_id IS NULL AND spp_fee_rule_section_id IS NULL AND spp_fee_rule_masjid_student_id IS NULL)
 OR (spp_fee_rule_scope = 'class_parent' AND spp_fee_rule_class_parent_id IS NOT NULL AND spp_fee_rule_class_id IS NULL AND spp_fee_rule_section_id IS NULL AND spp_fee_rule_masjid_student_id IS NULL)
 OR (spp_fee_rule_scope = 'class'        AND spp_fee_rule_class_id        IS NOT NULL AND spp_fee_rule_class_parent_id IS NULL AND spp_fee_rule_section_id IS NULL AND spp_fee_rule_masjid_student_id IS NULL)
 OR (spp_fee_rule_scope = 'section'      AND spp_fee_rule_section_id      IS NOT NULL AND spp_fee_rule_class_parent_id IS NULL AND spp_fee_rule_class_id IS NULL AND spp_fee_rule_masjid_student_id IS NULL)
 OR (spp_fee_rule_scope = 'student'      AND spp_fee_rule_masjid_student_id IS NOT NULL AND spp_fee_rule_class_parent_id IS NULL AND spp_fee_rule_class_id IS NULL AND spp_fee_rule_section_id IS NULL)
  ),
  CONSTRAINT ck_fee_rules_period CHECK (
    spp_fee_rule_term_id IS NOT NULL
    OR (spp_fee_rule_month IS NOT NULL AND spp_fee_rule_year IS NOT NULL)
  ),
  CONSTRAINT ck_fee_rules_effective_window CHECK (
    spp_fee_rule_effective_from IS NULL
    OR spp_fee_rule_effective_to IS NULL
    OR spp_fee_rule_effective_to >= spp_fee_rule_effective_from
  ),

  spp_fee_rule_effective_daterange daterange
    GENERATED ALWAYS AS (
      daterange(
        COALESCE(spp_fee_rule_effective_from, '-infinity'::date),
        COALESCE(spp_fee_rule_effective_to,   'infinity'::date),
        '[]'
      )
    ) STORED,

  EXCLUDE USING gist (
    spp_fee_rule_masjid_id WITH =,
    spp_fee_rule_scope     WITH =,
    spp_fee_rule_class_parent_id WITH =,
    spp_fee_rule_class_id  WITH =,
    spp_fee_rule_section_id WITH =,
    spp_fee_rule_masjid_student_id WITH =,
    spp_fee_rule_term_id   WITH =,
    spp_fee_rule_effective_daterange WITH &&
  ) WHERE (deleted_at IS NULL AND spp_fee_rule_term_id IS NOT NULL),

  EXCLUDE USING gist (
    spp_fee_rule_masjid_id WITH =,
    spp_fee_rule_scope     WITH =,
    spp_fee_rule_class_parent_id WITH =,
    spp_fee_rule_class_id  WITH =,
    spp_fee_rule_section_id WITH =,
    spp_fee_rule_masjid_student_id WITH =,
    spp_fee_rule_year      WITH =,
    spp_fee_rule_month     WITH =,
    spp_fee_rule_effective_daterange WITH &&
  ) WHERE (deleted_at IS NULL AND spp_fee_rule_term_id IS NULL
          AND spp_fee_rule_year IS NOT NULL AND spp_fee_rule_month IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_fee_rules_tenant_scope  ON spp_fee_rules (spp_fee_rule_masjid_id, spp_fee_rule_scope);
CREATE INDEX IF NOT EXISTS idx_fee_rules_term          ON spp_fee_rules (spp_fee_rule_term_id);
CREATE INDEX IF NOT EXISTS idx_fee_rules_month_year    ON spp_fee_rules (spp_fee_rule_year, spp_fee_rule_month);
CREATE INDEX IF NOT EXISTS idx_fee_rules_amount        ON spp_fee_rules (spp_fee_rule_amount_idr);
CREATE INDEX IF NOT EXISTS idx_fee_rules_option_code   ON spp_fee_rules (LOWER(spp_fee_rule_option_code));
CREATE INDEX IF NOT EXISTS idx_fee_rules_is_default    ON spp_fee_rules (spp_fee_rule_is_default);

-- =========================================================
-- SPP billings & user_spp_billings
-- =========================================================
CREATE TABLE IF NOT EXISTS spp_billings (
  spp_billing_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  spp_billing_masjid_id   UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  spp_billing_class_id    UUID REFERENCES classes(class_id)   ON DELETE SET NULL,
  spp_billing_month       SMALLINT NOT NULL CHECK (spp_billing_month BETWEEN 1 AND 12),
  spp_billing_year        SMALLINT NOT NULL CHECK (spp_billing_year BETWEEN 2000 AND 2100),
  spp_billing_term_id     UUID REFERENCES academic_terms(academic_term_id) ON UPDATE CASCADE ON DELETE SET NULL,
  spp_billing_title       TEXT NOT NULL,
  spp_billing_due_date    DATE,
  spp_billing_note        TEXT,
  spp_billing_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_spp_billings_batch
  ON spp_billings (spp_billing_masjid_id, spp_billing_class_id, spp_billing_month, spp_billing_year)
  WHERE spp_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_spp_billings_tenant_month_year_live
  ON spp_billings (spp_billing_masjid_id, spp_billing_year, spp_billing_month)
  WHERE spp_billing_deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS user_spp_billings (
  user_spp_billing_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_spp_billing_billing_id        UUID NOT NULL REFERENCES spp_billings(spp_billing_id) ON DELETE CASCADE,
  user_spp_billing_masjid_id         UUID NOT NULL,
  user_spp_billing_masjid_student_id UUID,
  CONSTRAINT fk_usb_student_tenant FOREIGN KEY (user_spp_billing_masjid_student_id, user_spp_billing_masjid_id)
    REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,
  user_spp_billing_payer_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
  user_spp_billing_option_code       VARCHAR(20),
  user_spp_billing_option_label      VARCHAR(60),
  user_spp_billing_amount_idr        INT NOT NULL CHECK (user_spp_billing_amount_idr >= 0),
  user_spp_billing_status            VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                                     CHECK (user_spp_billing_status IN ('unpaid','paid','canceled')),
  user_spp_billing_paid_at           TIMESTAMPTZ,
  user_spp_billing_note              TEXT,
  user_spp_billing_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_deleted_at        TIMESTAMPTZ,
  CONSTRAINT uq_usb_per_student UNIQUE (user_spp_billing_billing_id, user_spp_billing_masjid_student_id)
);

-- =========================================================
-- GENERAL billings (pakai KINDS table) + snapshots (minimal)
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

-- =========================================================
-- USER general billings (snapshot kind code/name)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_general_billings (
  user_general_billing_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_general_billing_masjid_id     UUID NOT NULL,
  user_general_billing_masjid_student_id UUID,
  CONSTRAINT fk_ugb_student_tenant FOREIGN KEY (user_general_billing_masjid_student_id, user_general_billing_masjid_id)
    REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  user_general_billing_payer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  user_general_billing_billing_id    UUID NOT NULL
    REFERENCES general_billings(general_billing_id) ON DELETE CASCADE,

  user_general_billing_amount_idr    INT NOT NULL CHECK (user_general_billing_amount_idr >= 0),

  user_general_billing_status        VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                                     CHECK (user_general_billing_status IN ('unpaid','paid','canceled')),
  user_general_billing_paid_at       TIMESTAMPTZ,
  user_general_billing_note          TEXT,

  user_general_billing_title_snapshot      TEXT,
  user_general_billing_kind_code_snapshot  TEXT,
  user_general_billing_kind_name_snapshot  TEXT,

  user_general_billing_meta           JSONB,

  user_general_billing_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_deleted_at    TIMESTAMPTZ,

  CONSTRAINT uq_ugb_per_student UNIQUE (user_general_billing_billing_id, user_general_billing_masjid_student_id),
  CONSTRAINT uq_ugb_per_payer   UNIQUE (user_general_billing_billing_id, user_general_billing_payer_user_id)
);

COMMIT;
