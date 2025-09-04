-- 20250829_down_all.sql
-- Rollback gabungan untuk:
-- 1) class_pricing_options
-- 2) class_sections
-- 3) classes
-- Catatan: tidak menghapus extension/constraint di tabel lain (mis. class_rooms)

BEGIN;

-- =========================================================
-- 1) CLASS_PRICING_OPTIONS
-- =========================================================

-- Drop views lebih dulu
DROP VIEW IF EXISTS v_class_pricing_options_active;
DROP VIEW IF EXISTS v_class_pricing_options_latest_per_type;
DROP VIEW IF EXISTS v_cpo_latest_per_type;

-- (Opsional) Drop indexes eksplisit
DROP INDEX IF EXISTS idx_class_pricing_options_label_per_class;
DROP INDEX IF EXISTS idx_class_pricing_options_class_type_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_class_id;

-- Drop table
DROP TABLE IF EXISTS class_pricing_options;

-- Drop enum jika tidak dipakai lagi
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_price_type') THEN
    IF NOT EXISTS (
      SELECT 1
      FROM pg_attribute a
      JOIN pg_class c ON a.attrelid = c.oid
      WHERE a.atttypid = 'class_price_type'::regtype
        AND c.relkind = 'r'  -- table
    ) THEN
      DROP TYPE class_price_type;
    END IF;
  END IF;
END$$;

-- =========================================================
-- 2) CLASS_SECTIONS
-- =========================================================

-- Drop FK composite ke class_rooms (room_id, masjid_id)
ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS fk_sections_room_same_masjid;

-- Drop FK ke masjid_teachers
ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS fk_class_sections_teacher;

-- Drop CHECK guard total_students
ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS class_sections_total_students_nonneg_chk;

-- Hapus constraint UNIQUE yang ditambahkan via index
ALTER TABLE IF EXISTS class_sections
  DROP CONSTRAINT IF EXISTS uq_class_sections_id_masjid;

-- (Opsional) Drop indexes eksplisit
DROP INDEX IF EXISTS idx_sections_total_students_alive;
DROP INDEX IF EXISTS idx_sections_class_room;
DROP INDEX IF EXISTS idx_sections_teacher;
DROP INDEX IF EXISTS idx_sections_slug;
DROP INDEX IF EXISTS idx_sections_created_at;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_active;
DROP INDEX IF EXISTS idx_sections_class;

DROP INDEX IF EXISTS uq_class_sections_id_masjid;
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_active;
DROP INDEX IF EXISTS uq_sections_class_name;

-- Drop table
DROP TABLE IF EXISTS class_sections;

-- (Tidak menyentuh constraint UNIQUE di class_rooms: uq_class_rooms_id_masjid)
-- Jika ingin dibersihkan juga:
-- ALTER TABLE IF EXISTS class_rooms
--   DROP CONSTRAINT IF EXISTS uq_class_rooms_id_masjid;

-- =========================================================
-- 3) CLASSES
-- =========================================================

-- (Opsional) Drop indexes eksplisit
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

-- Drop unique constraint komposit (kalau ada)
ALTER TABLE IF EXISTS classes
  DROP CONSTRAINT IF EXISTS uq_classes_id_masjid;

-- Drop table
DROP TABLE IF EXISTS classes;

-- (Tidak menghapus extension pgcrypto / btree_gist / pg_trgm agar aman untuk objek lain)

COMMIT;
