-- 20250823_01_academic_terms.down.sql

BEGIN;

-- Hapus trigger jika ada
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;

-- Hapus tabel academic_terms (otomatis akan hapus constraint/index terkait)
DROP TABLE IF EXISTS academic_terms CASCADE;

-- Hapus function
DROP FUNCTION IF EXISTS fn_touch_academic_terms_updated_at();

-- Hapus index yang mungkin masih ada (opsional, aman walau sudah terhapus karena CASCADE)
DROP INDEX IF EXISTS ix_academic_terms_tenant_dates;
DROP INDEX IF EXISTS ix_academic_terms_period_gist;
DROP INDEX IF EXISTS ix_academic_terms_tenant_active_live;
DROP INDEX IF EXISTS ix_academic_terms_name_trgm;
DROP INDEX IF EXISTS ix_academic_terms_year;
DROP INDEX IF EXISTS ix_academic_terms_year_trgm_lower;
DROP INDEX IF EXISTS ix_academic_terms_tenant_created_at;
DROP INDEX IF EXISTS ix_academic_terms_tenant_updated_at;

-- Hapus constraint unik jika masih ada (opsional)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid'
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT uq_academic_terms_id_masjid;
  END IF;
END$$;

COMMIT;
