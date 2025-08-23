-- 20250823_02_class_term_openings.down.sql

BEGIN;

-- Drop triggers
DO $$
BEGIN
  IF to_regclass('public.class_term_openings') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_trigger
      WHERE tgname = 'trg_touch_class_term_openings'
        AND tgrelid = 'public.class_term_openings'::regclass
    ) THEN
      EXECUTE 'DROP TRIGGER IF EXISTS trg_touch_class_term_openings ON public.class_term_openings';
    END IF;

    IF EXISTS (
      SELECT 1 FROM pg_trigger
      WHERE tgname = 'trg_cto_quota_nonnegative'
        AND tgrelid = 'public.class_term_openings'::regclass
    ) THEN
      EXECUTE 'DROP TRIGGER IF EXISTS trg_cto_quota_nonnegative ON public.class_term_openings';
    END IF;
  END IF;
END$$;

-- Drop indexes
DROP INDEX IF EXISTS ix_cto_tenant_term_open_live;
DROP INDEX IF EXISTS ix_cto_tenant_class_live;
DROP INDEX IF EXISTS ix_cto_reg_window_live;
DROP INDEX IF EXISTS gin_cto_notes_trgm_live;
DROP INDEX IF EXISTS ix_cto_created_at_live;
DROP INDEX IF EXISTS ix_cto_updated_at_live;

-- Drop FKs (ke classes & academic_terms)
DO $$
BEGIN
  IF to_regclass('public.class_term_openings') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'fk_cto_class_masjid_pair'
        AND conrelid = 'public.class_term_openings'::regclass
    ) THEN
      EXECUTE 'ALTER TABLE public.class_term_openings DROP CONSTRAINT fk_cto_class_masjid_pair';
    END IF;

    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conname = 'fk_cto_term_masjid_pair'
        AND conrelid = 'public.class_term_openings'::regclass
    ) THEN
      EXECUTE 'ALTER TABLE public.class_term_openings DROP CONSTRAINT fk_cto_term_masjid_pair';
    END IF;
  END IF;
END$$;

-- Drop table
DO $$
BEGIN
  IF to_regclass('public.class_term_openings') IS NOT NULL THEN
    EXECUTE 'DROP TABLE public.class_term_openings CASCADE';
  END IF;
END$$;

-- Drop functions
DROP FUNCTION IF EXISTS class_term_openings_claim(uuid);
DROP FUNCTION IF EXISTS class_term_openings_release(uuid);
DROP FUNCTION IF EXISTS fn_cto_quota_nonnegative() CASCADE;
DROP FUNCTION IF EXISTS fn_touch_class_term_openings_updated_at() CASCADE;

COMMIT;
