-- +migrate Down
BEGIN;

-- ==========================================
-- 1) USER CLASS SECTION SUBJECT TEACHERS (UC SST)
-- ==========================================
-- Drop indexes (idempotent)
DROP INDEX IF EXISTS brin_ucsst_created_at;
DROP INDEX IF EXISTS gin_ucsst_edits_history;
DROP INDEX IF EXISTS idx_ucsst_active_alive;
DROP INDEX IF EXISTS idx_ucsst_csst_alive;
DROP INDEX IF EXISTS idx_ucsst_student_alive;
DROP INDEX IF EXISTS idx_ucsst_masjid_alive;
DROP INDEX IF EXISTS uq_ucsst_slug_per_tenant_alive;
DROP INDEX IF EXISTS uq_ucsst_one_active_per_student_csst_alive;
DROP INDEX IF EXISTS uq_ucsst_id_tenant;

-- Drop table
DROP TABLE IF EXISTS user_class_section_subject_teachers;

-- ==========================================
-- 2) CLASS SECTION SUBJECT TEACHERS (CSST)
-- ==========================================
-- Drop indexes (idempotent)
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

-- Drop table (akan ikut menjatuhkan constraints/kolom generated)
DROP TABLE IF EXISTS class_section_subject_teachers;

-- ==========================================
-- 3) OPTIONAL: drop ENUM jika sudah tidak dipakai
--    (hanya dijatuhkan bila tidak ada kolom yang bergantung)
-- ==========================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    IF NOT EXISTS (
      SELECT 1
      FROM pg_attribute a
      JOIN pg_class c ON c.oid = a.attrelid
      WHERE a.atttypid = 'class_delivery_mode_enum'::regtype
        AND c.relkind IN ('r','p','v','m','f')  -- table/partition/view/mview/foreign
    ) THEN
      DROP TYPE class_delivery_mode_enum;
    END IF;
  END IF;
END$$;

-- ==========================================
-- 4) JANGAN drop prerequisite indexes & extensions
--    (uq_class_sections_id_tenant, uq_class_subjects_id_tenant,
--     uq_masjid_teachers_id_tenant, uq_class_rooms_id_tenant, pgcrypto, pg_trgm, btree_gin)
-- ==========================================

COMMIT;
