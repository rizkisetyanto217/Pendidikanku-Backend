-- +migrate Down
BEGIN;

-- =========================================================
-- A) class_attendance_sessions — drop trigger/func baru
-- =========================================================

-- 1) Constraint trigger validator (baru)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_validate_links') THEN
    EXECUTE 'DROP TRIGGER trg_cas_validate_links ON class_attendance_sessions';
  END IF;
END$$;

-- 1b) Function validator (baru)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_cas_validate_links') THEN
    EXECUTE 'DROP FUNCTION fn_cas_validate_links() CASCADE';
  END IF;
END$$;

-- 2) Trigger touch updated_at (baru)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_cas_touch_updated_at ON class_attendance_sessions';
  END IF;
END$$;

-- 2b) Function touch updated_at (baru)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_touch_class_attendance_sessions_updated_at') THEN
    EXECUTE 'DROP FUNCTION fn_touch_class_attendance_sessions_updated_at() CASCADE';
  END IF;
END$$;


-- =========================================================
-- B) class_attendance_sessions — drop INDEX/CONSTRAINT baru
-- =========================================================

DROP INDEX IF EXISTS uq_cas_masjid_section_subject_date;
DROP INDEX IF EXISTS idx_cas_section;
DROP INDEX IF EXISTS idx_cas_masjid;
DROP INDEX IF EXISTS idx_cas_date;
DROP INDEX IF EXISTS idx_cas_class_subject;
DROP INDEX IF EXISTS idx_cas_teacher_id;

ALTER TABLE IF EXISTS class_attendance_sessions
  DROP CONSTRAINT IF EXISTS fk_cas_section_masjid_pair,
  DROP CONSTRAINT IF EXISTS fk_cas_class_subject,
  DROP CONSTRAINT IF EXISTS fk_cas_masjid_teacher;


-- =========================================================
-- C) class_attendance_sessions — kolom/indeks lama (CSST) (boleh skip)
--   (Tidak perlu restore kalau ujungnya tabel akan di-DROP)
-- =========================================================
-- (dilewati agar ringkas)


-- =========================================================
-- D) class_attendance_session_url — drop semua & table (anak)
-- =========================================================

-- 1) Drop triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_casu_tenant_guard') THEN
    EXECUTE 'DROP TRIGGER trg_casu_tenant_guard ON class_attendance_session_url';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_casu_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_touch_casu_updated_at ON class_attendance_session_url';
  END IF;
END$$;

-- 2) Drop functions
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_casu_tenant_guard') THEN
    EXECUTE 'DROP FUNCTION fn_casu_tenant_guard() CASCADE';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_touch_casu_updated_at') THEN
    EXECUTE 'DROP FUNCTION fn_touch_casu_updated_at() CASCADE';
  END IF;
END$$;

-- 3) Drop indexes
DROP INDEX IF EXISTS uq_casu_href_per_session_alive;
DROP INDEX IF EXISTS idx_casu_session_alive;
DROP INDEX IF EXISTS idx_casu_created_at;

-- 4) Drop table URL (anak)
DROP TABLE IF EXISTS class_attendance_session_url;


-- =========================================================
-- E) PUTUS SEMUA FK dari tabel lain yang menunjuk ke class_attendance_sessions
-- =========================================================
DO $$
DECLARE r RECORD;
BEGIN
  FOR r IN
    SELECT conrelid::regclass AS referencing_table, conname AS fk_name
    FROM pg_constraint
    WHERE confrelid = 'class_attendance_sessions'::regclass
      AND contype = 'f'
  LOOP
    EXECUTE format('ALTER TABLE %s DROP CONSTRAINT IF EXISTS %I', r.referencing_table, r.fk_name);
  END LOOP;
END$$;


-- =========================================================
-- F) DROP TABLE class_attendance_sessions (tanpa CASCADE; fallback CASCADE)
-- =========================================================
DO $$
BEGIN
  -- Coba drop normal dulu
  EXECUTE 'DROP TABLE IF EXISTS class_attendance_sessions';
EXCEPTION
  WHEN dependent_objects_still_exist THEN
    -- Jika masih ada objek bergantung (mis. view/materialized view), pakai CASCADE
    EXECUTE 'DROP TABLE IF EXISTS class_attendance_sessions CASCADE';
END$$;

COMMIT;
