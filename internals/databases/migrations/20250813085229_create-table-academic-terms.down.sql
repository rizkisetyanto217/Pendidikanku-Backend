-- 20250823_01_academic_terms.down.sql

BEGIN;

-- Lepas FK yang mungkin masih menempel ke academic_terms (idempotent)
DO $$
BEGIN
  -- dari class_term_openings
  IF to_regclass('public.class_term_openings') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'fk_cto_term_masjid_pair'
        AND conrelid = 'public.class_term_openings'::regclass
    ) THEN
      EXECUTE 'ALTER TABLE public.class_term_openings DROP CONSTRAINT fk_cto_term_masjid_pair';
    END IF;
  END IF;

  -- dari user_classes (jika di migration lain kamu tambahkan FK tsb)
  IF to_regclass('public.user_classes') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'fk_uc_term_masjid_pair'
        AND conrelid = 'public.user_classes'::regclass
    ) THEN
      EXECUTE 'ALTER TABLE public.user_classes DROP CONSTRAINT fk_uc_term_masjid_pair';
    END IF;
  END IF;
END$$;

-- Drop trigger
DO $$
BEGIN
  IF to_regclass('public.academic_terms') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_trigger
      WHERE tgname = 'trg_touch_academic_terms'
        AND tgrelid = 'public.academic_terms'::regclass
    ) THEN
      EXECUTE 'DROP TRIGGER IF EXISTS trg_touch_academic_terms ON public.academic_terms';
    END IF;
  END IF;
END$$;

-- Drop function trigger
DROP FUNCTION IF EXISTS fn_touch_academic_terms_updated_at() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS ix_academic_terms_tenant_dates;
DROP INDEX IF EXISTS ix_academic_terms_period_gist;
DROP INDEX IF EXISTS ix_academic_terms_tenant_active_live;
DROP INDEX IF EXISTS ix_academic_terms_name_trgm;
DROP INDEX IF EXISTS ix_academic_terms_year;
DROP INDEX IF EXISTS ix_academic_terms_year_trgm_lower;
DROP INDEX IF EXISTS ix_academic_terms_tenant_created_at;
DROP INDEX IF EXISTS ix_academic_terms_tenant_updated_at;

-- Drop UNIQUE komposit
DO $$
BEGIN
  IF to_regclass('public.academic_terms') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'uq_academic_terms_id_masjid'
        AND conrelid = 'public.academic_terms'::regclass
    ) THEN
      EXECUTE 'ALTER TABLE public.academic_terms DROP CONSTRAINT uq_academic_terms_id_masjid';
    END IF;
  END IF;
END$$;

-- Drop table (sesuai konsep DOWN = revert)
DO $$
BEGIN
  IF to_regclass('public.academic_terms') IS NOT NULL THEN
    EXECUTE 'DROP TABLE public.academic_terms CASCADE';
  END IF;
END$$;

COMMIT;
