-- =========================================
-- DOWN Migration — Submissions & Submission URLs
-- =========================================
BEGIN;

-- Drop table submissions
DROP TABLE IF EXISTS submissions CASCADE;

COMMIT;
