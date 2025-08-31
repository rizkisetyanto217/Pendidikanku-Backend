-- 20250829_01_classes.down.sql
-- Revert classes, class_sections, class_pricing_options (+enum, triggers, views, indexes)

-- =========================================================
-- 1) CLASS_PRICING_OPTIONS
-- =========================================================
BEGIN;

-- Views
DROP VIEW IF EXISTS v_class_pricing_options_active;
DROP VIEW IF EXISTS v_class_pricing_options_latest_per_type;
DROP VIEW IF EXISTS v_cpo_latest_per_type;

-- Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_pricing_options_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_class_pricing_options_touch_updated_at ON class_pricing_options';
  END IF;
END$$;

-- Functions
DROP FUNCTION IF EXISTS fn_class_pricing_options_touch_updated_at();

-- Indexes
DROP INDEX IF EXISTS idx_class_pricing_options_label_per_class;
DROP INDEX IF EXISTS idx_class_pricing_options_class_type_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_class_id;

-- Table
DROP TABLE IF EXISTS class_pricing_options;

-- Enum (drop hanya jika sudah tidak dipakai)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_depend d
    JOIN pg_type t ON d.refobjid = t.oid
    WHERE t.typname = 'class_price_type'
      AND d.deptype = 'a'
  ) THEN
    DROP TYPE IF EXISTS class_price_type;
  END IF;
END$$;

COMMIT;

-- =========================================================
-- 2) CLASS_SECTIONS
-- =========================================================
BEGIN;

-- Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sections_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_class_sections_touch_updated_at ON class_sections';
  END IF;
END$$;

-- Functions
DROP FUNCTION IF EXISTS fn_class_sections_touch_updated_at();

-- Constraints
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_name='class_sections' AND constraint_name='fk_class_sections_teacher'
  ) THEN
    ALTER TABLE class_sections DROP CONSTRAINT fk_class_sections_teacher;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_name='class_sections' AND constraint_name='uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections DROP CONSTRAINT uq_class_sections_id_masjid;
  END IF;
END$$;

-- Indexes
DROP INDEX IF EXISTS uq_class_sections_id_masjid;
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_active;
DROP INDEX IF EXISTS uq_sections_class_name;
DROP INDEX IF EXISTS idx_sections_teacher;
DROP INDEX IF EXISTS idx_sections_slug;
DROP INDEX IF EXISTS idx_sections_created_at;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_active;
DROP INDEX IF EXISTS idx_sections_class;

-- Table
DROP TABLE IF EXISTS class_sections;

COMMIT;

-- =========================================================
-- 3) CLASSES
-- =========================================================
BEGIN;

-- Triggers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_classes_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_classes_touch_updated_at ON classes';
  END IF;
END$$;

-- Functions
DROP FUNCTION IF EXISTS fn_classes_touch_updated_at();

-- Constraints
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_name='classes' AND constraint_name='uq_classes_id_masjid'
  ) THEN
    ALTER TABLE classes DROP CONSTRAINT uq_classes_id_masjid;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_slug_key;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_slug_key;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_code_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_code_key;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_code_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_code_key;
  END IF;
END$$;

-- Indexes
DROP INDEX IF EXISTS idx_classes_masjid_mode_visible;
DROP INDEX IF EXISTS idx_classes_masjid_code_visible;
DROP INDEX IF EXISTS idx_classes_masjid_slug_visible;
DROP INDEX IF EXISTS idx_classes_pending_until;
DROP INDEX IF EXISTS idx_classes_visible;
DROP INDEX IF EXISTS uq_classes_code_per_masjid_active;
DROP INDEX IF EXISTS uq_classes_slug_per_masjid_active;
DROP INDEX IF EXISTS idx_classes_mode_lower;
DROP INDEX IF EXISTS idx_classes_code;
DROP INDEX IF EXISTS idx_classes_slug;
DROP INDEX IF EXISTS idx_classes_created_at;
DROP INDEX IF EXISTS idx_classes_active;
DROP INDEX IF EXISTS idx_classes_masjid;

-- Table
DROP TABLE IF EXISTS classes;

COMMIT;
