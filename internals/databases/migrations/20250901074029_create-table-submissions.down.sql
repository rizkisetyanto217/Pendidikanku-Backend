-- =========================================
-- DOWN Migration — SUBMISSIONS & SUBMISSION_URLS (FINAL)
-- =========================================
BEGIN;

-- =========================
-- 1) SUBMISSION_URLS (child)
-- =========================

-- Drop indexes / unique constraints
DROP INDEX IF EXISTS uq_sub_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS uq_sub_urls_submission_href_alive;
DROP INDEX IF EXISTS ix_sub_urls_by_owner_live;
DROP INDEX IF EXISTS ix_sub_urls_by_masjid_live;
DROP INDEX IF EXISTS ix_sub_urls_purge_due;
DROP INDEX IF EXISTS gin_sub_urls_label_trgm_live;

-- Drop table
DROP TABLE IF EXISTS submission_urls;

-- =========================
-- 2) SUBMISSIONS (parent)
-- =========================

-- Drop indexes / unique constraints
DROP INDEX IF EXISTS uq_submissions_assessment_student_alive;
DROP INDEX IF EXISTS idx_submissions_assessment;
DROP INDEX IF EXISTS idx_submissions_student;
DROP INDEX IF EXISTS idx_submissions_masjid;
DROP INDEX IF EXISTS idx_submissions_status_alive;
DROP INDEX IF EXISTS idx_submissions_graded_by_teacher;
DROP INDEX IF EXISTS idx_submissions_submitted_at;
DROP INDEX IF EXISTS brin_submissions_created_at;

-- Drop table
DROP TABLE IF EXISTS submissions;

COMMIT;
