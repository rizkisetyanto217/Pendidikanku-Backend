-- =========================================================
-- DROP TABLES (dengan urutan aman karena FK)
-- =========================================================
DROP TRIGGER IF EXISTS trg_csb_validate_tenant ON class_subject_books;
DROP FUNCTION IF EXISTS fn_csb_validate_tenant();

DROP TABLE IF EXISTS class_subject_books CASCADE;
DROP TABLE IF EXISTS books CASCADE;