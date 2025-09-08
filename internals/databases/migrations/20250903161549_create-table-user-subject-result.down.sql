-- =========================================
-- DOWN Migration — user_subject_summary (simple)
-- =========================================
BEGIN;

-- 1) Drop indexes (jika ada)
DROP INDEX IF EXISTS uq_user_subject_summary_unique_alive;
DROP INDEX IF EXISTS idx_user_subject_summary_cs_alive;
DROP INDEX IF EXISTS idx_user_subject_summary_csst_alive;
DROP INDEX IF EXISTS idx_user_subject_summary_masjid_grade;
DROP INDEX IF EXISTS brin_user_subject_summary_created_at;

-- (Opsional, hanya jika sebelumnya kamu buat indeks GIN breakdown)
DROP INDEX IF EXISTS gin_uss_breakdown;

-- 2) Drop table
DROP TABLE IF EXISTS user_subject_summary CASCADE;

COMMIT;
