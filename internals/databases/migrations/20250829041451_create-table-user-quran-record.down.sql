-- =========================================
-- DOWN Migration (Refactor Final) — destructive & idempotent
-- Menghapus:
-- - user_attendance_urls, user_attendance
-- - user_quran_urls, user_quran_records
-- - semua trigger, function, index, dan FK yang ditambahkan di UP
-- Catatan: extensions (pgcrypto, pg_trgm) dibiarkan (tidak di-drop).
-- =========================================
BEGIN;

-- ---------------------------------------------------------
-- D) USER ATTENDANCE URLS (child)
-- ---------------------------------------------------------

-- Drop triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_user_attendance_urls_tenant_guard') THEN
    DROP TRIGGER trg_user_attendance_urls_tenant_guard ON user_attendance_urls;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_user_attendance_urls_updated_at') THEN
    DROP TRIGGER trg_touch_user_attendance_urls_updated_at ON user_attendance_urls;
  END IF;
END$$;

-- Drop FK yang mungkin ditambahkan di UP
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uau_uploader_teacher') THEN
    ALTER TABLE user_attendance_urls DROP CONSTRAINT fk_uau_uploader_teacher;
  END IF;
END$$;

-- Drop indexes
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_uau_attendance_href') THEN
    EXECUTE 'DROP INDEX uq_uau_attendance_href';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uau_attendance_alive') THEN
    EXECUTE 'DROP INDEX idx_uau_attendance_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='brin_uau_created_at') THEN
    EXECUTE 'DROP INDEX brin_uau_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_attendance_urls_attendance') THEN
    EXECUTE 'DROP INDEX idx_user_attendance_urls_attendance';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_attendance_urls_masjid_created_at') THEN
    EXECUTE 'DROP INDEX idx_user_attendance_urls_masjid_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uau_uploader_teacher') THEN
    EXECUTE 'DROP INDEX idx_uau_uploader_teacher';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uau_uploader_user') THEN
    EXECUTE 'DROP INDEX idx_uau_uploader_user';
  END IF;
END$$;

-- Drop functions khusus attendance_urls
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_user_attendance_urls_tenant_guard') THEN
    DROP FUNCTION fn_user_attendance_urls_tenant_guard() CASCADE;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_touch_user_attendance_urls_updated_at') THEN
    DROP FUNCTION fn_touch_user_attendance_urls_updated_at() CASCADE;
  END IF;
END$$;

-- Drop table
DROP TABLE IF EXISTS user_attendance_urls;

-- ---------------------------------------------------------
-- C) USER ATTENDANCE (parent)
-- ---------------------------------------------------------

-- Drop triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_user_attendance_tenant_guard') THEN
    DROP TRIGGER trg_user_attendance_tenant_guard ON user_attendance;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_user_attendance_updated_at') THEN
    DROP TRIGGER trg_touch_user_attendance_updated_at ON user_attendance;
  END IF;
END$$;

-- Drop indexes
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_user_attendance_alive') THEN
    EXECUTE 'DROP INDEX uq_user_attendance_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='brin_user_attendance_created_at') THEN
    EXECUTE 'DROP INDEX brin_user_attendance_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_attendance_session') THEN
    EXECUTE 'DROP INDEX idx_user_attendance_session';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_attendance_user') THEN
    EXECUTE 'DROP INDEX idx_user_attendance_user';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_attendance_status') THEN
    EXECUTE 'DROP INDEX idx_user_attendance_status';
  END IF;
END$$;

-- Drop functions khusus attendance
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_user_attendance_tenant_guard') THEN
    DROP FUNCTION fn_user_attendance_tenant_guard() CASCADE;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_touch_user_attendance_updated_at') THEN
    DROP FUNCTION fn_touch_user_attendance_updated_at() CASCADE;
  END IF;
END$$;

-- Drop table
DROP TABLE IF EXISTS user_attendance;

-- ---------------------------------------------------------
-- B) USER QURAN URLS (child)
-- ---------------------------------------------------------

-- Drop triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_ts_user_quran_urls') THEN
    DROP TRIGGER set_ts_user_quran_urls ON user_quran_urls;
  END IF;
END$$;

-- Drop FK yang mungkin ditambahkan di UP
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uqri_uploader_teacher') THEN
    ALTER TABLE user_quran_urls DROP CONSTRAINT fk_uqri_uploader_teacher;
  END IF;
END$$;

-- Drop indexes
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_uqri_record_href') THEN
    EXECUTE 'DROP INDEX uq_uqri_record_href';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqri_record_alive') THEN
    EXECUTE 'DROP INDEX idx_uqri_record_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='brin_uqri_created_at') THEN
    EXECUTE 'DROP INDEX brin_uqri_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_quran_urls_record') THEN
    EXECUTE 'DROP INDEX idx_user_quran_urls_record';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqri_created_at') THEN
    EXECUTE 'DROP INDEX idx_uqri_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqri_uploader_teacher') THEN
    EXECUTE 'DROP INDEX idx_uqri_uploader_teacher';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqri_uploader_user') THEN
    EXECUTE 'DROP INDEX idx_uqri_uploader_user';
  END IF;
END$$;

-- Drop functions khusus quran_urls
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='trg_set_ts_user_quran_urls') THEN
    DROP FUNCTION trg_set_ts_user_quran_urls() CASCADE;
  END IF;
END$$;

-- Drop table
DROP TABLE IF EXISTS user_quran_urls;

-- ---------------------------------------------------------
-- A) USER QURAN RECORDS (parent)
-- ---------------------------------------------------------

-- Drop trigger
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_ts_user_quran_records') THEN
    DROP TRIGGER set_ts_user_quran_records ON user_quran_records;
  END IF;
END$$;

-- Drop indexes
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='gin_uqr_scope_trgm') THEN
    EXECUTE 'DROP INDEX gin_uqr_scope_trgm';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='brin_uqr_created_at') THEN
    EXECUTE 'DROP INDEX brin_uqr_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_user_quran_records_session') THEN
    EXECUTE 'DROP INDEX idx_user_quran_records_session';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqr_masjid_created_at') THEN
    EXECUTE 'DROP INDEX idx_uqr_masjid_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqr_user_created_at') THEN
    EXECUTE 'DROP INDEX idx_uqr_user_created_at';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqr_source_kind') THEN
    EXECUTE 'DROP INDEX idx_uqr_source_kind';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqr_is_next') THEN
    EXECUTE 'DROP INDEX idx_uqr_is_next';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_uqr_teacher') THEN
    EXECUTE 'DROP INDEX idx_uqr_teacher';
  END IF;
END$$;

-- Drop functions khusus quran_records
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='trg_set_ts_user_quran_records') THEN
    DROP FUNCTION trg_set_ts_user_quran_records() CASCADE;
  END IF;
END$$;

-- Drop table
DROP TABLE IF EXISTS user_quran_records;

COMMIT;
