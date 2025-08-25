-- =========================================================
-- DOWN MIGRATION (classes / class_sections / class_pricing_options)
-- Robust untuk skema lama (cpo_*) maupun nama panjang (class_pricing_options_*)
-- =========================================================

-- 1) VIEWS (drop dulu agar aman rename/kolom baru)
DROP VIEW IF EXISTS v_class_pricing_options_latest_per_type CASCADE;
DROP VIEW IF EXISTS v_cpo_latest_per_type CASCADE;
DROP VIEW IF EXISTS v_class_pricing_options_active CASCADE;

-- 2) TRIGGERS & FUNCTIONS
-- 2a) class_pricing_options
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_pricing_options_touch_updated_at') THEN
    DROP TRIGGER trg_class_pricing_options_touch_updated_at ON class_pricing_options;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cpo_touch_updated_at') THEN
    DROP TRIGGER trg_cpo_touch_updated_at ON class_pricing_options;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_class_pricing_options_touch_updated_at() CASCADE;
DROP FUNCTION IF EXISTS fn_cpo_touch_updated_at() CASCADE;

-- 2b) class_sections
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sections_touch_updated_at') THEN
    DROP TRIGGER trg_class_sections_touch_updated_at ON class_sections;
  END IF;
END$$;
DROP FUNCTION IF EXISTS fn_class_sections_touch_updated_at() CASCADE;

-- 2c) classes
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_classes_touch_updated_at') THEN
    DROP TRIGGER trg_classes_touch_updated_at ON classes;
  END IF;
END$$;
DROP FUNCTION IF EXISTS fn_classes_touch_updated_at() CASCADE;

-- 3) CONSTRAINTS (DROP CONSTRAINT terlebih dahulu agar index owned ikut terhapus)
DO $$
BEGIN
  -- class_pricing_options: CHECK constraints (nama baru/lama)
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_class_pricing_options_combo') THEN
    ALTER TABLE class_pricing_options DROP CONSTRAINT ck_class_pricing_options_combo;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_cpo_combo') THEN
    ALTER TABLE class_pricing_options DROP CONSTRAINT ck_cpo_combo;
  END IF;

  -- classes: UNIQUE (class_id, class_masjid_id)
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid') THEN
    ALTER TABLE classes DROP CONSTRAINT uq_classes_id_masjid;
  END IF;

  -- class_sections: UNIQUE (class_sections_id, class_sections_masjid_id)
  -- Catatan: JANGAN drop index-nya langsung. Drop constraint saja; index owned ikut terhapus.
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_masjid') THEN
    ALTER TABLE class_sections DROP CONSTRAINT uq_class_sections_id_masjid;
  END IF;
END$$;

-- 4) INDEXES (hanya index biasa yang bukan owned by constraint)
-- 4a) class_pricing_options (nama panjang)
DROP INDEX IF EXISTS idx_class_pricing_options_class_type_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_created_at;
DROP INDEX IF EXISTS idx_class_pricing_options_class_id;
DROP INDEX IF EXISTS idx_class_pricing_options_label_per_class;  -- non-unique opsional yang dibuat di UP
DROP INDEX IF EXISTS uq_class_pricing_options_label_per_class;   -- kalau di skema lama masih ada unique

-- 4b) class_pricing_options (legacy cpo_*)
DROP INDEX IF EXISTS idx_cpo_class_type_created_at;
DROP INDEX IF EXISTS idx_cpo_created_at;
DROP INDEX IF EXISTS idx_cpo_class_id;
DROP INDEX IF EXISTS uq_cpo_label_per_class;

-- 4c) class_sections
DROP INDEX IF EXISTS uq_sections_class_name;
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_active;
-- JANGAN drop: DROP INDEX IF EXISTS uq_class_sections_id_masjid;  -- index ini owned by constraint dan sudah dihapus saat DROP CONSTRAINT
DROP INDEX IF EXISTS idx_sections_teacher;
DROP INDEX IF EXISTS idx_sections_slug;
DROP INDEX IF EXISTS idx_sections_created_at;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_active;
DROP INDEX IF EXISTS idx_sections_class;

-- 4d) classes
DROP INDEX IF EXISTS uq_classes_slug_per_masjid_active;
DROP INDEX IF EXISTS idx_classes_slug;
DROP INDEX IF EXISTS idx_classes_created_at;
DROP INDEX IF EXISTS idx_classes_active;
DROP INDEX IF EXISTS idx_classes_masjid;

-- 5) TABLES (anak -> induk)  -- CASCADE untuk bersih
DROP TABLE IF EXISTS class_pricing_options CASCADE;
DROP TABLE IF EXISTS class_sections CASCADE;
DROP TABLE IF EXISTS classes CASCADE;

-- 6) ENUMS
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_price_type') THEN
    DROP TYPE class_price_type;
  END IF;
END$$;

-- Catatan:
-- - pgcrypto extension tidak di-drop.
-- - Karena tabel di-drop, rollback rename kolom (class_pricing_options_* -> cpo_*) tidak diperlukan.
-- - Aman dijalankan berkali-kali (idempotent).
