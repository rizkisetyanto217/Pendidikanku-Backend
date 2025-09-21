-- +migrate Down

-- 1) Child paling ujung
DROP TABLE IF EXISTS class_event_urls;

-- 2) Lalu parentnya
DROP TABLE IF EXISTS class_events;

-- 3) Sisanya bebas (independen / parent dari rules)
DROP TABLE IF EXISTS holidays;
DROP TABLE IF EXISTS class_schedule_rules;
DROP TABLE IF EXISTS class_schedules;

-- (opsional) drop enum jika sudah tak dipakai siapapun
DO $$
DECLARE t oid;
BEGIN
  SELECT oid INTO t FROM pg_type WHERE typname='class_delivery_mode_enum';
  IF t IS NOT NULL AND NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid=t) THEN
    EXECUTE 'DROP TYPE class_delivery_mode_enum';
  END IF;

  SELECT oid INTO t FROM pg_type WHERE typname='week_parity_enum';
  IF t IS NOT NULL AND NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid=t) THEN
    EXECUTE 'DROP TYPE week_parity_enum';
  END IF;

  SELECT oid INTO t FROM pg_type WHERE typname='session_status_enum';
  IF t IS NOT NULL AND NOT EXISTS (SELECT 1 FROM pg_depend WHERE refobjid=t) THEN
    EXECUTE 'DROP TYPE session_status_enum';
  END IF;
END$$;
