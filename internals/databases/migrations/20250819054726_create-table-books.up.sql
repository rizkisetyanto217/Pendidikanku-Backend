-- +migrate Up
-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops (ILIKE search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- optional combo indexes


/* =========================================================
   TABLE: books  (plural table, singular columns)
   ========================================================= */
CREATE TABLE IF NOT EXISTS books (
  book_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  book_school_id        UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  book_title            TEXT NOT NULL,
  book_author           TEXT,
  book_desc             TEXT,

  book_slug             VARCHAR(160),     -- unique per tenant (alive only)

  -- lokasi file/link
  book_image_url              TEXT,
  book_image_object_key          TEXT,
  book_image_url_old               TEXT,
  book_image_object_key_old      TEXT,
  book_image_delete_pending_until TIMESTAMPTZ,
  
  -- bibliographic (optional)
  book_publisher        TEXT,
  book_publication_year SMALLINT,

  -- timestamps (explicit)
  book_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_deleted_at       TIMESTAMPTZ
);

-- Pair unik utk FK komposit downstream (opsional, aman di-rerun)
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_id_school
  ON books (book_id, book_school_id);

-- Unik slug per school (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_slug_per_school_alive
  ON books (book_school_id, LOWER(book_slug))
  WHERE book_deleted_at IS NULL AND book_slug IS NOT NULL;

-- Lookup umum (alive only)
CREATE INDEX IF NOT EXISTS idx_books_school_alive
  ON books (book_school_id)
  WHERE book_deleted_at IS NULL;

-- Sort terbaru per tenant (alive only)
CREATE INDEX IF NOT EXISTS ix_books_tenant_created_alive
  ON books (book_school_id, book_created_at DESC)
  WHERE book_deleted_at IS NULL;

-- BRIN: time-scan besar
CREATE INDEX IF NOT EXISTS brin_books_created_at
  ON books USING BRIN (book_created_at);

-- Trigram search title/author (alive only)
CREATE INDEX IF NOT EXISTS gin_books_title_trgm_alive
  ON books USING GIN (LOWER(book_title) gin_trgm_ops)
  WHERE book_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_books_author_trgm_alive
  ON books USING GIN (LOWER(book_author) gin_trgm_ops)
  WHERE book_deleted_at IS NULL;


/* =========================================================
   TABLE: class_subject_books  (relasi Subject ↔️ Book per tenant)
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_subject_books (
  class_subject_book_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant
  class_subject_book_school_id        UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- relasi
  class_subject_book_class_subject_id UUID NOT NULL
    REFERENCES class_subjects(class_subject_id) ON DELETE CASCADE,

  class_subject_book_book_id          UUID NOT NULL
    REFERENCES books(book_id) ON DELETE RESTRICT,

  -- human-friendly identifier (opsional)
  class_subject_book_slug             VARCHAR(160),

  class_subject_book_is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  class_subject_book_desc             TEXT,

  /* ============================
     SNAPSHOTS dari books
     (dibekukan saat insert/ubah book_id via trigger)
     ============================ */
  class_subject_book_book_title_snapshot             TEXT,
  class_subject_book_book_author_snapshot            TEXT,
  class_subject_book_book_slug_snapshot              VARCHAR(160),
  class_subject_book_book_publisher_snapshot         TEXT,
  class_subject_book_book_publication_year_snapshot  SMALLINT,
  class_subject_book_book_image_url_snapshot         TEXT,

  class_subject_book_subject_id_snapshot   UUID,
  class_subject_book_subject_code_snapshot VARCHAR(40),
  class_subject_book_subject_name_snapshot VARCHAR(120),
  class_subject_book_subject_slug_snapshot VARCHAR(160),


  -- timestamps (explicit)
  class_subject_book_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_book_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_book_deleted_at       TIMESTAMPTZ
);

-- Unik per (tenant, class_subject, book) — soft-delete aware
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_unique_alive
  ON class_subject_books (
    class_subject_book_school_id,
    class_subject_book_class_subject_id,
    class_subject_book_book_id
  )
  WHERE class_subject_book_deleted_at IS NULL;

-- Unik SLUG per tenant (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_slug_per_tenant_alive
  ON class_subject_books (
    class_subject_book_school_id,
    LOWER(class_subject_book_slug)
  )
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_slug IS NOT NULL;

-- Pencarian slug cepat (alive only)
CREATE INDEX IF NOT EXISTS gin_csb_slug_trgm_alive
  ON class_subject_books USING GIN (LOWER(class_subject_book_slug) gin_trgm_ops)
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_slug IS NOT NULL;

-- Lookup umum (alive only)
CREATE INDEX IF NOT EXISTS idx_csb_school_alive
  ON class_subject_books (class_subject_book_school_id)
  WHERE class_subject_book_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csb_class_subject_alive
  ON class_subject_books (class_subject_book_class_subject_id)
  WHERE class_subject_book_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csb_book_alive
  ON class_subject_books (class_subject_book_book_id)
  WHERE class_subject_book_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csb_active_alive
  ON class_subject_books (class_subject_book_is_active)
  WHERE class_subject_book_deleted_at IS NULL;

-- List cepat per (tenant, subject, active) + sort waktu (alive only)
CREATE INDEX IF NOT EXISTS ix_csb_tenant_subject_active_created_alive
  ON class_subject_books (
    class_subject_book_school_id,
    class_subject_book_class_subject_id,
    class_subject_book_is_active,
    class_subject_book_created_at DESC
  )
  WHERE class_subject_book_deleted_at IS NULL;

-- BRIN: time-scan besar
CREATE INDEX IF NOT EXISTS brin_csb_created_at
  ON class_subject_books USING BRIN (class_subject_book_created_at);

-- Index pencarian pada snapshot judul/slug buku (alive only)
CREATE INDEX IF NOT EXISTS gin_csb_book_title_snap_trgm_alive
  ON class_subject_books USING GIN (LOWER(class_subject_book_book_title_snapshot) gin_trgm_ops)
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_book_title_snapshot IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_csb_book_slug_snap_alive
  ON class_subject_books (LOWER(class_subject_book_book_slug_snapshot))
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_book_slug_snapshot IS NOT NULL;


/* =========================================================
   5) INDEX — bantu pencarian cepat di snapshot SUBJECT
========================================================= */
-- Cari cepat berdasarkan nama subject (trgm, alive only)
CREATE INDEX IF NOT EXISTS gin_csb_subject_name_snap_trgm_alive
  ON class_subject_books USING GIN (LOWER(class_subject_book_subject_name_snapshot) gin_trgm_ops)
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_subject_name_snapshot IS NOT NULL;

-- Lookup cepat slug subject snapshot (case-insensitive, alive only)
CREATE INDEX IF NOT EXISTS idx_csb_subject_slug_snap_alive
  ON class_subject_books (LOWER(class_subject_book_subject_slug_snapshot))
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_subject_slug_snapshot IS NOT NULL;

-- (Opsional) kode subject snapshot (case-insensitive, alive only)
CREATE INDEX IF NOT EXISTS idx_csb_subject_code_snap_alive
  ON class_subject_books (LOWER(class_subject_book_subject_code_snapshot))
  WHERE class_subject_book_deleted_at IS NULL
    AND class_subject_book_subject_code_snapshot IS NOT NULL;



/* =========================================================
   TABLE: book_urls  (assets/links per book)
   ========================================================= */
CREATE TABLE IF NOT EXISTS book_urls (
  book_url_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & owner
  book_url_school_id           UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,
  book_url_book_id             UUID NOT NULL
    REFERENCES books(book_id) ON DELETE CASCADE,

  -- jenis/peran aset
  book_url_kind                VARCHAR(24) NOT NULL,

  -- lokasi file/link
  book_url               TEXT,
  book_url_object_key          TEXT,
  book_url_old               TEXT,
  book_url_object_key_old      TEXT,
  book_url_delete_pending_until TIMESTAMPTZ,

  -- tampilan
  book_url_label               VARCHAR(160),
  book_url_order               INT NOT NULL DEFAULT 0,
  book_url_is_primary          BOOLEAN NOT NULL DEFAULT FALSE,

  -- timestamps & retensi (explicit)
  book_url_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  book_url_deleted_at          TIMESTAMPTZ
);

-- Listing per-book (alive only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_book_urls_by_owner_live
  ON book_urls (
    book_url_book_id,
    book_url_kind,
    book_url_is_primary DESC,
    book_url_order,
    book_url_created_at
  )
  WHERE book_url_deleted_at IS NULL;

-- Filter per tenant (alive only)
CREATE INDEX IF NOT EXISTS ix_book_urls_by_school_live
  ON book_urls (book_url_school_id)
  WHERE book_url_deleted_at IS NULL;

-- Satu primary per (book, kind) (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_book_urls_primary_per_kind_alive
  ON book_urls (book_url_book_id, book_url_kind)
  WHERE book_url_deleted_at IS NULL
    AND book_url_is_primary = TRUE;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_book_urls_purge_due
  ON book_urls (book_url_delete_pending_until)
  WHERE book_url_delete_pending_until IS NOT NULL
    AND (
      (book_url_deleted_at IS NULL  AND book_url_object_key_old IS NOT NULL) OR
      (book_url_deleted_at IS NOT NULL AND book_url_object_key     IS NOT NULL)
    );

-- BRIN: time-scan besar
CREATE INDEX IF NOT EXISTS brin_book_urls_created_at
  ON book_urls USING BRIN (book_url_created_at);