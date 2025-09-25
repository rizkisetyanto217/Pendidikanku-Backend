-- +migrate Down
-- =========================================================
-- DOWN MIGRATION
-- =========================================================

-- ============================
-- 1) USER CLASS SECTION SUBJECT TEACHERS
-- ============================
DROP INDEX IF EXISTS brin_ucsst_created_at;
DROP INDEX IF EXISTS idx_ucsst_active_alive;
DROP INDEX IF EXISTS idx_ucsst_teacher_alive;
DROP INDEX IF EXISTS idx_ucsst_class_subject_alive;
DROP INDEX IF EXISTS idx_ucsst_section_alive;
DROP INDEX IF EXISTS idx_ucsst_masjid_alive;

DROP INDEX IF EXISTS uq_ucsst_one_active_per_section_subject_alive;
DROP INDEX IF EXISTS uq_ucsst_unique_alive;
DROP INDEX IF EXISTS uq_ucsst_id_tenant;

DROP TABLE IF EXISTS user_class_section_subject_teachers;

-- ============================
-- 2) CLASS SECTION SUBJECT TEACHERS
-- ============================
DROP INDEX IF EXISTS brin_csst_created_at;
DROP INDEX IF EXISTS gin_csst_slug_trgm_alive;
DROP INDEX IF EXISTS uq_csst_slug_per_tenant_alive;

DROP INDEX IF EXISTS idx_csst_room_alive;
DROP INDEX IF EXISTS idx_csst_teacher_alive;
DROP INDEX IF EXISTS idx_csst_class_subject_alive;
DROP INDEX IF EXISTS idx_csst_section_alive;
DROP INDEX IF EXISTS idx_csst_masjid_alive;

DROP INDEX IF EXISTS uq_csst_one_active_per_section_subject_alive;
DROP INDEX IF EXISTS uq_csst_unique_alive;
DROP INDEX IF EXISTS uq_csst_id_tenant;

DROP TABLE IF EXISTS class_section_subject_teachers;

-- ============================
-- 3) PREREQUISITE UNIQUE INDEXES (on referenced tables)
--    (Drop jika memang dibuat oleh migration ini)
-- ============================
DROP INDEX IF EXISTS uq_class_rooms_id_tenant;
DROP INDEX IF EXISTS uq_masjid_teachers_id_tenant;
DROP INDEX IF EXISTS uq_class_subjects_id_tenant;
DROP INDEX IF EXISTS uq_class_sections_id_tenant;

-- ============================
-- 4) EXTENSIONS (opsional)
--    Biasanya JANGAN drop pgcrypto karena dipakai luas.
--    Drop btree_gin & pg_trgm hanya jika memang khusus migrasi ini.
-- ============================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gin') THEN
    DROP EXTENSION btree_gin;
  END IF;
END$$;
