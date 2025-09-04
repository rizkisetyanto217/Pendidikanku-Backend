-- =========================================
-- UP MIGRATION (ALL-IN)
-- =========================================

-- Prasyarat
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================
-- BOOKS (inti + slug)
-- =========================================
CREATE TABLE IF NOT EXISTS books (
  books_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  books_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  books_title     TEXT NOT NULL,
  books_author    TEXT,
  books_desc      TEXT,

  -- URL dipindah ke tabel anak book_urls
  books_slug      VARCHAR(160),

  books_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_deleted_at TIMESTAMPTZ
);

-- (Migration) hapus unique lama jika ada (title+edition) karena kolom edition tak dipakai lagi
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_books_title_edition_per_masjid'
  ) THEN
    EXECUTE 'DROP INDEX uq_books_title_edition_per_masjid';
  END IF;
END$$;

-- Tambahkan kolom slug kalau belum ada
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
     WHERE table_schema='public' AND table_name='books' AND column_name='books_slug'
  ) THEN
    EXECUTE 'ALTER TABLE books ADD COLUMN books_slug VARCHAR(160)';
  END IF;
END$$;

-- Backfill slug untuk baris yang slug-nya NULL/blank, lalu dedup per masjid
DO $$
BEGIN
  -- generate slug dari title
  UPDATE books
     SET books_slug =
       CASE
         WHEN COALESCE(books_title,'') = '' THEN 'book-' || substr(books_id::text,1,8)
         ELSE
           trim(both '-' from
             regexp_replace(
               regexp_replace(
                 regexp_replace(lower(btrim(books_title)), '\s+', '-', 'g'),
               '[^a-z0-9\-]', '-', 'g'),
             '-+', '-', 'g')
           )
       END
   WHERE (books_slug IS NULL OR btrim(books_slug) = '');

  -- antisipasi slug kosong setelah normalisasi → fallback unik
  UPDATE books
     SET books_slug = 'book-' || substr(books_id::text,1,8)
   WHERE (books_slug IS NULL OR btrim(books_slug) = '');

  -- dedup per masjid (case-insensitive, only alive). Tambah suffix -xxxx
  WITH dup AS (
    SELECT
      b.books_id,
      ROW_NUMBER() OVER (
        PARTITION BY b.books_masjid_id, lower(b.books_slug)
        ORDER BY b.books_id
      ) AS rn
    FROM books b
    WHERE b.books_deleted_at IS NULL
      AND b.books_slug IS NOT NULL
      AND btrim(b.books_slug) <> ''
  )
  UPDATE books t
     SET books_slug = t.books_slug || '-' || substr(t.books_id::text,1,4)
  FROM dup
  WHERE t.books_id = dup.books_id
    AND dup.rn > 1;
END$$;

-- Unique slug per masjid (soft-delete aware, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_slug_per_masjid_alive
  ON books (books_masjid_id, lower(books_slug))
  WHERE books_deleted_at IS NULL;

-- Index bantu
CREATE INDEX IF NOT EXISTS idx_books_masjid_alive
  ON books (books_masjid_id)
  WHERE books_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_books_created_at
  ON books (books_created_at DESC);

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

-- =========================================
-- CLASS_SUBJECT_BOOKS (relasi + status aktif)
-- =========================================
CREATE TABLE IF NOT EXISTS class_subject_books (
  class_subject_books_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subject_books_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_subject_books_class_subject_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

  class_subject_books_book_id UUID NOT NULL
    REFERENCES books(books_id) ON DELETE RESTRICT,

  class_subject_books_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  class_subject_books_desc      TEXT,

  class_subject_books_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_books_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_books_deleted_at TIMESTAMPTZ
);

-- Unik per (masjid, class_subject, book) — soft-delete aware
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_unique
  ON class_subject_books (
    class_subject_books_masjid_id,
    class_subject_books_class_subject_id,
    class_subject_books_book_id
  )
  WHERE class_subject_books_deleted_at IS NULL;

-- Index bantu
CREATE INDEX IF NOT EXISTS idx_csb_masjid
  ON class_subject_books (class_subject_books_masjid_id);
CREATE INDEX IF NOT EXISTS idx_csb_class_subject
  ON class_subject_books (class_subject_books_class_subject_id);
CREATE INDEX IF NOT EXISTS idx_csb_book
  ON class_subject_books (class_subject_books_book_id);
CREATE INDEX IF NOT EXISTS idx_csb_active_alive
  ON class_subject_books (class_subject_books_is_active)
  WHERE class_subject_books_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_csb_created_at
  ON class_subject_books (class_subject_books_created_at DESC);

-- Validasi tenant: book, class_subject, dan row harus 1 masjid yang sama
CREATE OR REPLACE FUNCTION fn_csb_validate_tenant()
RETURNS trigger AS $$
DECLARE
  v_cs_masjid UUID;
  v_b_masjid  UUID;
BEGIN
  SELECT class_subjects_masjid_id
    INTO v_cs_masjid
    FROM class_subjects
   WHERE class_subjects_id = NEW.class_subject_books_class_subject_id
     AND class_subjects_deleted_at IS NULL;

  SELECT books_masjid_id
    INTO v_b_masjid
    FROM books
   WHERE books_id = NEW.class_subject_books_book_id
     AND books_deleted_at IS NULL;

  IF v_cs_masjid IS NULL THEN
    RAISE EXCEPTION 'class_subject tidak valid/terhapus';
  END IF;
  IF v_b_masjid IS NULL THEN
    RAISE EXCEPTION 'book tidak valid/terhapus';
  END IF;

  IF v_cs_masjid <> v_b_masjid
     OR v_b_masjid <> NEW.class_subject_books_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch pada class_subject_books';
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_csb_validate_tenant') THEN
    EXECUTE 'DROP TRIGGER trg_csb_validate_tenant ON class_subject_books';
  END IF;

  EXECUTE $tg$
    CREATE CONSTRAINT TRIGGER trg_csb_validate_tenant
      AFTER INSERT OR UPDATE OF
        class_subject_books_masjid_id,
        class_subject_books_class_subject_id,
        class_subject_books_book_id
      ON class_subject_books
      DEFERRABLE INITIALLY DEFERRED
      FOR EACH ROW
      EXECUTE FUNCTION fn_csb_validate_tenant()
  $tg$;
END$$;

-- =========================================
-- END UP
-- =========================================
