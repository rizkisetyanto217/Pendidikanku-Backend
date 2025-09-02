-- =========================================
-- DOWN MIGRATION (ALL-IN, revert optimized UP) + DROP books
-- =========================================

-- -----------------------------
-- A. DROP indexes di tabel relasi lain (JOIN/FILTER)
-- -----------------------------

-- class_section_subject_teachers
DROP INDEX IF EXISTS idx_csst_section_subject_alive;
DROP INDEX IF EXISTS idx_csst_teacher_alive;

-- class_sections
DROP INDEX IF EXISTS idx_sections_class;

-- class_subjects
DROP INDEX IF EXISTS idx_cs_subject;

-- -----------------------------
-- B. class_subject_books
-- -----------------------------

-- Drop constraint trigger & function (tenant validate)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_csb_validate_tenant') THEN
    EXECUTE 'DROP TRIGGER trg_csb_validate_tenant ON class_subject_books';
  END IF;
EXCEPTION WHEN undefined_table THEN
  -- table mungkin sudah tidak ada; abaikan
END$$;

DROP FUNCTION IF EXISTS fn_csb_validate_tenant() CASCADE;

-- Drop touch updated_at trigger & function
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tg_touch_csb_updated_at') THEN
    EXECUTE 'DROP TRIGGER tg_touch_csb_updated_at ON class_subject_books';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

DROP FUNCTION IF EXISTS trg_touch_csb_updated_at() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS uq_csb_unique;
DROP INDEX IF EXISTS idx_csb_masjid;
DROP INDEX IF EXISTS idx_csb_class_subject;
DROP INDEX IF EXISTS idx_csb_book;
DROP INDEX IF EXISTS idx_csb_active_alive;
DROP INDEX IF EXISTS idx_csb_created_at;
DROP INDEX IF EXISTS idx_csb_masjid_alive;
DROP INDEX IF EXISTS idx_csb_masjid_created_alive;

-- Drop table
DROP TABLE IF EXISTS class_subject_books;

-- -----------------------------
-- C. book_urls
-- -----------------------------

-- Drop tenant guard trigger & function
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tg_book_urls_tenant_guard') THEN
    EXECUTE 'DROP TRIGGER tg_book_urls_tenant_guard ON book_urls';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

DROP FUNCTION IF EXISTS trg_book_urls_tenant_guard() CASCADE;

-- Drop touch updated_at trigger & function
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tg_touch_book_urls_updated_at') THEN
    EXECUTE 'DROP TRIGGER tg_touch_book_urls_updated_at ON book_urls';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

DROP FUNCTION IF EXISTS trg_touch_book_urls_updated_at() CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS uq_book_urls_href_per_book_alive;
DROP INDEX IF EXISTS idx_book_urls_book_alive;
DROP INDEX IF EXISTS idx_book_urls_type_alive;
DROP INDEX IF EXISTS idx_book_urls_created_at;
DROP INDEX IF EXISTS idx_book_urls_book_type_created_alive;

-- Drop table
DROP TABLE IF EXISTS book_urls;

-- -----------------------------
-- D. books (BERSIHKAN JEJAK UP, LALU DROP TABEL)
-- -----------------------------

-- Drop touch updated_at trigger & function
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tg_touch_books_updated_at') THEN
    EXECUTE 'DROP TRIGGER tg_touch_books_updated_at ON books';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

DROP FUNCTION IF EXISTS trg_touch_books_updated_at() CASCADE;

-- Drop indexes di books
DROP INDEX IF EXISTS uq_books_slug_per_masjid_alive;
DROP INDEX IF EXISTS idx_books_masjid_alive;
DROP INDEX IF EXISTS idx_books_created_at;
DROP INDEX IF EXISTS idx_books_masjid_created_alive;
DROP INDEX IF EXISTS gin_books_title_trgm;

-- Optional: drop kolom slug bila belum sempat
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='books' AND column_name='books_slug'
  ) THEN
    EXECUTE 'ALTER TABLE books DROP COLUMN books_slug';
  END IF;
EXCEPTION WHEN undefined_table THEN
END$$;

-- TERAKHIR: DROP TABLE books (pakai CASCADE untuk jaga-jaga jika masih ada FK lain)
DROP TABLE IF EXISTS books CASCADE;

-- Catatan:
-- - UP tidak menambah kolom lama (books_url/books_image_url), jadi DOWN ini tidak menghidupkannya lagi.
-- - Jika kamu ingin juga men-drop extension pg_trgm karena hanya dipakai index di atas,
--   pastikan tidak dipakai tabel lain lalu jalankan: DROP EXTENSION IF EXISTS pg_trgm;
--   (Tidak disertakan di sini agar aman bagi objek lain.)

-- =========================================
-- END DOWN
-- =========================================
