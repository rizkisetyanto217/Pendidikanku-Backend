BEGIN;

-- =========================================================
-- Extensions (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- untuk index range/EXCLUDE jika kelak dibutuhkan

-- =========================================================
-- Function: touch updated_at (idempotent)
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_academic_terms_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.academic_terms_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =========================================================
-- Table
-- =========================================================
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_terms_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_terms_masjid_id     UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  academic_terms_academic_year TEXT NOT NULL,  -- contoh: '2026/2027'
  academic_terms_name          TEXT NOT NULL,  -- 'Ganjil' | 'Genap' | 'Pendek' | 'Khusus' | dst.

  academic_terms_start_date    TIMESTAMP NOT NULL,
  academic_terms_end_date      TIMESTAMP NOT NULL,
  academic_terms_is_active     BOOLEAN   NOT NULL DEFAULT TRUE,

  -- Half-open range: [start, end) — berguna untuk query tanggal
  academic_terms_period        DATERANGE GENERATED ALWAYS AS
    (daterange(academic_terms_start_date::date, academic_terms_end_date::date, '[)')) STORED,

  academic_terms_created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  academic_terms_updated_at    TIMESTAMP          DEFAULT CURRENT_TIMESTAMP,
  academic_terms_deleted_at    TIMESTAMP,

  CHECK (academic_terms_end_date >= academic_terms_start_date)
);

-- =========================================================
-- Trigger updated_at
-- =========================================================
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;
CREATE TRIGGER trg_touch_academic_terms
BEFORE UPDATE ON academic_terms
FOR EACH ROW EXECUTE FUNCTION fn_touch_academic_terms_updated_at();

-- =========================================================
-- Cleanup legacy constraints/indexes (jaga-jaga)
-- =========================================================

-- 1) Hapus UNIQUE (masjid, year, name) kalau pernah dibuat
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

-- 2) Hapus anti-overlap kalau pernah ada (overlap sekarang diperbolehkan)
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

-- 3) Hapus partial-unique "hanya 1 aktif" jika pernah ada
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

-- =========================================================
-- Index pendukung query (semua NON-UNIQUE)
-- =========================================================

-- a) List per masjid, urut tanggal
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_dates
ON academic_terms (academic_terms_masjid_id, academic_terms_start_date, academic_terms_end_date)
WHERE academic_terms_deleted_at IS NULL;

-- b) Query “tanggal termasuk dalam term” (range)
CREATE INDEX IF NOT EXISTS ix_academic_terms_period_gist
ON academic_terms USING GIST (academic_terms_period)
WHERE academic_terms_deleted_at IS NULL;

-- c) Ambil term aktif per masjid (bukan unique)
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
ON academic_terms (academic_terms_masjid_id)
WHERE academic_terms_is_active = TRUE
  AND academic_terms_deleted_at IS NULL;

-- d) Pencarian nama term (ILIKE)
CREATE INDEX IF NOT EXISTS ix_academic_terms_name_trgm
ON academic_terms USING GIN (lower(academic_terms_name) gin_trgm_ops)
WHERE academic_terms_deleted_at IS NULL;

-- e) Filter by academic_year (exact)
CREATE INDEX IF NOT EXISTS ix_academic_terms_year
ON academic_terms (academic_terms_masjid_id, academic_terms_academic_year)
WHERE academic_terms_deleted_at IS NULL;

-- f) Pencarian academic_year (ILIKE)
DROP INDEX IF EXISTS ix_academic_terms_year_trgm;
CREATE INDEX IF NOT EXISTS ix_academic_terms_year_trgm_lower
ON academic_terms USING GIN (lower(academic_terms_academic_year) gin_trgm_ops)
WHERE academic_terms_deleted_at IS NULL;

-- g) Sorting/pagination by created_at & updated_at
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_created_at
ON academic_terms (academic_terms_masjid_id, academic_terms_created_at)
WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_updated_at
ON academic_terms (academic_terms_masjid_id, academic_terms_updated_at)
WHERE academic_terms_deleted_at IS NULL;

COMMIT;
