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
  academic_terms_masjid_id     UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

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
  academic_terms_sequence      SMALLINT,        -- urutan dalam tahun ajaran (1=Ganjil, 2=Genap, ...)
  academic_terms_color         VARCHAR(16),
  academic_terms_sort_weight   SMALLINT DEFAULT 0,

  -- ===== Penjadwalan operasional =====
  academic_terms_timezone      VARCHAR(50),     -- override TZ tenant (opsional)
  academic_terms_registration_start TIMESTAMPTZ,
  academic_terms_registration_end   TIMESTAMPTZ,
  academic_terms_grading_start      TIMESTAMPTZ,
  academic_terms_grading_end        TIMESTAMPTZ,
  academic_terms_week_count    SMALLINT,
  academic_terms_min_credits   SMALLINT,
  academic_terms_max_credits   SMALLINT,

  -- ===== Kontrol & lifecycle =====
  academic_terms_is_default    BOOLEAN   NOT NULL DEFAULT FALSE,
  academic_terms_lock_enrollment BOOLEAN NOT NULL DEFAULT FALSE,
  academic_terms_lock_grades     BOOLEAN NOT NULL DEFAULT FALSE,
  academic_terms_frozen_at     TIMESTAMPTZ,
  academic_terms_archived_at   TIMESTAMPTZ,

  -- ===== Relasi & referensi antar term =====
  academic_terms_prev_id UUID REFERENCES academic_terms(academic_terms_id) ON DELETE SET NULL,
  academic_terms_next_id UUID REFERENCES academic_terms(academic_terms_id) ON DELETE SET NULL,

  -- ===== Kebijakan & metadata =====
  academic_terms_carry_over_policy JSONB,   -- aturan remedial/lanjut
  academic_terms_fee_policy        JSONB,   -- biaya administrasi per term
  academic_terms_public_notes      TEXT,
  academic_terms_internal_notes    TEXT,
  academic_terms_metadata          JSONB,
  academic_terms_translations      JSONB,

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
