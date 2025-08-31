-- =========================================
-- DOWN MIGRATION (REVERT ALL-IN, DESTRUCTIVE)
-- =========================================
BEGIN;

-- 1) BOOKS: restore legacy columns from book_urls (optional)
--    (Jika tujuannya benar-benar menghapus books, bagian restore ini bisa dilewati,
--     tapi saya biarkan supaya jika ada data lama tetap sempat tersalin.)
ALTER TABLE books
  ADD COLUMN IF NOT EXISTS books_image_url TEXT,
  ADD COLUMN IF NOT EXISTS books_url       TEXT;

WITH cover AS (
  SELECT u.book_url_book_id AS books_id,
         u.book_url_href    AS href,
         ROW_NUMBER() OVER (
           PARTITION BY u.book_url_book_id
           ORDER BY COALESCE(u.book_url_updated_at, u.book_url_created_at) DESC, u.book_url_id DESC
         ) AS rn
  FROM book_urls u
  WHERE u.book_url_deleted_at IS NULL
    AND u.book_url_type = 'cover'
)
UPDATE books b SET books_image_url = c.href
FROM cover c
WHERE b.books_id = c.books_id AND c.rn = 1;

WITH descs AS (
  SELECT u.book_url_book_id AS books_id,
         u.book_url_href    AS href,
         ROW_NUMBER() OVER (
           PARTITION BY u.book_url_book_id
           ORDER BY COALESCE(u.book_url_updated_at, u.book_url_created_at) DESC, u.book_url_id DESC
         ) AS rn
  FROM book_urls u
  WHERE u.book_url_deleted_at IS NULL
    AND u.book_url_type = 'desc'
)
UPDATE books b SET books_url = d.href
FROM descs d
WHERE b.books_id = d.books_id AND d.rn = 1;

-- Hapus indeks & kolom slug di books (sebelum drop table, supaya aman jika batal drop)
DROP INDEX IF EXISTS uq_books_slug_per_masjid_alive;
DROP INDEX IF EXISTS idx_books_masjid_alive;
DROP INDEX IF EXISTS idx_books_created_at;
DROP INDEX IF EXISTS gin_books_title_trgm;
ALTER TABLE books DROP COLUMN IF EXISTS books_slug;

-- 2) BOOK_URLS: drop triggers, functions, indexes, table
DROP TRIGGER IF EXISTS tg_book_urls_tenant_guard ON book_urls;
DROP TRIGGER IF EXISTS tg_touch_book_urls_updated_at ON book_urls;
DROP FUNCTION IF EXISTS trg_book_urls_tenant_guard();
DROP FUNCTION IF EXISTS trg_touch_book_urls_updated_at();
DROP INDEX IF EXISTS uq_book_urls_href_per_book_alive;
DROP INDEX IF EXISTS idx_book_urls_book_alive;
DROP INDEX IF EXISTS idx_book_urls_type_alive;
DROP INDEX IF EXISTS idx_book_urls_created_at;
DROP TABLE IF EXISTS book_urls;

-- 3) CLASS_SUBJECT_BOOKS: drop trigger, function, indexes, table
DROP TRIGGER IF EXISTS trg_csb_validate_tenant ON class_subject_books;
DROP FUNCTION IF EXISTS fn_csb_validate_tenant();
DROP INDEX IF EXISTS uq_csb_unique;
DROP INDEX IF EXISTS idx_csb_masjid;
DROP INDEX IF EXISTS idx_csb_class_subject;
DROP INDEX IF EXISTS idx_csb_book;
DROP INDEX IF EXISTS idx_csb_active_alive;
DROP INDEX IF EXISTS idx_csb_created_at;
DROP TABLE IF EXISTS class_subject_books;

-- 4) BOOKS: DROP TABLE (destructive)
DROP TABLE IF EXISTS books;

COMMIT;
-- =========================================
-- END DOWN (DESTRUCTIVE)
-- =========================================
