BEGIN;

-- Langsung drop tabelnya (triggers ikut terhapus)
DROP TABLE IF EXISTS class_section_subject_teachers CASCADE;

-- Lanjutkan objek lain
-- (functions yang dipakai trigger boleh di-DROP IF EXISTS, itu aman)
DROP FUNCTION IF EXISTS fn_csst_validate_consistency() CASCADE;
DROP FUNCTION IF EXISTS trg_set_timestamp_class_sec_subj_teachers() CASCADE;

-- ... bagian class_subjects & subjects kamu lanjutkan seperti sebelumnya ...
COMMIT;
