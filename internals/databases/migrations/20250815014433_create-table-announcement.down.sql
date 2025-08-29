-- +migrate Down
-- =========================================================
-- DOWN (SAFE & IDEMPOTENT)
-- Target:
--   - announcement_urls
--   - announcements
--   - announcement_themes
-- Catatan:
--  - Tidak pakai BEGIN/COMMIT manual.
--  - Putus FK yang MEREFERENSIKAN target sebelum DROP TABLE.
--  - TIDAK lagi mencoba drop constraint 'uq_class_sections_id_masjid'.
-- =========================================================


-------------------------------
-- Util 1: Putus semua FK yang menunjuk ke sebuah tabel (dipanggil per-target)
-------------------------------
-- Putus semua FK yang MEREFERENSIKAN public.announcement_urls
DO $$
DECLARE
  target regclass := to_regclass('public.announcement_urls');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname AS src_schema, c.relname AS src_table, con.conname AS constraint_name
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.src_schema, r.src_table, r.constraint_name);
    END LOOP;
  END IF;
END$$;

-- Putus semua FK yang MEREFERENSIKAN public.announcements
DO $$
DECLARE
  target regclass := to_regclass('public.announcements');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname AS src_schema, c.relname AS src_table, con.conname AS constraint_name
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.src_schema, r.src_table, r.constraint_name);
    END LOOP;
  END IF;
END$$;

-- Putus semua FK yang MEREFERENSIKAN public.announcement_themes
DO $$
DECLARE
  target regclass := to_regclass('public.announcement_themes');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname AS src_schema, c.relname AS src_table, con.conname AS constraint_name
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.src_schema, r.src_table, r.constraint_name);
    END LOOP;
  END IF;
END$$;


-------------------------------
-- 1) announcement_urls (child of announcements)
-------------------------------
-- Drop trigger
DO $$
BEGIN
  IF to_regclass('public.announcement_urls') IS NOT NULL
     AND EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcement_urls_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_announcement_urls_touch_updated_at ON public.announcement_urls';
  END IF;
END$$;

-- Drop function
DROP FUNCTION IF EXISTS public.fn_announcement_urls_touch_updated_at();

-- Drop indexes (aman walau tabel ikut hilang)
DROP INDEX IF EXISTS public.uq_announcement_urls_id_tenant;
DROP INDEX IF EXISTS public.uq_announcement_urls_announcement_href_live;
DROP INDEX IF EXISTS public.ix_announcement_urls_announcement_live;
DROP INDEX IF EXISTS public.ix_announcement_urls_masjid_live;
DROP INDEX IF EXISTS public.ix_announcement_urls_label_trgm_live;
DROP INDEX IF EXISTS public.ix_announcement_urls_delete_pending;

-- Drop table
DROP TABLE IF EXISTS public.announcement_urls;


-------------------------------
-- 2) announcements (child of announcement_themes; parent of urls yang sudah di-drop)
-------------------------------
-- Putus FK yang ditambahkan di Up (jaga-jaga)
DO $$
BEGIN
  IF to_regclass('public.announcements') IS NOT NULL THEN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_ann_theme_same_tenant') THEN
      EXECUTE 'ALTER TABLE public.announcements DROP CONSTRAINT fk_ann_theme_same_tenant';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_ann_section_same_tenant') THEN
      EXECUTE 'ALTER TABLE public.announcements DROP CONSTRAINT fk_ann_section_same_tenant';
    END IF;
  END IF;
END$$;

-- Drop trigger
DO $$
BEGIN
  IF to_regclass('public.announcements') IS NOT NULL
     AND EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcements_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_announcements_touch_updated_at ON public.announcements';
  END IF;
END$$;

-- Drop function
DROP FUNCTION IF EXISTS public.fn_announcements_touch_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS public.uq_announcements_id_tenant;
DROP INDEX IF EXISTS public.ix_announcements_tenant_date_live;
DROP INDEX IF EXISTS public.ix_announcements_theme_live;
DROP INDEX IF EXISTS public.ix_announcements_section_live;
DROP INDEX IF EXISTS public.ix_announcements_created_by_live;
DROP INDEX IF EXISTS public.ix_announcements_search_gin_live;
DROP INDEX IF EXISTS public.ix_announcements_title_trgm_live;

-- Drop table
DROP TABLE IF EXISTS public.announcements;

-- ⚠️ Tidak menyentuh constraint/UNIQUE pada class_sections lagi untuk menghindari konflik dependensi.


-------------------------------
-- 3) announcement_themes (parent)
-------------------------------
-- Drop trigger
DO $$
BEGIN
  IF to_regclass('public.announcement_themes') IS NOT NULL
     AND EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcement_themes_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_announcement_themes_touch_updated_at ON public.announcement_themes';
  END IF;
END$$;

-- Drop function
DROP FUNCTION IF EXISTS public.fn_announcement_themes_touch_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS public.uq_announcement_themes_tenant_name_live;
DROP INDEX IF EXISTS public.uq_announcement_themes_tenant_slug_live;
DROP INDEX IF EXISTS public.ix_announcement_themes_tenant_active_live;
DROP INDEX IF EXISTS public.ix_announcement_themes_name_trgm_live;
DROP INDEX IF EXISTS public.uq_announcement_themes_id_tenant;

-- Drop table
DROP TABLE IF EXISTS public.announcement_themes;

-- (Tidak drop EXTENSION pg_trgm agar tidak mengganggu objek lain)
