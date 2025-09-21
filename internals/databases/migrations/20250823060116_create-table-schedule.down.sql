-- +migrate Down
BEGIN;

-- Tabel (drop otomatis juga semua index/constraint di tabel tsb)
DROP TABLE IF EXISTS holidays;
DROP TABLE IF EXISTS class_schedule_rules;
DROP TABLE IF EXISTS class_schedules;

-- Enums: hapus hanya jika TIDAK ada yang depend
DO $$
DECLARE
  t_oid oid;
BEGIN
  -- week_parity_enum
  SELECT oid INTO t_oid FROM pg_type WHERE typname = 'week_parity_enum';
  IF t_oid IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid = t_oid) THEN
      EXECUTE 'DROP TYPE week_parity_enum';
    END IF;
  END IF;

  -- session_status_enum
  SELECT oid INTO t_oid FROM pg_type WHERE typname = 'session_status_enum';
  IF t_oid IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid = t_oid) THEN
      EXECUTE 'DROP TYPE session_status_enum';
    END IF;
  END IF;
END$$;

-- Catatan: extensions (pgcrypto, pg_trgm) sengaja tidak di-DROP di Down
-- karena bisa dipakai objek lain.

COMMIT;
