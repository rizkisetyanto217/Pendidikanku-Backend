-- =========================================
-- DOWN Migration — User Progress Reports
-- =========================================
BEGIN;

-- -------------------------
-- Drop indexes (Approvals)
-- -------------------------
DROP INDEX IF EXISTS idx_user_progress_report_approvals_report;

-- -------------------------
-- Drop indexes (Receipts)
-- -------------------------
DROP INDEX IF EXISTS uidx_user_progress_report_receipts_report_parent_student;

-- -------------------------
-- Drop indexes (Report ↔ Notes)
-- -------------------------
DROP INDEX IF EXISTS idx_user_progress_report_user_notes_report;

-- -------------------------
-- Drop indexes (Reports)
-- -------------------------
DROP INDEX IF EXISTS uidx_user_progress_reports_student_period_exact;
DROP INDEX IF EXISTS idx_user_progress_reports_status;
DROP INDEX IF EXISTS idx_user_progress_reports_class_period;
DROP INDEX IF EXISTS idx_user_progress_reports_masjid_student_period;

-- -------------------------
-- Drop tables (children first)
-- -------------------------
DROP TABLE IF EXISTS user_progress_report_approvals;
DROP TABLE IF EXISTS user_progress_report_receipts;
DROP TABLE IF EXISTS user_progress_report_user_notes;
DROP TABLE IF EXISTS user_progress_reports;

-- (Opsional) Jika ingin benar-benar bersih:
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
