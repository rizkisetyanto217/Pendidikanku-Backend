BEGIN;

-- 1) Trigger
DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_service_plans ON public.masjid_service_plans;

-- 2) Function (harus pakai tanda kurung)
DROP FUNCTION IF EXISTS public.set_updated_at_masjid_service_plans();

-- 3) Putus semua FK yang REFER ke tabel ini
DO $$
DECLARE r RECORD;
BEGIN
  FOR r IN
    SELECT conrelid::regclass AS tbl, conname
    FROM pg_constraint
    WHERE confrelid = 'public.masjid_service_plans'::regclass
      AND contype   = 'f'
  LOOP
    EXECUTE format('ALTER TABLE %s DROP CONSTRAINT IF EXISTS %I', r.tbl, r.conname);
  END LOOP;
END$$;

-- 4) Drop table
-- a) Coba tanpa CASCADE dulu:
DROP TABLE IF EXISTS public.masjid_service_plans;

-- b) Jika masih ada dependensi (mis. view) dan kamu mau “paksa”, ganti dengan:
-- DROP TABLE IF EXISTS public.masjid_service_plans CASCADE;

COMMIT;
