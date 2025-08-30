-- =========================================
-- DOWN MIGRATION (revert ALL-IN tanpa drop "books")
-- =========================================

BEGIN;

-- =========================================
-- 1) books: kembalikan kolom lama + backfill dari book_urls
-- =========================================

-- Tambah kolom lama jika belum ada
ALTER TABLE books
  ADD COLUMN IF NOT EXISTS books_image_url TEXT,
  ADD COLUMN IF NOT EXISTS books_url TEXT;

-- Backfill books_image_url dari cover (ambil cover terbaru/alive)
WITH latest_cover AS (
  SELECT
    u.book_url_book_id AS books_id,
    u.book_url_href,
    ROW_NUMBER() OVER (
      PARTITION BY u.book_url_book_id
      ORDER BY u.book_url_created_at DESC, u.book_url_id DESC
    ) AS rn
  FROM book_urls u
  WHERE u.book_url_deleted_at IS NULL
    AND u.book_url_type = 'cover'
)
UPDATE books b
SET books_image_url = lc.book_url_href
FROM latest_cover lc
WHERE b.books_id = lc.books_id
  AND lc.rn = 1
  AND (b.books_image_url IS NULL OR btrim(b.books_image_url) = '');

-- Backfill books_url dari desc (ambil desc tertua/alive)
WITH first_desc AS (
  SELECT
    u.book_url_book_id AS books_id,
    u.book_url_href,
    ROW_NUMBER() OVER (
      PARTITION BY u.book_url_book_id
      ORDER BY u.book_url_created_at ASC, u.book_url_id ASC
    ) AS rn
  FROM book_urls u
  WHERE u.book_url_deleted_at IS NULL
    AND u.book_url_type = 'desc'
)
UPDATE books b
SET books_url = fd.book_url_href
FROM first_desc fd
WHERE b.books_id = fd.books_id
  AND fd.rn = 1
  AND (b.books_url IS NULL OR btrim(b.books_url) = '');

-- =========================================
-- 2) book_urls: drop trigger, fungsi, indeks, lalu tabel
-- =========================================

-- Drop triggers
DROP TRIGGER IF EXISTS tg_book_urls_tenant_guard ON book_urls;
DROP TRIGGER IF EXISTS tg_touch_book_urls_updated_at ON book_urls;

-- Drop functions
DROP FUNCTION IF EXISTS trg_book_urls_tenant_guard() CASCADE;
DROP FUNCTION IF EXISTS trg_touch_book_urls_updated_at() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS uq_book_urls_href_per_book_alive;
DROP INDEX IF EXISTS idx_book_urls_book_alive;
DROP INDEX IF EXISTS idx_book_urls_type_alive;
DROP INDEX IF EXISTS idx_book_urls_created_at;

-- Drop table
DROP TABLE IF EXISTS book_urls;

-- =========================================
-- 3) class_subject_books: drop trigger, fungsi, indeks, tabel
-- =========================================

-- Drop trigger (constraint trigger)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_csb_validate_tenant') THEN
    EXECUTE 'DROP TRIGGER trg_csb_validate_tenant ON class_subject_books';
  END IF;
END$$;

-- Drop function
DROP FUNCTION IF EXISTS fn_csb_validate_tenant() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS uq_csb_unique;
DROP INDEX IF EXISTS idx_csb_masjid;
DROP INDEX IF EXISTS idx_csb_class_subject;
DROP INDEX IF EXISTS idx_csb_book;
DROP INDEX IF EXISTS idx_csb_active_alive;
DROP INDEX IF EXISTS idx_csb_created_at;

-- Drop table
DROP TABLE IF EXISTS class_subject_books;

-- =========================================
-- 4) books: bersihkan index yang ditambahkan (opsional)
--    (hapus kalau memang mau revert penuh index yang dibuat di UP)
-- =========================================
DROP INDEX IF EXISTS idx_books_created_at;
DROP INDEX IF EXISTS idx_books_masjid_alive;

-- Jika ingin benar-benar balik sebelum penambahan slug unik:
-- (OPSIONAL) hapus unique index slug per masjid
-- DROP INDEX IF EXISTS uq_books_slug_per_masjid_alive;

COMMIT;
