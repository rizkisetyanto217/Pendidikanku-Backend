-- =========================================================
-- Extensions (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- perlu utk EXCLUDE overlap

-- =========================================================
-- Trigger: touch updated_at khusus academic_terms
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_academic_terms_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.academic_terms_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =========================================================
-- ACADEMIC TERMS / SEMESTERS
-- =========================================================
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_terms_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_terms_masjid_id         UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  academic_terms_academic_year     TEXT NOT NULL,   -- contoh: '2025/2026'
  academic_terms_name              TEXT NOT NULL,   -- 'Ganjil' | 'Genap' | dst.

  academic_terms_start_date        TIMESTAMP NOT NULL,
  academic_terms_end_date          TIMESTAMP NOT NULL,
  academic_terms_is_active         BOOLEAN   NOT NULL DEFAULT TRUE,

  -- generated daterange utk deteksi overlap (pakai DATE saja dari timestamp)
  academic_terms_period            DATERANGE GENERATED ALWAYS AS
    (daterange(academic_terms_start_date::date, academic_terms_end_date::date, '[]')) STORED,

  academic_terms_created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  academic_terms_updated_at        TIMESTAMP,
  academic_terms_deleted_at        TIMESTAMP,

  CHECK (academic_terms_end_date >= academic_terms_start_date)
);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;
CREATE TRIGGER trg_touch_academic_terms
BEFORE UPDATE ON academic_terms
FOR EACH ROW EXECUTE FUNCTION fn_touch_academic_terms_updated_at();

-- =========================================================
-- Constraints & Indexing
-- =========================================================

-- 1) Unik kombinasi year+name per masjid (row live saja)
CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_terms_tenant_year_name_live
ON academic_terms (academic_terms_masjid_id, academic_terms_academic_year, academic_terms_name)
WHERE academic_terms_deleted_at IS NULL;

-- 2) Maksimal 1 term aktif per masjid (row live saja)
CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_terms_one_active_per_tenant
ON academic_terms (academic_terms_masjid_id)
WHERE academic_terms_is_active = TRUE AND academic_terms_deleted_at IS NULL;

-- 3) Larang overlap periode antar term dalam satu masjid (row live saja)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'ex_academic_terms_no_overlap_per_tenant'
  ) THEN
    ALTER TABLE academic_terms
    ADD CONSTRAINT ex_academic_terms_no_overlap_per_tenant
    EXCLUDE USING GIST (
      academic_terms_masjid_id WITH =,
      academic_terms_period    WITH &&
    )
    WHERE (academic_terms_deleted_at IS NULL);
  END IF;
END$$;

-- 4) Index pola query umum

-- a) List per masjid, urut tanggal
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_dates
ON academic_terms (academic_terms_masjid_id, academic_terms_start_date, academic_terms_end_date)
WHERE academic_terms_deleted_at IS NULL;

-- b) Lookup term yang "mengandung" tanggal tertentu (pakai daterange)
CREATE INDEX IF NOT EXISTS ix_academic_terms_period_gist
ON academic_terms USING GIST (academic_terms_period)
WHERE academic_terms_deleted_at IS NULL;

-- c) Ambil term aktif per masjid
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
ON academic_terms (academic_terms_masjid_id)
WHERE academic_terms_is_active = TRUE AND academic_terms_deleted_at IS NULL;

-- d) Pencarian by academic_terms_name (opsional: search LIKE/ILIKE)
CREATE INDEX IF NOT EXISTS ix_academic_terms_name_trgm
ON academic_terms USING GIN (lower(academic_terms_name) gin_trgm_ops)
WHERE academic_terms_deleted_at IS NULL;

-- e) Filter by academic_year (persis match)
CREATE INDEX IF NOT EXISTS ix_academic_terms_year
ON academic_terms (academic_terms_masjid_id, academic_terms_academic_year)
WHERE academic_terms_deleted_at IS NULL;

-- f) Search academic_year (LIKE/ILIKE)
CREATE INDEX IF NOT EXISTS ix_academic_terms_year_trgm
ON academic_terms USING GIN (academic_terms_academic_year gin_trgm_ops)
WHERE academic_terms_deleted_at IS NULL;