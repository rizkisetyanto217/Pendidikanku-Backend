-- 20250823_01_academic_terms.up.sql

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- TABEL: academic_terms (final columns)
-- =========================================================
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_terms_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_terms_school_id     UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Identitas dasar
  academic_terms_academic_year TEXT NOT NULL,   -- contoh: '2026/2027'
  academic_terms_name          TEXT NOT NULL,   -- 'Ganjil' | 'Genap' | 'Pendek' | dst.

  -- Periode waktu
  academic_terms_start_date    TIMESTAMPTZ NOT NULL,
  academic_terms_end_date      TIMESTAMPTZ NOT NULL,
  academic_terms_is_active     BOOLEAN   NOT NULL DEFAULT TRUE,

  -- Angkatan opsional (tahun masuk)
  academic_terms_angkatan      INTEGER,

  -- ===== Identitas & tampilan =====
  academic_terms_code          VARCHAR(24),     -- ex: 2026GJ
  academic_terms_slug          VARCHAR(60),     -- URL-friendly per tenant

  academic_terms_color         VARCHAR(16),

  -- ===== Penjadwalan operasional =====
  academic_terms_registration_start TIMESTAMPTZ,
  academic_terms_registration_end   TIMESTAMPTZ,

  -- ===== Kebijakan & metadata =====
  academic_terms_public_notes      TEXT,
  academic_terms_metadata          JSONB,

  -- half-open range [start, end) - normalisasi ke UTC date
  academic_terms_period        DATERANGE GENERATED ALWAYS AS
    (
      daterange(
        (academic_terms_start_date AT TIME ZONE 'UTC')::date,
        (academic_terms_end_date   AT TIME ZONE 'UTC')::date,
        '[)'
      )
    ) STORED,

  -- Audit
  academic_terms_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_terms_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  academic_terms_deleted_at    TIMESTAMPTZ,

  -- ===== Guards =====
  CHECK (academic_terms_end_date >= academic_terms_start_date),

  CONSTRAINT ck_academic_terms_credits CHECK (
    (academic_terms_min_credits IS NULL OR academic_terms_min_credits >= 0) AND
    (academic_terms_max_credits IS NULL OR academic_terms_max_credits >= academic_terms_min_credits)
  ),

  CONSTRAINT ck_academic_terms_json_types CHECK (
    (academic_terms_metadata IS NULL OR jsonb_typeof(academic_terms_metadata) = 'object') AND
    (academic_terms_translations IS NULL OR jsonb_typeof(academic_terms_translations) = 'object') AND
    (academic_terms_carry_over_policy IS NULL OR jsonb_typeof(academic_terms_carry_over_policy) = 'object') AND
    (academic_terms_fee_policy IS NULL OR jsonb_typeof(academic_terms_fee_policy) = 'object')
  )
);

COMMIT;
