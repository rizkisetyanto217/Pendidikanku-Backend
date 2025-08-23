BEGIN;



-- =========================================================
-- D. class_sections (JANGAN drop table; revert tambahan UP)
-- =========================================================
-- Trigger & function
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'trg_class_sections_touch_updated_at'
      AND tgrelid = 'class_sections'::regclass
  ) THEN
    DROP TRIGGER trg_class_sections_touch_updated_at ON class_sections;
  END IF;
END$$;
DROP FUNCTION IF EXISTS fn_class_sections_touch_updated_at();

-- Constraints & Indexes yang ditambahkan di UP
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections DROP CONSTRAINT uq_class_sections_id_masjid;
  END IF;
END$$;
DROP INDEX IF EXISTS uq_class_sections_id_masjid;

DROP INDEX IF EXISTS uq_sections_class_name;
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_active;

DROP INDEX IF EXISTS idx_sections_class;
DROP INDEX IF EXISTS idx_sections_active;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS idx_sections_created_at;
DROP INDEX IF EXISTS idx_sections_slug;
DROP INDEX IF EXISTS idx_sections_teacher;

-- (Catatan) Kami tidak menambahkan kembali UNIQUE lama di kolom class_sections_slug
-- karena nama constraint asal tidak pasti. Tambahkan sendiri jika diperlukan.



-- =========================================================
-- E. classes (JANGAN drop table; revert tambahan UP)
-- =========================================================
-- Trigger & function
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'trg_classes_touch_updated_at'
      AND tgrelid = 'classes'::regclass
  ) THEN
    DROP TRIGGER trg_classes_touch_updated_at ON classes;
  END IF;
END$$;
DROP FUNCTION IF EXISTS fn_classes_touch_updated_at();

-- Indexes dari UP
DROP INDEX IF EXISTS idx_classes_masjid;
DROP INDEX IF EXISTS idx_classes_active;
DROP INDEX IF EXISTS idx_classes_created_at;
DROP INDEX IF EXISTS idx_classes_slug;
DROP INDEX IF EXISTS uq_classes_slug_per_masjid_active;

-- Constraint komposit yang ditambahkan di UP (hanya jika tidak dipakai objek lain)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid'
  ) THEN
    ALTER TABLE classes DROP CONSTRAINT uq_classes_id_masjid;
  END IF;
END$$;

-- (Catatan) Kami tidak mengembalikan UNIQUE di kolom class_slug yang mungkin
-- ada sebelum UP, karena nama constraint historis bisa berbeda-beda.



-- =========================================================
-- F. academic_terms unique komposit (hanya jika aman)
-- =========================================================
-- Hanya drop jika tidak ada yang depend. Jika gagal karena dependency, abaikan.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid'
  ) THEN
    BEGIN
      ALTER TABLE academic_terms DROP CONSTRAINT uq_academic_terms_id_masjid;
    EXCEPTION WHEN others THEN
      -- mungkin masih dipakai objek lain; abaikan
      NULL;
    END;
  END IF;
END$$;



-- =========================================================
-- G. Extensions (BIARKAN; bisa dipakai objek lain)
-- =========================================================
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
