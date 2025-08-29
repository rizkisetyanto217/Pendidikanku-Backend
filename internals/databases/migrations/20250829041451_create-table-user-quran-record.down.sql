-- =========================================
-- DOWN Migration for user_quran_records (+ images)
-- =========================================
BEGIN;

-- 1) Hapus TRIGGER & FUNCTION (kalau masih ada)
DROP TRIGGER IF EXISTS set_ts_user_quran_records ON user_quran_records;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_records();

-- 2) Hapus INDEX yang dibuat pada UP (opsional, table drop juga akan menghapus)
--    (Tetap di-drop eksplisit untuk jelas & jika ada objek lain yang refer)
DROP INDEX IF EXISTS idx_user_quran_records_session;
DROP INDEX IF EXISTS idx_uqr_masjid_created_at;
DROP INDEX IF EXISTS idx_uqr_user_created_at;
DROP INDEX IF EXISTS idx_uqr_source_kind;
DROP INDEX IF EXISTS idx_uqr_status_next;
DROP INDEX IF EXISTS gin_uqr_scope_trgm;
DROP INDEX IF EXISTS brin_uqr_created_at;
DROP INDEX IF EXISTS idx_uqr_teacher;

-- Jika sebelumnya kamu mengaktifkan dedup opsional
DROP INDEX IF EXISTS uidx_uqr_dedup;

-- 3) Hapus INDEX tabel images (opsional)
DROP INDEX IF EXISTS idx_user_quran_record_images_record;
DROP INDEX IF EXISTS idx_uqri_created_at;
DROP INDEX IF EXISTS idx_uqri_uploader_role;

-- 4) Hapus tabel CHILD lebih dulu
DROP TABLE IF EXISTS user_quran_record_images CASCADE;

-- 5) Terakhir, hapus tabel PARENT
DROP TABLE IF EXISTS user_quran_records CASCADE;

-- Catatan:
-- - Extension pg_trgm TIDAK di-drop (bisa dipakai objek lain).
--   Kalau benar-benar ingin bersih:
--   DROP EXTENSION IF EXISTS pg_trgm;
--   (tidak direkomendasikan di shared DB)

COMMIT;
