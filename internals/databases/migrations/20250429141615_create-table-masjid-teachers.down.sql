BEGIN;

-- 1) Drop trigger (jika tabel & trigger ada)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_name = 'masjid_teachers'
  ) THEN
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_set_updated_at_mtj') THEN
      EXECUTE 'DROP TRIGGER trg_set_updated_at_mtj ON masjid_teachers';
    END IF;
  END IF;
END$$;

-- 2) Drop function helper trigger (aman kalau sudah tidak ada)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'set_updated_at_mtj') THEN
    -- coba RESTRICT dulu; kalau masih ada dependensi, fallback CASCADE
    BEGIN
      DROP FUNCTION set_updated_at_mtj() RESTRICT;
    EXCEPTION WHEN dependent_objects_still_exist THEN
      DROP FUNCTION set_updated_at_mtj() CASCADE;
    END;
  END IF;
END$$;

-- 3) Drop indexes (opsional; tabel drop juga akan menghapusnya, ini buat idempotensi)
DROP INDEX IF EXISTS ux_mtj_masjid_user_alive;
DROP INDEX IF EXISTS idx_mtj_user_alive;
DROP INDEX IF EXISTS idx_mtj_masjid_alive;

-- 4) Drop table
-- Catatan: jika ada FK dari tabel lain ke masjid_teachers, perintah ini akan gagal.
-- Dalam kasus itu, drop FK terkait dulu atau gunakan CASCADE dengan sadar.
DROP TABLE IF EXISTS masjid_teachers;

COMMIT;
