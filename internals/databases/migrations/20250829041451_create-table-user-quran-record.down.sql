-- =========================================
-- DOWN Migration (Refactor Final)
-- =========================================
BEGIN;

-- =========================================
-- D) USER ATTENDANCE URLS (child) - drop
-- =========================================
-- Drop triggers
DROP TRIGGER IF EXISTS trg_user_attendance_urls_tenant_guard ON user_attendance_urls;
DROP TRIGGER IF EXISTS trg_touch_user_attendance_urls_updated_at ON user_attendance_urls;

-- Drop functions
DROP FUNCTION IF EXISTS fn_user_attendance_urls_tenant_guard();
DROP FUNCTION IF EXISTS fn_touch_user_attendance_urls_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS uq_uau_attendance_href;
DROP INDEX IF EXISTS brin_uau_created_at;
DROP INDEX IF EXISTS idx_uau_attendance_alive;
DROP INDEX IF EXISTS idx_uau_uploader_user;
DROP INDEX IF EXISTS idx_uau_uploader_teacher;
DROP INDEX IF EXISTS idx_user_attendance_urls_masjid_created_at;
DROP INDEX IF EXISTS idx_user_attendance_urls_attendance;

-- Drop table
DROP TABLE IF EXISTS user_attendance_urls;

-- =========================================
-- C) USER ATTENDANCE (parent) - drop
-- =========================================
-- Drop triggers
DROP TRIGGER IF EXISTS trg_user_attendance_tenant_guard ON user_attendance;
DROP TRIGGER IF EXISTS trg_touch_user_attendance_updated_at ON user_attendance;

-- Drop functions
DROP FUNCTION IF EXISTS fn_user_attendance_tenant_guard();
DROP FUNCTION IF EXISTS fn_touch_user_attendance_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS brin_user_attendance_created_at;
DROP INDEX IF EXISTS idx_user_attendance_status;
DROP INDEX IF EXISTS idx_user_attendance_user;
DROP INDEX IF EXISTS idx_user_attendance_session;
DROP INDEX IF EXISTS uq_user_attendance_alive;

-- Drop table
DROP TABLE IF EXISTS user_attendance;

-- =========================================
-- B) USER QURAN URLS (child) - drop
-- =========================================
-- Drop trigger & function
DROP TRIGGER IF EXISTS set_ts_user_quran_urls ON user_quran_urls;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_urls();

-- Drop indexes
DROP INDEX IF EXISTS uq_uqri_record_href;
DROP INDEX IF EXISTS brin_uqri_created_at;
DROP INDEX IF EXISTS idx_uqri_record_alive;
DROP INDEX IF EXISTS idx_uqri_uploader_user;
DROP INDEX IF EXISTS idx_uqri_uploader_teacher;
DROP INDEX IF EXISTS idx_uqri_created_at;
DROP INDEX IF EXISTS idx_user_quran_urls_record;

-- Drop table
DROP TABLE IF EXISTS user_quran_urls;

-- =========================================
-- A) USER QURAN RECORDS (parent) - rollback kolom & index
-- =========================================

-- 1) Kembalikan kolom lama (status, next) bila belum ada
ALTER TABLE user_quran_records
  ADD COLUMN IF NOT EXISTS user_quran_records_status VARCHAR(24);

ALTER TABLE user_quran_records
  ADD COLUMN IF NOT EXISTS user_quran_records_next VARCHAR(24);

-- 2) Isi kembali kolom "next" dari boolean "is_next"
UPDATE user_quran_records
SET user_quran_records_next = CASE
  WHEN user_quran_records_is_next IS TRUE THEN 'next'
  WHEN user_quran_records_is_next IS FALSE THEN 'no'
  ELSE NULL
END
WHERE user_quran_records_next IS NULL;

-- 3) Bersihkan index yang ditambahkan UP (aman jika tidak ada)
DROP INDEX IF EXISTS brin_uqr_created_at;
DROP INDEX IF EXISTS gin_uqr_scope_trgm;
DROP INDEX IF EXISTS idx_uqr_teacher;
DROP INDEX IF EXISTS idx_uqr_is_next;
DROP INDEX IF EXISTS idx_uqr_source_kind;
DROP INDEX IF EXISTS idx_uqr_user_created_at;
DROP INDEX IF EXISTS idx_uqr_masjid_created_at;
DROP INDEX IF EXISTS idx_user_quran_records_session;

-- 4) Hapus kolom baru dari UP
ALTER TABLE user_quran_records
  DROP COLUMN IF EXISTS user_quran_records_is_next;

ALTER TABLE user_quran_records
  DROP COLUMN IF EXISTS user_quran_records_score;

-- 5) Kembalikan index gabungan lama (status, next) jika diperlukan
CREATE INDEX IF NOT EXISTS idx_uqr_status_next
  ON user_quran_records(user_quran_records_status, user_quran_records_next);

-- 6) Drop trigger & function updated_at yang ditambahkan UP
DROP TRIGGER IF EXISTS set_ts_user_quran_records ON user_quran_records;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_records();

COMMIT;
