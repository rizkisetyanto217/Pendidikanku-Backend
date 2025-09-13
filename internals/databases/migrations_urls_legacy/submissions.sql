

-- =========================================================
-- 5) SUBMISSION URLS (lampiran kiriman user)
-- =========================================================
CREATE TABLE IF NOT EXISTS submission_urls (
  submission_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  submission_urls_submission_id UUID NOT NULL
    REFERENCES submissions(submissions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  submission_urls_label VARCHAR(120),
  submission_urls_href  TEXT NOT NULL,

  -- opsi "trash" seperti versi guru
  submission_urls_trash_url            TEXT,
  submission_urls_delete_pending_until TIMESTAMPTZ,

  submission_urls_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  submission_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submission_urls_deleted_at TIMESTAMPTZ
);

-- anti duplikat href per submission (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submission_urls_submission_href
  ON submission_urls(submission_urls_submission_id, submission_urls_href)
  WHERE submission_urls_deleted_at IS NULL;

-- time-scan
CREATE INDEX IF NOT EXISTS brin_submission_urls_created_at
  ON submission_urls USING BRIN (submission_urls_created_at);

