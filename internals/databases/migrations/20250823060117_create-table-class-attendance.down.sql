-- =========================================================
-- ========== DOWN: user_class_attendance_sessions =========
-- =========================================================

-- 0) Siapkan regclass tabel kalau ada
DO $$
DECLARE
  rel regclass := to_regclass('public.user_class_attendance_sessions');
BEGIN
  IF rel IS NOT NULL THEN
    -- 1) Drop trigger set_timestamp (jika ada)
    IF EXISTS (
      SELECT 1 FROM pg_trigger t
      WHERE t.tgname = 'set_timestamptz_ucas'
        AND t.tgrelid = rel
    ) THEN
      EXECUTE format('DROP TRIGGER IF EXISTS set_timestamptz_ucas ON %s', rel);
    END IF;

    -- 2) Drop UNIQUE constraint (jika ada)
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conrelid = rel
        AND conname  = 'uq_ucas_session_userclass'
    ) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT uq_ucas_session_userclass', rel);
    END IF;

    -- 3) Drop table (indexes/constraints yang menempel akan ikut hilang)
    EXECUTE format('DROP TABLE IF EXISTS %s', rel);
  END IF;
END$$;

-- 4) Drop indexes “lepas” (jika kamu pernah buat index dengan nama-nama ini terpisah)
DROP INDEX IF EXISTS public.uidx_ucas_session_userclass;
DROP INDEX IF EXISTS public.idx_ucas_masjid_created_at;
DROP INDEX IF EXISTS public.idx_ucas_session_status;
DROP INDEX IF EXISTS public.idx_ucas_userclass_created_at;
DROP INDEX IF EXISTS public.idx_ucas_masjid_session;
DROP INDEX IF EXISTS public.brin_ucas_created_at;
DROP INDEX IF EXISTS public.gin_ucas_search;
DROP INDEX IF EXISTS public.idx_ucas_session_attended;
DROP INDEX IF EXISTS public.idx_ucas_session_absent;

-- Legacy (jaga-jaga nama lama)
DROP INDEX IF EXISTS public.idx_ucae_session_present_only;
DROP INDEX IF EXISTS public.idx_ucas_session_present_only;
DROP INDEX IF EXISTS public.idx_ucae_session_attended;
DROP INDEX IF EXISTS public.idx_ucae_session_absent;
DROP INDEX IF EXISTS public.idx_ucae_session_status;

-- 5) Drop function trigger (aman meskipun table sudah tidak ada)
DROP FUNCTION IF EXISTS public.trg_set_timestamptz_ucas();

-- ❌ Jangan taruh COMMIT di sini; migrate tool yang mengelola transaksi.
