-- 20250829_01_classes.down.sql (FIXED ORDER)
BEGIN;

-- =========================================================
-- VIEWs
-- =========================================================
DROP VIEW IF EXISTS v_class_pricing_options_active;
DROP VIEW IF EXISTS v_class_pricing_options_latest_per_type;

-- =========================================================
-- TRIGGER & FUNCTION: class_pricing_options
-- =========================================================
DROP TRIGGER IF EXISTS trg_class_pricing_options_touch_updated_at ON class_pricing_options;
DROP FUNCTION IF EXISTS fn_class_pricing_options_touch_updated_at();

-- =========================================================
-- INDEXES: class_pricing_options
-- =========================================================
DROP INDEX IF EXISTS idx_class_pricing_options_label_per_class;
DROP INDEX IF EXISTS idx_class_pricing_options_class_type_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_class_id;

-- =========================================================
-- CONSTRAINTS: class_pricing_options
-- =========================================================
ALTER TABLE IF EXISTS class_pricing_options
  DROP CONSTRAINT IF EXISTS ck_class_pricing_options_combo;

-- =========================================================
-- TABLE: class_pricing_options & ENUM
-- =========================================================
DROP TABLE IF EXISTS class_pricing_options;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_price_type') THEN
    DROP TYPE class_price_type;
  END IF;
END$$;

-- =========================================================
-- TRIGGER & FUNCTION: class_sections
-- =========================================================
DROP TRIGGER IF EXISTS trg_class_sections_touch_updated_at ON class_sections;
DROP FUNCTION IF EXISTS fn_class_sections_touch_updated_at();

-- =========================================================
-- CONSTRAINTS: class_sections
-- (DROP CONSTRAINT dulu, karena constraint "memiliki" index-nya)
-- =========================================================
-- optional FK kalau pernah dibuat di versi lain
ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS fk_sections_class_tenant;

ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS uq_class_sections_id_masjid;

-- =========================================================
-- INDEXES: class_sections
-- (yang murni index unik/non-unik bisa langsung di-drop)
-- =========================================================
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_active;
DROP INDEX IF EXISTS uq_sections_class_name;

DROP INDEX IF EXISTS idx_sections_teacher;
DROP INDEX IF EXISTS idx_sections_slug;
DROP INDEX IF EXISTS idx_sections_created_at;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_active;
DROP INDEX IF EXISTS idx_sections_class;

-- Kalau constraint di atas tidak otomatis menghapus index-nya (kasus edge),
-- ini aman karena IF EXISTS:
DROP INDEX IF EXISTS uq_class_sections_id_masjid;

-- =========================================================
-- TABLE: class_sections
-- =========================================================
DROP TABLE IF EXISTS class_sections;

-- =========================================================
-- TRIGGER & FUNCTION: classes
-- =========================================================
DROP TRIGGER IF EXISTS trg_classes_touch_updated_at ON classes;
DROP FUNCTION IF EXISTS fn_classes_touch_updated_at();

-- =========================================================
-- INDEXES: classes
-- =========================================================
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

-- =========================================================
-- CONSTRAINTS: classes (optional)
-- =========================================================
ALTER TABLE IF EXISTS classes
  DROP CONSTRAINT IF EXISTS uq_classes_id_masjid;

-- =========================================================
-- COLUMNS: classes (opsional – uncomment jika ingin benar2 revert kolom)
-- =========================================================
-- ALTER TABLE classes DROP COLUMN IF EXISTS class_code;
-- ALTER TABLE classes DROP COLUMN IF EXISTS class_trash_url;
-- ALTER TABLE classes DROP COLUMN IF EXISTS class_delete_pending_until;
-- ALTER TABLE classes DROP COLUMN IF EXISTS class_mode;

COMMIT;
