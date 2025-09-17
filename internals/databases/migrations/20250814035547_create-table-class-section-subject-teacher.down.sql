-- =========================================================
-- DOWN: Revert creation of CSST & UCSST + indexes
-- =========================================================
BEGIN;

-- ---------------------------------------------------------
-- (OPSIONAL) Lepas FK dari tabel lain yang refer ke UCSST
-- Aktifkan baris ini kalau kamu sebelumnya bikin FK di user_class_subjects
-- ---------------------------------------------------------
-- ALTER TABLE IF EXISTS user_class_subjects
--   DROP CONSTRAINT IF EXISTS fk_user_class_subject_ucsst;

-- ---------------------------------------------------------
-- Drop INDEXES yang dibuat untuk UCSST
-- ---------------------------------------------------------
DROP INDEX IF EXISTS brin_ucsst_created_at;
DROP INDEX IF EXISTS idx_ucsst_active_alive;
DROP INDEX IF EXISTS idx_ucsst_teacher_alive;
DROP INDEX IF EXISTS idx_ucsst_class_subjects_alive;
DROP INDEX IF EXISTS idx_ucsst_section_alive;
DROP INDEX IF EXISTS idx_ucsst_masjid_alive;
DROP INDEX IF EXISTS uq_ucsst_one_active_per_section_subject_alive;
DROP INDEX IF EXISTS uq_ucsst_unique_alive;

-- ---------------------------------------------------------
-- Drop INDEXES yang dibuat untuk CSST
-- (Kalau CSST adalah tabel lama di projectmu, kamu boleh skip bagian ini)
-- ---------------------------------------------------------
DROP INDEX IF EXISTS brin_csst_created_at;
DROP INDEX IF EXISTS idx_csst_active_alive;
DROP INDEX IF EXISTS idx_csst_teacher_alive;
DROP INDEX IF EXISTS idx_csst_class_subjects_alive;
DROP INDEX IF EXISTS idx_csst_section_alive;
DROP INDEX IF EXISTS idx_csst_masjid_alive;
DROP INDEX IF EXISTS uq_csst_one_active_per_section_subject_alive;
DROP INDEX IF EXISTS uq_csst_unique_alive;

-- ---------------------------------------------------------
-- Drop TABLE UCSST
-- ---------------------------------------------------------
DROP TABLE IF EXISTS user_class_section_subject_teachers;

-- ---------------------------------------------------------
-- Drop TABLE CSST
-- ⚠️ Kalau ini adalah tabel lama yang memang sudah ada sebelum UP ini,
--    SEBAIKNYA JANGAN di-drop. Comment baris di bawah jika perlu.
-- ---------------------------------------------------------
DROP TABLE IF EXISTS class_section_subject_teachers;

COMMIT;
