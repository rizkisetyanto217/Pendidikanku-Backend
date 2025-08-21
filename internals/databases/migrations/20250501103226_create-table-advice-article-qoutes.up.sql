-- =====================================================================
-- Migration: advices, articles, carousels, quotes
-- DB: PostgreSQL
-- Notes:
-- - Idempotent: aman dipanggil berulang.
-- - Optimized indexes: composite, partial (soft delete), and trigram search.
-- - Timestamps: trigger "touch" untuk updated_at.
-- =====================================================================

BEGIN;

-- =========================================================
-- Extensions (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- Trigger function: touch updated_at
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW."updated_at" := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =========================================================
-- ========================  UP  ===========================
-- =========================================================

-- =========================
-- advices
-- =========================
CREATE TABLE IF NOT EXISTS advices (
  advice_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  advice_description TEXT NOT NULL,
  advice_lecture_id  UUID REFERENCES lectures(lecture_id) ON DELETE SET NULL,
  advice_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  advice_created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for advices
-- Filter & sort by user + time
CREATE INDEX IF NOT EXISTS idx_advices_user_created_at
  ON advices(advice_user_id, advice_created_at DESC);

-- Filter by lecture
CREATE INDEX IF NOT EXISTS idx_advices_lecture_id
  ON advices(advice_lecture_id);

-- =========================
-- articles (soft delete + timestamps)
-- =========================
CREATE TABLE IF NOT EXISTS articles (
  article_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  article_title       VARCHAR(255) NOT NULL,
  article_description TEXT NOT NULL,
  article_image_url   TEXT,
  article_order_id    INT,

  -- Masjid scope
  article_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- timestamps
  article_created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  article_updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  article_deleted_at  TIMESTAMP
);

-- Trigger: touch updated_at
DROP TRIGGER IF EXISTS trg_articles_touch ON articles;
CREATE TRIGGER trg_articles_touch
BEFORE UPDATE ON articles
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

-- Indexes for articles
-- Partial: only non-deleted rows
CREATE INDEX IF NOT EXISTS idx_articles_masjid_active
  ON articles(article_masjid_id, article_order_id NULLS LAST)
  WHERE article_deleted_at IS NULL;

-- Time-based queries
CREATE INDEX IF NOT EXISTS idx_articles_created_at
  ON articles(article_created_at DESC)
  WHERE article_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_updated_at
  ON articles(article_updated_at DESC)
  WHERE article_deleted_at IS NULL;

-- Search by title (trigram)
CREATE INDEX IF NOT EXISTS idx_articles_title_trgm
  ON articles USING GIN (article_title gin_trgm_ops)
  WHERE article_deleted_at IS NULL;

-- Optional: keep order unique within masjid for active articles
-- (only if Anda memang ingin memastikan urutan unik per masjid)
--CREATE UNIQUE INDEX IF NOT EXISTS uq_articles_masjid_order_active
--  ON articles(article_masjid_id, article_order_id)
--  WHERE article_deleted_at IS NULL AND article_order_id IS NOT NULL;

-- =========================
-- carousels (optional link ke articles)
-- =========================
CREATE TABLE IF NOT EXISTS carousels (
  carousel_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  carousel_title     VARCHAR(255),
  carousel_caption   TEXT,
  carousel_image_url TEXT NOT NULL,
  carousel_target_url TEXT,
  carousel_type      VARCHAR(50),         -- 'artikel', 'event', 'pengumuman', dst.
  carousel_article_id UUID,               -- relasi opsional ke articles
  carousel_order     INT,
  carousel_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  carousel_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  carousel_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT fk_carousel_article
    FOREIGN KEY (carousel_article_id) REFERENCES articles(article_id) ON DELETE SET NULL
);

-- Trigger: touch updated_at
DROP TRIGGER IF EXISTS trg_carousels_touch ON carousels;
CREATE TRIGGER trg_carousels_touch
BEFORE UPDATE ON carousels
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

-- Indexes for carousels
CREATE INDEX IF NOT EXISTS idx_carousels_active_order
  ON carousels(carousel_is_active, carousel_order NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_carousels_article_id
  ON carousels(carousel_article_id);

-- =========================
-- quotes (simple publish queue)
-- =========================
CREATE TABLE IF NOT EXISTS quotes (
  quote_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quote_text    TEXT NOT NULL,
  is_published  BOOLEAN NOT NULL DEFAULT FALSE,
  display_order INT,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for quotes
-- Active display pipeline
CREATE INDEX IF NOT EXISTS idx_quotes_published_order
  ON quotes(is_published, display_order NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_quotes_created_at
  ON quotes(created_at DESC);

-- Optional: enforce unique display order among published quotes
--CREATE UNIQUE INDEX IF NOT EXISTS uq_quotes_published_order
--  ON quotes(display_order)
--  WHERE is_published = TRUE AND display_order IS NOT NULL;


-- =========================
-- =========================
COMMIT;