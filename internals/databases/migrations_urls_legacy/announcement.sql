
-- =========================================================
-- announcement_urls
-- =========================================================
CREATE TABLE IF NOT EXISTS announcement_urls (
  announcement_url_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_url_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_url_announcement_id UUID NOT NULL,

  announcement_url_label       VARCHAR(120),
  announcement_url_href        TEXT NOT NULL,
  announcement_url_trash_url   TEXT,
  announcement_url_delete_pending_until TIMESTAMPTZ,

  announcement_url_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_deleted_at  TIMESTAMPTZ
);

-- Composite FK ke announcements
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_au_announcement_same_tenant') THEN
    ALTER TABLE announcement_urls
      ADD CONSTRAINT fk_au_announcement_same_tenant
      FOREIGN KEY (announcement_url_announcement_id, announcement_url_masjid_id)
      REFERENCES announcements (announcement_id, announcement_masjid_id)
      ON UPDATE CASCADE
      ON DELETE CASCADE;
  END IF;
END$$;

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_urls_id_tenant
  ON announcement_urls (announcement_url_id, announcement_url_masjid_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_urls_announcement_href_live
  ON announcement_urls (announcement_url_announcement_id, lower(announcement_url_href))
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_announcement_live
  ON announcement_urls (announcement_url_announcement_id)
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_masjid_live
  ON announcement_urls (announcement_url_masjid_id)
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_label_trgm_live
  ON announcement_urls USING GIN (announcement_url_label gin_trgm_ops)
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_delete_pending
  ON announcement_urls (announcement_url_delete_pending_until)
  WHERE announcement_url_deleted_at IS NULL;