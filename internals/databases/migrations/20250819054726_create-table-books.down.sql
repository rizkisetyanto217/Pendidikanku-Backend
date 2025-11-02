-- =========================================
-- DOWN MIGRATION (revert UP) — urut: book_urls → class_subject_books → books
-- =========================================
BEGIN;

-- -----------------------------
-- 1) BOOK_URLS (child of books/schools)
-- -----------------------------
DROP INDEX IF EXISTS brin_book_urls_created_at;
DROP INDEX IF EXISTS ix_book_urls_purge_due;
DROP INDEX IF EXISTS uq_book_urls_book_href_alive;
DROP INDEX IF EXISTS uq_book_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS ix_book_urls_by_school_live;
DROP INDEX IF EXISTS ix_book_urls_by_owner_live;
-- (Jika di-UP kamu aktifkan index trigram label, turunkan juga:)
DROP INDEX IF EXISTS gin_book_urls_label_trgm_alive;

DROP TABLE IF EXISTS book_urls;

-- -----------------------------
-- 2) CLASS_SUBJECT_BOOKS
-- -----------------------------
-- (UP kamu tidak bikin trigger/function; jadi tak perlu DROP TRIGGER/FUNCTION)
DROP INDEX IF EXISTS uq_csb_unique_alive;
DROP INDEX IF EXISTS idx_csb_school;
DROP INDEX IF EXISTS idx_csb_class_subject;
DROP INDEX IF EXISTS idx_csb_book;
DROP INDEX IF EXISTS idx_csb_active_alive;
DROP INDEX IF EXISTS ix_csb_tenant_subject_active_created_alive;
DROP INDEX IF EXISTS brin_csb_created_at;

DROP TABLE IF EXISTS class_subject_books;

-- -----------------------------
-- 3) BOOKS
-- -----------------------------
DROP INDEX IF EXISTS uq_books_slug_per_school_alive;
DROP INDEX IF EXISTS idx_books_school_alive;
DROP INDEX IF EXISTS ix_books_tenant_created_alive;
DROP INDEX IF EXISTS brin_books_created_at;
DROP INDEX IF EXISTS gin_books_title_trgm_alive;
DROP INDEX IF EXISTS gin_books_author_trgm_alive;

DROP TABLE IF EXISTS books CASCADE;

COMMIT;
