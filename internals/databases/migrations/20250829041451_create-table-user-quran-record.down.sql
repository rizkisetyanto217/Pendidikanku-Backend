-- =========================================
-- DOWN Migration: ABSENSI (sessions + urls) & user_quran_records
-- =========================================
BEGIN;

-- =========================================
-- A) ABSENSI: class_attendance_session_url (CHILD) → drop triggers, indexes, table
-- =========================================

-- Triggers & Functions (defensive)
DROP TRIGGER  IF EXISTS trg_casu_tenant_guard       ON class_attendance_session_url;
DROP FUNCTION IF EXISTS fn_casu_tenant_guard();

DROP TRIGGER  IF EXISTS trg_touch_casu_updated_at   ON class_attendance_session_url;
DROP FUNCTION IF EXISTS fn_touch_casu_updated_at();

-- Indexes (defensive)
DROP INDEX IF EXISTS uq_casu_href_per_session_alive;
DROP INDEX IF EXISTS idx_casu_session_alive;
DROP INDEX IF EXISTS idx_casu_created_at;
DROP INDEX IF EXISTS uq_casu_primary_per_session_alive; -- if ever existed

-- Table (child)
DROP TABLE IF EXISTS class_attendance_session_url CASCADE;


-- =========================================
-- B) ABSENSI: class_attendance_sessions (PARENT) → drop triggers, indexes, table
-- =========================================

-- Triggers & Functions (defensive)
DROP TRIGGER  IF EXISTS trg_cas_validate_links       ON class_attendance_sessions;
DROP FUNCTION IF EXISTS fn_cas_validate_links();

DROP TRIGGER  IF EXISTS trg_cas_touch_updated_at     ON class_attendance_sessions;
DROP FUNCTION IF EXISTS fn_touch_class_attendance_sessions_updated_at();

-- Indexes (defensive)
DROP INDEX IF EXISTS idx_cas_section;
DROP INDEX IF EXISTS idx_cas_masjid;
DROP INDEX IF EXISTS idx_cas_date;
DROP INDEX IF EXISTS idx_cas_class_subject;
DROP INDEX IF EXISTS idx_cas_csst;
DROP INDEX IF EXISTS idx_cas_teacher_user;
DROP INDEX IF EXISTS uq_cas_section_date_when_cs_null;
DROP INDEX IF EXISTS uq_cas_section_cs_date_when_cs_not_null;

-- Table (parent)
DROP TABLE IF EXISTS class_attendance_sessions CASCADE;


-- =========================================
-- C) USER QURAN RECORDS (+ images)
--    (sesuai pola yang kamu pakai sebelumnya)
-- =========================================

-- Triggers & Functions
DROP TRIGGER  IF EXISTS set_ts_user_quran_records ON user_quran_records;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_records();

-- Indexes (defensive; table drop akan menghapus juga, tapi eksplisit)
DROP INDEX IF EXISTS idx_user_quran_records_session;
DROP INDEX IF EXISTS idx_uqr_masjid_created_at;
DROP INDEX IF EXISTS idx_uqr_user_created_at;
DROP INDEX IF EXISTS idx_uqr_source_kind;
DROP INDEX IF EXISTS idx_uqr_status_next;
DROP INDEX IF EXISTS gin_uqr_scope_trgm;
DROP INDEX IF EXISTS brin_uqr_created_at;
DROP INDEX IF EXISTS idx_uqr_teacher;
DROP INDEX IF EXISTS uidx_uqr_dedup;

-- CHILD indexes
DROP INDEX IF EXISTS idx_user_quran_record_images_record;
DROP INDEX IF EXISTS idx_uqri_created_at;
DROP INDEX IF EXISTS idx_uqri_uploader_role;

-- CHILD table
DROP TABLE IF EXISTS user_quran_record_images CASCADE;

-- PARENT table
DROP TABLE IF EXISTS user_quran_records CASCADE;

-- Catatan:
-- - Jangan drop EXTENSION pg_trgm / pgcrypto di sini (bisa dipakai object lain)
--   Jika perlu benar-benar bersih, lakukan manual:
--   DROP EXTENSION IF EXISTS pg_trgm;
--   DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
