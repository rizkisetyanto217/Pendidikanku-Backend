

-- =========================================================
-- 3) ASSESSMENT URLS (tanpa mime/size/checksum/audience)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_urls_assessment_id UUID NOT NULL
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  assessment_urls_label VARCHAR(120),
  assessment_urls_href  TEXT NOT NULL,

  -- tambahan kolom baru
  assessment_urls_trash_url           TEXT,
  assessment_urls_delete_pending_until TIMESTAMPTZ,

  assessment_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_urls_deleted_at TIMESTAMPTZ,

  -- publish flags (disederhanakan)
  assessment_urls_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_urls_is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_urls_published_at TIMESTAMPTZ,
  assessment_urls_expires_at   TIMESTAMPTZ,
  assessment_urls_public_slug  VARCHAR(64),
  assessment_urls_public_token VARCHAR(64)
);

-- anti duplikat file per assessment (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_assessment_href
  ON assessment_urls(assessment_urls_assessment_id, assessment_urls_href)
  WHERE assessment_urls_deleted_at IS NULL;

-- filter publikasi cepat
CREATE INDEX IF NOT EXISTS idx_assessment_urls_publish_flags
  ON assessment_urls(assessment_urls_is_published, assessment_urls_is_active)
  WHERE assessment_urls_deleted_at IS NULL;

-- time-scan
CREATE INDEX IF NOT EXISTS brin_assessment_urls_created_at
  ON assessment_urls USING BRIN (assessment_urls_created_at);


COMMIT;
