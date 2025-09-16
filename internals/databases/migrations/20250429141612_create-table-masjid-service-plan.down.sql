BEGIN;

-- 1) Drop index yang dibuat di UP
DROP INDEX IF EXISTS ux_msp_code_ci;
DROP INDEX IF EXISTS idx_msp_active_alive;
DROP INDEX IF EXISTS idx_msp_active_price_monthly_alive;
DROP INDEX IF EXISTS brin_msp_created_at;

-- 2) Hapus seed yang dimasukkan oleh UP (jika tabel ada)
DO $$
BEGIN
  IF to_regclass('public.masjid_service_plans') IS NOT NULL THEN
    EXECUTE $SQL$
      DELETE FROM public.masjid_service_plans
      WHERE lower(masjid_service_plan_code) IN ('basic','premium','exclusive');
    $SQL$;
  END IF;
END$$;

-- 3) Buang kolom generated CI
ALTER TABLE IF EXISTS masjid_service_plans
  DROP COLUMN IF EXISTS masjid_service_plan_code_ci;

-- 4) Kembalikan index fungsional lama (case-insensitive) bila tabel ada
DO $$
BEGIN
  IF to_regclass('public.masjid_service_plans') IS NOT NULL THEN
    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS ux_msp_code_lower
             ON public.masjid_service_plans (LOWER(masjid_service_plan_code))';
  END IF;
END$$;

COMMIT;
