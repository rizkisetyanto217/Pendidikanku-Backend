
-- (opsional) pencarian judul cepat
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- CREATE INDEX IF NOT EXISTS gin_books_title_trgm
--   ON books USING gin (books_title gin_trgm_ops)
--   WHERE books_deleted_at IS NULL;
-- =========================================
-- BOOK_URLS (sederhana: cover, desc, download, purchase)
-- =========================================
CREATE TABLE IF NOT EXISTS book_urls (
  book_url_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  book_url_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  book_url_book_id   UUID NOT NULL REFERENCES books(books_id)   ON DELETE CASCADE,

  book_url_label     VARCHAR(120),
  book_url_type      VARCHAR(20) NOT NULL
                      CHECK (book_url_type IN ('cover','desc','download','purchase')),
  book_url_href      TEXT NOT NULL,

  -- housekeeping optional
  book_url_trash_url             TEXT,
  book_url_delete_pending_until  TIMESTAMPTZ,

  book_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_deleted_at TIMESTAMPTZ
);

-- Tenant guard: masjid_id harus sama dengan masjid_id di books
CREATE OR REPLACE FUNCTION trg_book_urls_tenant_guard()
RETURNS TRIGGER AS $$
DECLARE v_mid UUID;
BEGIN
  SELECT b.books_masjid_id INTO v_mid
  FROM books b
  WHERE b.books_id = NEW.book_url_book_id;

  IF v_mid IS NULL THEN
    RAISE EXCEPTION 'Book not found for id=%', NEW.book_url_book_id;
  END IF;

  IF NEW.book_url_masjid_id IS DISTINCT FROM v_mid THEN
    RAISE EXCEPTION 'Tenant mismatch: book_url_masjid_id(%) != books_masjid_id(%)',
      NEW.book_url_masjid_id, v_mid;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tg_book_urls_tenant_guard ON book_urls;
CREATE TRIGGER tg_book_urls_tenant_guard
BEFORE INSERT OR UPDATE ON book_urls
FOR EACH ROW EXECUTE FUNCTION trg_book_urls_tenant_guard();

-- Indeks unik & bantu
CREATE UNIQUE INDEX IF NOT EXISTS uq_book_urls_href_per_book_alive
  ON book_urls (book_url_book_id, lower(book_url_href))
  WHERE book_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_book_urls_book_alive
  ON book_urls (book_url_book_id)
  WHERE book_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_book_urls_type_alive
  ON book_urls (book_url_type)
  WHERE book_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_book_urls_created_at
  ON book_urls (book_url_created_at DESC);

-- =========================================
-- BACKFILL dari kolom lama di books (jika masih ada)
-- =========================================
DO $$
DECLARE
  has_img_url  boolean;
  has_book_url boolean;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='books' AND column_name='books_image_url'
  ) INTO has_img_url;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='books' AND column_name='books_url'
  ) INTO has_book_url;

  IF has_img_url THEN
    EXECUTE $ins$
      INSERT INTO book_urls (
        book_url_masjid_id, book_url_book_id, book_url_label, book_url_type, book_url_href
      )
      SELECT
        b.books_masjid_id,
        b.books_id,
        'Cover',
        'cover',
        b.books_image_url
      FROM books b
      WHERE b.books_image_url IS NOT NULL
        AND btrim(b.books_image_url) <> ''
        AND NOT EXISTS (
          SELECT 1 FROM book_urls u
          WHERE u.book_url_book_id = b.books_id
            AND u.book_url_deleted_at IS NULL
            AND lower(u.book_url_href) = lower(b.books_image_url)
        );
    $ins$;
  END IF;

  IF has_book_url THEN
    EXECUTE $ins2$
      INSERT INTO book_urls (
        book_url_masjid_id, book_url_book_id, book_url_label, book_url_type, book_url_href
      )
      SELECT
        b.books_masjid_id,
        b.books_id,
        'Deskripsi',
        'desc',
        b.books_url
      FROM books b
      WHERE b.books_url IS NOT NULL
        AND btrim(b.books_url) <> ''
        AND NOT EXISTS (
          SELECT 1 FROM book_urls u
          WHERE u.book_url_book_id = b.books_id
            AND u.book_url_deleted_at IS NULL
            AND lower(u.book_url_href) = lower(b.books_url)
        );
    $ins2$;
  END IF;
END$$;
