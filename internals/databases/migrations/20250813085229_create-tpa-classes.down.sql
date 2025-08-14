-- =========================================
-- DOWN: drop tables in dependency order
-- =========================================

-- 1) Child paling bawah
DROP TABLE IF EXISTS user_class_invoices;

-- 2) Penempatan siswa per section
DROP TABLE IF EXISTS user_class_sections;

-- 3) Enrolment siswa ke class
DROP TABLE IF EXISTS user_classes;

-- 4) Sections
DROP TABLE IF EXISTS class_sections;

-- 5) Master classes
DROP TABLE IF EXISTS classes;
