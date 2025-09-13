
-- =========================================================
-- 3) CERTIFICATE URLS (tautan file sertifikat)
--    (tanpa mime/size/checksum/audience; mirip assessment_urls)
-- =========================================================
CREATE TABLE IF NOT EXISTS certificate_urls (
  certificate_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  certificate_urls_certificate_id UUID NOT NULL
    REFERENCES certificates(certificates_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  certificate_urls_label VARCHAR(120),
  certificate_urls_href  TEXT NOT NULL,

  certificate_urls_trash_url             TEXT,
  certificate_urls_delete_pending_until  TIMESTAMPTZ,

  certificate_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificate_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificate_urls_deleted_at TIMESTAMPTZ,

  -- publikasi sederhana
  certificate_urls_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  certificate_urls_is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  certificate_urls_published_at TIMESTAMPTZ,
  certificate_urls_expires_at   TIMESTAMPTZ,
  certificate_urls_public_slug  VARCHAR(64),
  certificate_urls_public_token VARCHAR(64)
);

-- anti duplikat href per certificate (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_certificate_urls_cert_href_alive
  ON certificate_urls(certificate_urls_certificate_id, certificate_urls_href)
  WHERE certificate_urls_deleted_at IS NULL;

-- filter publikasi cepat
CREATE INDEX IF NOT EXISTS idx_certificate_urls_publish_flags
  ON certificate_urls(certificate_urls_is_published, certificate_urls_is_active)
  WHERE certificate_urls_deleted_at IS NULL;

-- time-scan
CREATE INDEX IF NOT EXISTS brin_certificate_urls_created_at
  ON certificate_urls USING BRIN (certificate_urls_created_at);
