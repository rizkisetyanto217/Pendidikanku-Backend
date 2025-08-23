BEGIN;


-- =========================================================
-- B. user_class_sections
-- =========================================================
-- FK (harus drop sebelum drop table / unique parent)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_user_class_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      DROP CONSTRAINT fk_ucs_user_class_masjid_pair;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_section_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      DROP CONSTRAINT fk_ucs_section_masjid_pair;
  END IF;
END$$;

-- Indexes
DROP INDEX IF EXISTS uq_user_class_sections_active_per_user_class;
DROP INDEX IF EXISTS idx_user_class_sections_user_class;
DROP INDEX IF EXISTS idx_user_class_sections_section;
DROP INDEX IF EXISTS idx_user_class_sections_assigned_at;
DROP INDEX IF EXISTS idx_user_class_sections_unassigned_at;
DROP INDEX IF EXISTS idx_user_class_sections_masjid;
DROP INDEX IF EXISTS idx_user_class_sections_masjid_active;
DROP INDEX IF EXISTS idx_user_class_sections_section_active;
DROP INDEX IF EXISTS idx_user_class_sections_section_assigned_desc;

-- Table
DROP TABLE IF EXISTS user_class_sections CASCADE;



-- =========================================================
-- C. user_classes
-- =========================================================
-- Trigger & function
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'trg_touch_user_classes'
      AND tgrelid = 'user_classes'::regclass
  ) THEN
    DROP TRIGGER trg_touch_user_classes ON user_classes;
  END IF;
END$$;
DROP FUNCTION IF EXISTS fn_touch_user_classes_updated_at();

-- Indexes
DROP INDEX IF EXISTS uq_uc_active_per_user_class_term;
DROP INDEX IF EXISTS idx_uc_user;
DROP INDEX IF EXISTS idx_uc_class;
DROP INDEX IF EXISTS idx_uc_masjid;
DROP INDEX IF EXISTS idx_uc_term;
DROP INDEX IF EXISTS idx_uc_created_at;
DROP INDEX IF EXISTS idx_uc_user_active;
DROP INDEX IF EXISTS idx_uc_class_active;
DROP INDEX IF EXISTS idx_uc_masjid_active;
DROP INDEX IF EXISTS idx_uc_opening;

-- FKs (drop dulu sebelum drop tabel)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_opening'
  ) THEN
    ALTER TABLE user_classes DROP CONSTRAINT fk_uc_opening;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_term_masjid_pair'
  ) THEN
    ALTER TABLE user_classes DROP CONSTRAINT fk_uc_term_masjid_pair;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_class_masjid_pair'
  ) THEN
    ALTER TABLE user_classes DROP CONSTRAINT fk_uc_class_masjid_pair;
  END IF;
END$$;

-- Unique komposit yang ditambahkan di UP (sebagai syarat FK komposit anak)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_classes_id_masjid'
  ) THEN
    ALTER TABLE user_classes DROP CONSTRAINT uq_user_classes_id_masjid;
  END IF;
END$$;

-- Table
DROP TABLE IF EXISTS user_classes CASCADE;



COMMIT;