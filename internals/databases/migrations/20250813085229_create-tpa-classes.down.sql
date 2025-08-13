-- 20250813_drop_tpa_tables.down.sql
BEGIN;

-- 1) Invoices (child)
DROP TABLE IF EXISTS user_tpa_class_invoices;

-- 2) Penempatan section (child of user_tpa_classes & tpa_class_sections)
DROP TABLE IF EXISTS user_tpa_class_sections;

-- 3) Enrolment ke class (child of tpa_classes)
DROP TABLE IF EXISTS user_tpa_classes;

-- 4) Section/rombongan belajar (child of tpa_classes)
DROP TABLE IF EXISTS tpa_class_sections;

-- 5) Class/level (parent)
DROP TABLE IF EXISTS tpa_classes;

COMMIT;
