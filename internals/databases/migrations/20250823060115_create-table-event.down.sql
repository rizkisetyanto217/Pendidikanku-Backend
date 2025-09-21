-- +migrate Down

-- 0) Putus FK eksternal yang menunjuk ke class_events (jika ada)
--    (default name dari Postgres saat kolom didefinisikan dengan REFERENCES)
ALTER TABLE IF EXISTS class_attendance_sessions
  DROP CONSTRAINT IF EXISTS class_attendance_sessions_class_attendance_sessions_override_event_id_fkey;

-- 1) Child paling ujung
DROP TABLE IF EXISTS class_event_urls;

-- 2) Lalu parent
DROP TABLE IF EXISTS class_events;

-- 3) Terakhir master tema
DROP TABLE IF EXISTS class_event_themes;

-- 4) Hapus enum jika sudah tidak dipakai objek lain
DO $$
DECLARE
  t_oid oid;
BEGIN
  SELECT oid INTO t_oid FROM pg_type WHERE typname = 'class_delivery_mode_enum';
  IF t_oid IS NOT NULL THEN
    -- hanya drop jika tidak ada yang depend
    IF NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid = t_oid) THEN
      EXECUTE 'DROP TYPE class_delivery_mode_enum';
    END IF;
  END IF;
END$$;
