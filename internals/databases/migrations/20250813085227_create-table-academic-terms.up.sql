BEGIN;

-- Extensions yang diperlukan
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram untuk GIN trgm

-- =========================================================
-- TABLE: academic_terms (plural) + columns singular
-- =========================================================
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_term_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_term_school_id     UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  academic_term_academic_year TEXT NOT NULL,  -- contoh: '2026/2027'
  academic_term_name          TEXT NOT NULL,  -- 'Ganjil' | 'Genap' | 'Pendek' | dst.

  academic_term_start_date    TIMESTAMPTZ NOT NULL,
  academic_term_end_date      TIMESTAMPTZ NOT NULL,
  academic_term_is_active     BOOLEAN   NOT NULL DEFAULT TRUE,

  academic_term_code          VARCHAR(24),     -- ex: 2026GJ
  academic_term_slug          VARCHAR(50),     -- URL-friendly per tenan

  -- angkatan (opsional). Disimpan sebagai tahun masuk/angkatan (mis. 2024).
  academic_term_angkatan      INTEGER,

  academic_term_description   TEXT,

  -- half-open range [start, end) - IMMUTABLE via explicit timezone
  academic_term_period        DATERANGE GENERATED ALWAYS AS
    (
      daterange(
        (academic_term_start_date AT TIME ZONE 'UTC')::date,
        (academic_term_end_date   AT TIME ZONE 'UTC')::date,
        '[)'
      )
    ) STORED,

  academic_term_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_term_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_term_deleted_at    TIMESTAMPTZ,

  CHECK (academic_term_end_date >= academic_term_start_date)
);

-- =========================================================
-- INDEXES (soft-delete aware di WHERE)
-- =========================================================

-- Rentang tanggal per tenant (range query cepat)
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_dates
  ON academic_terms (academic_term_school_id, academic_term_start_date, academic_term_end_date)
  WHERE academic_term_deleted_at IS NULL;

-- GIST untuk period range (cek overlap/periode berjalan)
CREATE INDEX IF NOT EXISTS ix_academic_terms_period_gist
  ON academic_terms USING GIST (academic_term_period)
  WHERE academic_term_deleted_at IS NULL;

-- Satu set indeks untuk "yang aktif" per tenant
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
  ON academic_terms (academic_term_school_id)
  WHERE academic_term_is_active = TRUE
    AND academic_term_deleted_at IS NULL;

-- Pencarian nama dengan trigram (case-insensitive)
CREATE INDEX IF NOT EXISTS ix_academic_terms_name_trgm
  ON academic_terms USING GIN (lower(academic_term_name) gin_trgm_ops)
  WHERE academic_term_deleted_at IS NULL;

-- Tahun akademik per tenant
CREATE INDEX IF NOT EXISTS ix_academic_terms_year
  ON academic_terms (academic_term_school_id, academic_term_academic_year)
  WHERE academic_term_deleted_at IS NULL;

-- Pencarian tahun akademik fuzzy (trgm, lower)
CREATE INDEX IF NOT EXISTS ix_academic_terms_year_trgm_lower
  ON academic_terms USING GIN (lower(academic_term_academic_year) gin_trgm_ops)
  WHERE academic_term_deleted_at IS NULL;

-- Angkatan per tenant
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_angkatan
  ON academic_terms (academic_term_school_id, academic_term_angkatan)
  WHERE academic_term_deleted_at IS NULL;

-- Arsip waktu per tenant
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_created_at
  ON academic_terms (academic_term_school_id, academic_term_created_at)
  WHERE academic_term_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_updated_at
  ON academic_terms (academic_term_school_id, academic_term_updated_at)
  WHERE academic_term_deleted_at IS NULL;

-- (Opsional) Unique komposit tenant-safe (berguna untuk FK komposit di downstream)
CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_terms_id_school
  ON academic_terms (academic_term_id, academic_term_school_id);

COMMIT;