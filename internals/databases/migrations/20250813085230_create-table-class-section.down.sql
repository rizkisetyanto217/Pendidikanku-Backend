BEGIN;

-- =========================================================
-- LEPAS FK YANG MEREFERENSIKAN class_sections (DIBUAT DI UP)
-- =========================================================
-- user_class_sections → class_sections
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.table_constraints
             WHERE table_name='user_class_sections'
               AND constraint_type='FOREIGN KEY'
               AND constraint_name='fk_ucs_section_masjid_pair') THEN
    ALTER TABLE user_class_sections
      DROP CONSTRAINT fk_ucs_section_masjid_pair;
  END IF;
END$$;

-- class_schedules → class_sections
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.table_constraints
             WHERE table_name='class_schedules'
               AND constraint_type='FOREIGN KEY'
               AND constraint_name='fk_cs_section_same_masjid') THEN
    ALTER TABLE class_schedules
      DROP CONSTRAINT fk_cs_section_same_masjid;
  END IF;
END$$;

-- =========================================================
-- DROP INDEXES BUAT class_sections
-- =========================================================
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_alive;
DROP INDEX IF EXISTS uq_sections_class_name_alive;
DROP INDEX IF EXISTS uq_sections_code_per_class_alive;

DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_class;
DROP INDEX IF EXISTS idx_sections_teacher;
DROP INDEX IF EXISTS idx_sections_class_room;

DROP INDEX IF EXISTS ix_sections_masjid_active_created;
DROP INDEX IF EXISTS ix_sections_class_active_created;
DROP INDEX IF EXISTS ix_sections_teacher_active_created;
DROP INDEX IF EXISTS ix_sections_room_active_created;

DROP INDEX IF EXISTS gin_sections_name_trgm_alive;
DROP INDEX IF EXISTS gin_sections_slug_trgm_alive;

DROP INDEX IF EXISTS idx_sections_group_url_alive;

DROP INDEX IF EXISTS brin_sections_created_at;

-- =========================================================
-- DROP TABLE
-- =========================================================
DROP TABLE IF EXISTS class_sections;

COMMIT;
