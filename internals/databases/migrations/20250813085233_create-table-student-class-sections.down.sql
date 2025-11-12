-- +migrate Down
BEGIN;

-- Balikkan pembuatan tabel student_class_sections beserta seluruh index/constraintnya
DROP TABLE IF EXISTS student_class_sections;

COMMIT;
