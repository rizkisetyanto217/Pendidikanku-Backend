-- =====================================================================
-- Migration: post_themes, posts, post_likes
-- DB: PostgreSQL
-- =====================================================================

BEGIN;

-- -------------------------------------------------
-- Extensions (idempotent)
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- -------------------------------------------------
-- Trigger functions: touch updated_at kolom spesifik
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION fn_touch_post_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.post_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_post_like_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.post_like_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =================================================
-- =======================  UP  ====================
-- =================================================

-- =========================
-- post_themes
-- =========================
CREATE TABLE IF NOT EXISTS post_themes (
  post_theme_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_theme_name        VARCHAR(100) NOT NULL,
  post_theme_description TEXT,
  post_theme_masjid_id   UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  post_theme_created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexing (theme)
CREATE INDEX IF NOT EXISTS idx_post_themes_name_trgm
  ON post_themes USING GIN (post_theme_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_post_themes_masjid_created
  ON post_themes(post_theme_masjid_id, post_theme_created_at DESC);

-- Opsional: nama tema unik per masjid
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_post_themes_masjid_name
--   ON post_themes(post_theme_masjid_id, post_theme_name);


-- =========================
-- posts  (soft delete + publish)
-- =========================
CREATE TABLE IF NOT EXISTS posts (
  post_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_title        VARCHAR(255) NOT NULL,
  post_content      TEXT NOT NULL,
  post_image_url    TEXT,
  post_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  post_type         VARCHAR(50) NOT NULL DEFAULT 'text',
  post_theme_id     UUID REFERENCES post_themes(post_theme_id) ON DELETE CASCADE,
  post_masjid_id    UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  post_user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
  post_created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  post_updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  post_deleted_at   TIMESTAMP,

  -- Batasi nilai post_type agar konsisten (sesuaikan daftar bila perlu)
  CONSTRAINT chk_posts_type
    CHECK (post_type IN ('text','image','video','link','masjid','motivasi'))
);

-- Trigger: touch post_updated_at
DROP TRIGGER IF EXISTS trg_posts_touch ON posts;
CREATE TRIGGER trg_posts_touch
BEFORE UPDATE ON posts
FOR EACH ROW
EXECUTE FUNCTION fn_touch_post_updated_at();

-- Indexing (posts)
-- 1) Listing default per masjid, hanya yang published & belum dihapus
CREATE INDEX IF NOT EXISTS idx_posts_masjid_pub_created
  ON posts(post_masjid_id, post_is_published, post_created_at DESC)
  WHERE post_deleted_at IS NULL;

-- 2) Listing per tema (aktif saja)
CREATE INDEX IF NOT EXISTS idx_posts_theme_pub_created
  ON posts(post_theme_id, post_is_published, post_created_at DESC)
  WHERE post_deleted_at IS NULL;

-- 3) Filter berdasarkan user (misal halaman "post saya")
CREATE INDEX IF NOT EXISTS idx_posts_user_created
  ON posts(post_user_id, post_created_at DESC)
  WHERE post_deleted_at IS NULL;

-- 4) Search judul & konten (trigram), hanya aktif
CREATE INDEX IF NOT EXISTS idx_posts_title_trgm
  ON posts USING GIN (post_title gin_trgm_ops)
  WHERE post_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_content_trgm
  ON posts USING GIN (post_content gin_trgm_ops)
  WHERE post_deleted_at IS NULL;

-- 5) Opsional: pastikan judul unik per masjid untuk yang aktif
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_posts_masjid_title_active
--   ON posts(post_masjid_id, post_title)
--   WHERE post_deleted_at IS NULL;

-- 6) Penopang query moderasi/arsip (non-published)
CREATE INDEX IF NOT EXISTS idx_posts_unpublished_created
  ON posts(post_is_published, post_created_at DESC)
  WHERE post_deleted_at IS NULL AND post_is_published = FALSE;


-- =========================
-- post_likes (toggle-like)
-- =========================
CREATE TABLE IF NOT EXISTS post_likes (
  post_like_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_like_is_liked  BOOLEAN NOT NULL DEFAULT TRUE,
  post_like_post_id   UUID NOT NULL REFERENCES posts(post_id) ON DELETE CASCADE,
  post_like_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_like_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  post_like_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  CONSTRAINT unique_post_like UNIQUE (post_like_post_id, post_like_user_id)
);

-- Trigger: touch post_like_updated_at
DROP TRIGGER IF EXISTS trg_post_likes_touch ON post_likes;
CREATE TRIGGER trg_post_likes_touch
BEFORE UPDATE ON post_likes
FOR EACH ROW
EXECUTE FUNCTION fn_touch_post_like_updated_at();

-- Indexing (likes)
-- 1) Aggregasi cepat: total like per post (hanya yang liked=TRUE)
CREATE INDEX IF NOT EXISTS idx_post_likes_post_liked
  ON post_likes(post_like_post_id)
  WHERE post_like_is_liked = TRUE;

-- 2) Cek status like user terhadap post
CREATE INDEX IF NOT EXISTS idx_post_likes_user_post
  ON post_likes(post_like_user_id, post_like_post_id);

-- 3) Aktivitas terbaru (riwayat like)
CREATE INDEX IF NOT EXISTS idx_post_likes_updated_at
  ON post_likes(post_like_updated_at DESC);


COMMIT;