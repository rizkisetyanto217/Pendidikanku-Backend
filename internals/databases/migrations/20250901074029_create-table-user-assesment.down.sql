-- =========================================
-- DOWN Migration — Submissions & Submission URLs
-- =========================================
BEGIN;

-- Drop trigger + function submission_urls
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_touch_submission_urls_updated_at') THEN
    DROP TRIGGER trg_touch_submission_urls_updated_at ON submission_urls;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_touch_submission_urls_updated_at();

-- Drop table submission_urls
DROP TABLE IF EXISTS submission_urls CASCADE;

-- Drop trigger + function submissions
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_touch_submissions_updated_at') THEN
    DROP TRIGGER trg_touch_submissions_updated_at ON submissions;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_touch_submissions_updated_at();

-- Drop table submissions
DROP TABLE IF EXISTS submissions CASCADE;

COMMIT;
