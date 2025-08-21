-- =========================================================
-- =======================  DOWN  ==========================
-- =========================================================
-- Catatan: DOWN menghapus objek. Jalankan hanya bila perlu rollback.
BEGIN;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_carousels_touch ON carousels;
DROP TRIGGER IF EXISTS trg_articles_touch  ON articles;

-- Drop indexes (idempotent: IF EXISTS)
-- quotes
DROP INDEX IF EXISTS uq_quotes_published_order;
DROP INDEX IF EXISTS idx_quotes_created_at;
DROP INDEX IF EXISTS idx_quotes_published_order;

-- carousels
DROP INDEX IF EXISTS idx_carousels_article_id;
DROP INDEX IF EXISTS idx_carousels_active_order;

-- articles
DROP INDEX IF EXISTS uq_articles_masjid_order_active;
DROP INDEX IF EXISTS idx_articles_title_trgm;
DROP INDEX IF EXISTS idx_articles_updated_at;
DROP INDEX IF EXISTS idx_articles_created_at;
DROP INDEX IF EXISTS idx_articles_masjid_active;

-- advices
DROP INDEX IF EXISTS idx_advices_lecture_id;
DROP INDEX IF EXISTS idx_advices_user_created_at;

-- Drop tables (respect FK order)
DROP TABLE IF EXISTS quotes;
DROP TABLE IF EXISTS carousels;
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS advices;

-- Keep shared trigger function & extensions (umumnya tidak di-drop)
-- Jika ingin bersih total, uncomment baris berikut:
-- DROP FUNCTION IF EXISTS fn_touch_updated_at;

COMMIT;