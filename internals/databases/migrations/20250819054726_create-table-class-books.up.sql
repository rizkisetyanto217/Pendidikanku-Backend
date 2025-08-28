-- =========================================
-- Prasyarat
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================
-- BOOKS (sederhana + slug)
-- =========================================
CREATE TABLE IF NOT EXISTS books (
  books_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  books_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  books_title     TEXT NOT NULL,
  books_author    TEXT,
  books_desc      TEXT,
  books_image_url TEXT,
  books_url       TEXT,

  -- baru: slug untuk URL-friendly (unik per masjid, soft-delete aware)
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
    DROP INDEX uq_books_title_edition_per_masjid;
  END IF;
END$$;

-- Tambahkan kolom slug kalau belum ada (aman bila sudah ada)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
     WHERE table_schema='public' AND table_name='books' AND column_name='books_slug'
  ) THEN
    ALTER TABLE books ADD COLUMN books_slug VARCHAR(160);
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
           -- lower, trim, ganti whitespace -> '-', buang non [a-z0-9-], rapikan dash
           trim(both '-' from
             regexp_replace(
               regexp_replace(
                 regexp_replace(lower(btrim(books_title)), '\s+', '-', 'g'),
               '[^a-z0-9\-]', '-', 'g'),
             '-+', '-', 'g')
           )
       END
   WHERE (books_slug IS NULL OR btrim(books_slug) = '');

  -- antisipasi slug kosong setelah normalisasi → isi fallback unik
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
    DROP TRIGGER trg_csb_validate_tenant ON class_subject_books;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_csb_validate_tenant
    AFTER INSERT OR UPDATE OF
      class_subject_books_masjid_id,
      class_subject_books_class_subject_id,
      class_subject_books_book_id
    ON class_subject_books
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_csb_validate_tenant();
END$$;