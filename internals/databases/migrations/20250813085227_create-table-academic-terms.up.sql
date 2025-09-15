-- 20250823_01_academic_terms.up.sql

BEGIN;

-- Extensions yang diperlukan
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Function: touch updated_at
CREATE OR REPLACE FUNCTION fn_touch_academic_terms_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.academic_terms_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- Table: academic_terms
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_terms_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_terms_masjid_id     UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  academic_terms_academic_year TEXT NOT NULL,  -- contoh: '2026/2027'
  academic_terms_name          TEXT NOT NULL,  -- 'Ganjil' | 'Genap' | 'Pendek' | dst.

  academic_terms_start_date    TIMESTAMPTZ NOT NULL,
  academic_terms_end_date      TIMESTAMPTZ NOT NULL,
  academic_terms_is_active     BOOLEAN   NOT NULL DEFAULT TRUE,

  academic_terms_code          VARCHAR(24),     -- ex: 2026GJ
  academic_terms_slug          VARCHAR(50),     -- URL-friendly per tenant

  -- NEW: angkatan (opsional). Disimpan sebagai tahun masuk/angkatan (mis. 2024).
  academic_terms_angkatan      INTEGER,

  academic_terms_description TEXT,

  -- half-open range [start, end) - IMMUTABLE via explicit timezone
  academic_terms_period        DATERANGE GENERATED ALWAYS AS
    (
      daterange(
        (academic_terms_start_date AT TIME ZONE 'UTC')::date,
        (academic_terms_end_date   AT TIME ZONE 'UTC')::date,
        '[)'
      )
    ) STORED,

  academic_terms_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_terms_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_terms_deleted_at    TIMESTAMPTZ,

  CHECK (academic_terms_end_date >= academic_terms_start_date)
);

-- Bersihkan constraint/index lama (opsional; aman kalau tidak ada)
DROP INDEX IF EXISTS uq_academic_terms_tenant_year_name_live;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_academic_terms_tenant_year_name_live'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT uq_academic_terms_tenant_year_name_live;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'ex_academic_terms_no_overlap_per_tenant'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT ex_academic_terms_no_overlap_per_tenant;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_academic_terms_one_active_per_tenant'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT uq_academic_terms_one_active_per_tenant;
  END IF;
END$$;

DROP INDEX IF EXISTS uq_academic_terms_one_active_per_tenant;

-- Indexes non-unique (soft-delete aware)
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_dates
  ON academic_terms (academic_terms_masjid_id, academic_terms_start_date, academic_terms_end_date)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_period_gist
  ON academic_terms USING GIST (academic_terms_period)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
  ON academic_terms (academic_terms_masjid_id)
  WHERE academic_terms_is_active = TRUE
    AND academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_name_trgm
  ON academic_terms USING GIN (lower(academic_terms_name) gin_trgm_ops)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_year
  ON academic_terms (academic_terms_masjid_id, academic_terms_academic_year)
  WHERE academic_terms_deleted_at IS NULL;

DROP INDEX IF EXISTS ix_academic_terms_year_trgm;
CREATE INDEX IF NOT EXISTS ix_academic_terms_year_trgm_lower
  ON academic_terms USING GIN (lower(academic_terms_academic_year) gin_trgm_ops)
  WHERE academic_terms_deleted_at IS NULL;

-- NEW: index per-tenant untuk angkatan (filtering cepat)
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_angkatan
  ON academic_terms (academic_terms_masjid_id, academic_terms_angkatan)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_created_at
  ON academic_terms (academic_terms_masjid_id, academic_terms_created_at)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_updated_at
  ON academic_terms (academic_terms_masjid_id, academic_terms_updated_at)
  WHERE academic_terms_deleted_at IS NULL;

-- Composite-unique untuk FK komposit (tenant-safe)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid'
  ) THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_masjid
      UNIQUE (academic_terms_id, academic_terms_masjid_id);
  END IF;
END$$;

COMMIT;
