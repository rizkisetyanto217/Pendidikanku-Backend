-- =========================================
-- UP MIGRATION (CLEAN, NO TRIGGERS)
-- =========================================
BEGIN;

-- Prasyarat
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- untuk trigram search (opsional)

-- =========================================
-- TABEL: books (inti + slug)
-- =========================================
CREATE TABLE IF NOT EXISTS books (
  books_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  books_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  books_title      TEXT NOT NULL,
  books_author     TEXT,
  books_desc       TEXT,

  books_slug       VARCHAR(160),           -- unik per masjid (alive)

  -- Bibliografis
  books_publisher          TEXT,
  books_publication_year   SMALLINT,

  books_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_deleted_at TIMESTAMPTZ
);

-- Unik slug per masjid (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_slug_per_masjid_alive
  ON books (books_masjid_id, lower(books_slug))
  WHERE books_deleted_at IS NULL;

-- Index dasar & optimasi query umum
CREATE INDEX IF NOT EXISTS idx_books_masjid_alive
  ON books (books_masjid_id)
  WHERE books_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_books_tenant_created_alive
  ON books (books_masjid_id, books_created_at DESC)
  WHERE books_deleted_at IS NULL;

-- (Opsional) BRIN: efisien untuk range waktu besar
CREATE INDEX IF NOT EXISTS brin_books_created_at
  ON books USING BRIN (books_created_at);

-- (Opsional) Trigram search: title/author
CREATE INDEX IF NOT EXISTS gin_books_title_trgm_alive
  ON books USING GIN (LOWER(books_title) gin_trgm_ops)
  WHERE books_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_books_author_trgm_alive
  ON books USING GIN (LOWER(books_author) gin_trgm_ops)
  WHERE books_deleted_at IS NULL;

-- =========================================
-- TABEL: class_subject_books (relasi + status)
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
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_unique_alive
  ON class_subject_books (
    class_subject_books_masjid_id,
    class_subject_books_class_subject_id,
    class_subject_books_book_id
  )
  WHERE class_subject_books_deleted_at IS NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_csb_masjid
  ON class_subject_books (class_subject_books_masjid_id);

CREATE INDEX IF NOT EXISTS idx_csb_class_subject
  ON class_subject_books (class_subject_books_class_subject_id);

CREATE INDEX IF NOT EXISTS idx_csb_book
  ON class_subject_books (class_subject_books_book_id);

CREATE INDEX IF NOT EXISTS idx_csb_active_alive
  ON class_subject_books (class_subject_books_is_active)
  WHERE class_subject_books_deleted_at IS NULL;

-- List cepat per (masjid, subject, active) dengan sort waktu
CREATE INDEX IF NOT EXISTS ix_csb_tenant_subject_active_created_alive
  ON class_subject_books (
    class_subject_books_masjid_id,
    class_subject_books_class_subject_id,
    class_subject_books_is_active,
    class_subject_books_created_at DESC
  )
  WHERE class_subject_books_deleted_at IS NULL;

-- (Opsional) BRIN
CREATE INDEX IF NOT EXISTS brin_csb_created_at
  ON class_subject_books USING BRIN (class_subject_books_created_at);

COMMIT;

-- =========================================
-- BOOK_URLS — selaras dengan announcement_urls
-- =========================================
CREATE TABLE IF NOT EXISTS book_urls (
  book_url_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  book_url_masjid_id           UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  book_url_book_id             UUID NOT NULL
    REFERENCES books(books_id)     ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'cover','image','preview','attachment','download','purchase','link')
  book_url_kind                VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  book_url_href                TEXT,        -- URL publik (boleh NULL jika pakai object storage)
  book_url_object_key          TEXT,        -- object key aktif di storage
  book_url_object_key_old      TEXT,        -- object key lama (retensi in-place replace)

  -- Tampilan
  book_url_label               VARCHAR(160),
  book_url_order               INT NOT NULL DEFAULT 0,
  book_url_is_primary          BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  book_url_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_deleted_at          TIMESTAMPTZ,          -- soft delete (versi-per-baris)
  book_url_delete_pending_until TIMESTAMPTZ          -- tenggat purge (baris aktif dgn *_old atau baris soft-deleted)
);

-- =========================================
-- INDEXING / OPTIMIZATION (paritas dg announcement_urls)
-- =========================================

-- Lookup per book (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_book_urls_by_owner_live
  ON book_urls (
    book_url_book_id,
    book_url_kind,
    book_url_is_primary DESC,
    book_url_order,
    book_url_created_at
  )
  WHERE book_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_book_urls_by_masjid_live
  ON book_urls (book_url_masjid_id)
  WHERE book_url_deleted_at IS NULL;

-- Satu primary per (book, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_book_urls_primary_per_kind_alive
  ON book_urls (book_url_book_id, book_url_kind)
  WHERE book_url_deleted_at IS NULL
    AND book_url_is_primary = TRUE;

-- Anti-duplikat href per book (live only) — opsional, berguna utk link eksternal
CREATE UNIQUE INDEX IF NOT EXISTS uq_book_urls_book_href_alive
  ON book_urls (book_url_book_id, lower(book_url_href))
  WHERE book_url_deleted_at IS NULL
    AND book_url_href IS NOT NULL;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_book_urls_purge_due
  ON book_urls (book_url_delete_pending_until)
  WHERE book_url_delete_pending_until IS NOT NULL
    AND (
      (book_url_deleted_at IS NULL  AND book_url_object_key_old IS NOT NULL) OR
      (book_url_deleted_at IS NOT NULL AND book_url_object_key     IS NOT NULL)
    );

-- Time-scan (arsip/waktu)
CREATE INDEX IF NOT EXISTS brin_book_urls_created_at
  ON book_urls USING BRIN (book_url_created_at);

-- (opsional) pencarian label cepat (live only)
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- CREATE INDEX IF NOT EXISTS gin_book_urls_label_trgm_alive
--   ON book_urls USING GIN (book_url_label gin_trgm_ops)
--   WHERE book_url_deleted_at IS NULL;
