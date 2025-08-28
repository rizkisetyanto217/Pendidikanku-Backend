-- =====================================================================
-- Migration: advices, articles, carousels, quotes
-- DB: PostgreSQL (pakai soft delete)
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
-- advices (pakai soft delete)
-- =========================
CREATE TABLE IF NOT EXISTS advices (
  advice_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  advice_description TEXT NOT NULL,
  advice_lecture_id  UUID REFERENCES lectures(lecture_id) ON DELETE SET NULL,
  advice_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  advice_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  advice_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  advice_deleted_at  TIMESTAMPTZ
);

-- Trigger: touch updated_at
DROP TRIGGER IF EXISTS trg_advices_touch ON advices;
CREATE TRIGGER trg_advices_touch
BEFORE UPDATE ON advices
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

-- Indexes for advices
CREATE INDEX IF NOT EXISTS idx_advices_user_created_at
  ON advices(advice_user_id, advice_created_at DESC)
  WHERE advice_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_advices_lecture_id
  ON advices(advice_lecture_id)
  WHERE advice_deleted_at IS NULL;

-- =========================
-- articles (sudah ada deleted_at)
-- =========================
CREATE TABLE IF NOT EXISTS articles (
  article_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  article_title       VARCHAR(255) NOT NULL,
  article_description TEXT NOT NULL,
  article_image_url   TEXT,
  article_order_id    INT,

  article_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  article_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  article_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  article_deleted_at  TIMESTAMPTZ
);

DROP TRIGGER IF EXISTS trg_articles_touch ON articles;
CREATE TRIGGER trg_articles_touch
BEFORE UPDATE ON articles
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

CREATE INDEX IF NOT EXISTS idx_articles_masjid_active
  ON articles(article_masjid_id, article_order_id NULLS LAST)
  WHERE article_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_created_at
  ON articles(article_created_at DESC)
  WHERE article_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_updated_at
  ON articles(article_updated_at DESC)
  WHERE article_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_title_trgm
  ON articles USING GIN (article_title gin_trgm_ops)
  WHERE article_deleted_at IS NULL;

-- =========================
-- carousels (pakai soft delete)
-- =========================
CREATE TABLE IF NOT EXISTS carousels (
  carousel_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  carousel_title      VARCHAR(255),
  carousel_caption    TEXT,
  carousel_image_url  TEXT NOT NULL,
  carousel_target_url TEXT,
  carousel_type       VARCHAR(50),
  carousel_article_id UUID,
  carousel_order      INT,
  carousel_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  carousel_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  carousel_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  carousel_deleted_at TIMESTAMPTZ,

  CONSTRAINT fk_carousel_article
    FOREIGN KEY (carousel_article_id) REFERENCES articles(article_id) ON DELETE SET NULL
);

DROP TRIGGER IF EXISTS trg_carousels_touch ON carousels;
CREATE TRIGGER trg_carousels_touch
BEFORE UPDATE ON carousels
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

CREATE INDEX IF NOT EXISTS idx_carousels_active_order
  ON carousels(carousel_is_active, carousel_order NULLS LAST)
  WHERE carousel_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_carousels_article_id
  ON carousels(carousel_article_id)
  WHERE carousel_deleted_at IS NULL;

-- =========================================================
-- Table: quotes (soft delete + explicit constraints)
-- =========================================================
CREATE TABLE IF NOT EXISTS quotes (
  -- PK
  quote_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Konten
  quote_text        TEXT NOT NULL,
  -- publish pipeline
  quote_is_published  BOOLEAN NOT NULL DEFAULT FALSE,
  quote_display_order INT,

  -- timestamps
  quote_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quote_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quote_deleted_at  TIMESTAMPTZ NULL,

  -- =========================
  -- Explicit constraints
  -- =========================
  CONSTRAINT quotes_text_nonempty CHECK (length(btrim(quote_text)) > 0),
  CONSTRAINT quotes_display_order_pos CHECK (
    quote_display_order IS NULL OR quote_display_order >= 1
  )
);

-- =========================================================
-- Trigger: auto-touch updated_at
-- =========================================================
DROP TRIGGER IF EXISTS trg_quotes_touch ON quotes;
CREATE TRIGGER trg_quotes_touch
BEFORE UPDATE ON quotes
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at();

-- =========================================================
-- Indexes (hanya baris aktif: quote_deleted_at IS NULL)
-- =========================================================
CREATE INDEX IF NOT EXISTS idx_quotes_published_order
  ON quotes (quote_is_published, quote_display_order NULLS LAST)
  WHERE quote_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quotes_created_at
  ON quotes (quote_created_at DESC)
  WHERE quote_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quotes_updated_at
  ON quotes (quote_updated_at DESC)
  WHERE quote_deleted_at IS NULL;

-- (Opsional) enforce unique order
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_quotes_published_order
--   ON quotes (quote_display_order)
--   WHERE quote_is_published = TRUE AND quote_deleted_at IS NULL AND quote_display_order IS NOT NULL;

COMMIT;