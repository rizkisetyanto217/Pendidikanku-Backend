
BEGIN;
-- =================================================
-- ======================  DOWN  ===================
-- =================================================
-- Catatan: jalankan hanya bila perlu rollback

-- Drop triggers
DROP TRIGGER IF EXISTS trg_post_likes_touch ON post_likes;
DROP TRIGGER IF EXISTS trg_posts_touch ON posts;

-- Drop indexes (idempotent)
-- post_likes
DROP INDEX IF EXISTS idx_post_likes_updated_at;
DROP INDEX IF EXISTS idx_post_likes_user_post;
DROP INDEX IF EXISTS idx_post_likes_post_liked;

-- posts
DROP INDEX IF EXISTS uq_posts_masjid_title_active;
DROP INDEX IF EXISTS idx_posts_unpublished_created;
DROP INDEX IF EXISTS idx_posts_content_trgm;
DROP INDEX IF EXISTS idx_posts_title_trgm;
DROP INDEX IF EXISTS idx_posts_user_created;
DROP INDEX IF EXISTS idx_posts_theme_pub_created;
DROP INDEX IF EXISTS idx_posts_masjid_pub_created;

-- post_themes
DROP INDEX IF EXISTS uq_post_themes_masjid_name;
DROP INDEX IF EXISTS idx_post_themes_masjid_created;
DROP INDEX IF EXISTS idx_post_themes_name_trgm;

-- Drop tables (respect FK dependency)
DROP TABLE IF EXISTS post_likes;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS post_themes;

-- (Biarkan fungsi & extensions tetap ada; uncomment bila ingin bersih total)
-- DROP FUNCTION IF EXISTS fn_touch_post_like_updated_at;
-- DROP FUNCTION IF EXISTS fn_touch_post_updated_at;

COMMIT;