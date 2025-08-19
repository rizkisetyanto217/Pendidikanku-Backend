-- =========================================================
-- MASTER BUKU
-- =========================================================
CREATE TABLE IF NOT EXISTS books (
  books_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  books_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  books_title     TEXT NOT NULL,
  books_author    TEXT,
  books_edition   TEXT,
  books_publisher TEXT,
  books_isbn      TEXT,
  books_year      INT,
  books_url       TEXT,
  books_image_url TEXT,
  books_image_thumb_url TEXT,

  books_created_at timestamptz NOT NULL DEFAULT now(),
  books_updated_at timestamptz,
  books_deleted_at timestamptz
);

-- Hindari duplikasi judul+edisi di tenant yang sama (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_books_title_edition_per_masjid
  ON books (books_masjid_id, lower(books_title), COALESCE(books_edition,''))
  WHERE books_deleted_at IS NULL;

-- =========================================================
-- RELASI CLASS_SUBJECT <-> BOOKS
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subject_books (
  class_subject_books_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_subject_books_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_subject_books_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,
  class_subject_books_book_id UUID NOT NULL REFERENCES books(books_id) ON DELETE RESTRICT,

  -- opsional: periode pakai; NULL = berlaku terus
  valid_from date,
  valid_to   date,

  -- penandaan
  is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  notes TEXT,

  class_subject_books_created_at timestamptz NOT NULL DEFAULT now(),
  class_subject_books_updated_at timestamptz,
  class_subject_books_deleted_at timestamptz
);

-- Satu buku tidak boleh didaftarkan dua kali ke subject yang sama (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_unique
  ON class_subject_books (class_subject_books_masjid_id,
                          class_subject_books_class_subject_id,
                          class_subject_books_book_id)
  WHERE class_subject_books_deleted_at IS NULL;

-- Maksimal 1 primary per class_subject
CREATE UNIQUE INDEX IF NOT EXISTS uq_csb_one_primary_per_cs
  ON class_subject_books (class_subject_books_class_subject_id)
  WHERE is_primary = TRUE AND class_subject_books_deleted_at IS NULL;

-- =========================================================
-- TRIGGER VALIDASI TENANT
-- Pastikan masjid_id book & class_subject harus sama
-- =========================================================
CREATE OR REPLACE FUNCTION fn_csb_validate_tenant()
RETURNS trigger AS $$
DECLARE
  v_cs_masjid UUID;
  v_b_masjid UUID;
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
    AFTER INSERT OR UPDATE OF class_subject_books_masjid_id,
                            class_subject_books_class_subject_id,
                            class_subject_books_book_id
    ON class_subject_books
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_csb_validate_tenant();
END$$;
