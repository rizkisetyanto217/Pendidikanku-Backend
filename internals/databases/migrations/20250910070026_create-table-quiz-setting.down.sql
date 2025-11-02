BEGIN;

-- =========================================
-- DOWN Migration — QUIZ_SETTINGS
--  - Hapus index, table, lalu ENUMs (aman/idempotent)
-- =========================================

-- 1) Drop indexes (aman meski sudah terhapus karena DROP TABLE)
DROP INDEX IF EXISTS uq_quiz_settings_quiz_alive;
DROP INDEX IF EXISTS idx_qs_school_alive;
DROP INDEX IF EXISTS idx_qs_quiz_alive;
DROP INDEX IF EXISTS brin_qs_created_at;
DROP INDEX IF EXISTS idx_qs_review_window;

-- 2) Drop constraints (opsional; akan ikut terhapus bila tabel di-drop)
ALTER TABLE IF EXISTS quiz_settings
  DROP CONSTRAINT IF EXISTS ck_qs_after_close_requires_close_at;
ALTER TABLE IF EXISTS quiz_settings
  DROP CONSTRAINT IF EXISTS ck_qs_review_window_order;

-- 3) Drop table
DROP TABLE IF EXISTS quiz_settings;

-- 4) Drop ENUM types (hanya jika tidak dipakai objek lain)
DROP TYPE IF EXISTS quiz_result_visibility_enum;
DROP TYPE IF EXISTS quiz_question_order_enum;

COMMIT;
