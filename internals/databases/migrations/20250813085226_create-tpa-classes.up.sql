-- =========================================================
-- Supabase/Postgres setup (jika belum)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- untuk gen_random_uuid()

-- =========================================================
-- EXTENSION (Supabase/Postgres)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================================
-- TABEL: classes  (fresh install friendly)
-- =========================================================
CREATE TABLE IF NOT EXISTS classes (
  class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Rekomendasi: NOT NULL + ON DELETE CASCADE agar tenant konsisten.
  -- Jika kamu ingin tetap SET NULL, hapus NOT NULL & ganti CASCADE -> SET NULL.
  class_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_name VARCHAR(120) NOT NULL,
  -- Hapus UNIQUE di level kolom; nanti pakai unique index per masjid
  class_slug VARCHAR(160) NOT NULL,

  class_description TEXT,
  class_level TEXT,         -- "TK A", "Tahfidz", dst
  class_image_url TEXT,     -- opsional

  -- NULL = gratis; >= 0 = tarif per bulan (IDR)
  class_fee_monthly_idr INT
    CHECK (class_fee_monthly_idr IS NULL OR class_fee_monthly_idr >= 0),

  class_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_updated_at TIMESTAMPTZ,
  class_deleted_at TIMESTAMPTZ
);

-- Pastikan ada UNIQUE (class_id, class_masjid_id) agar bisa jadi target FK komposit
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid'
  ) THEN
    ALTER TABLE classes
      ADD CONSTRAINT uq_classes_id_masjid
      UNIQUE (class_id, class_masjid_id);
  END IF;
END$$;


-- =========================================================
-- MIGRASI: hilangkan UNIQUE global di kolom class_slug (jika ada)
-- =========================================================
DO $$
BEGIN
  -- Nama constraint bisa berbeda; coba beberapa kemungkinan umum
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_slug_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_slug_key;
  END IF;
END$$;

-- =========================================================
-- INDEXES
-- =========================================================

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_classes_masjid
  ON classes(class_masjid_id);

CREATE INDEX IF NOT EXISTS idx_classes_active
  ON classes(class_is_active);

CREATE INDEX IF NOT EXISTS idx_classes_created_at
  ON classes(class_created_at DESC);

-- (Opsional) jika sering cari by slug apa adanya
CREATE INDEX IF NOT EXISTS idx_classes_slug
  ON classes(class_slug);

-- UNIQUE per MASJID (case-insensitive), soft-delete aware:
-- slug boleh sama antar masjid, tapi unik dalam satu masjid untuk row yang belum dihapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_active
  ON classes (class_masjid_id, lower(class_slug))
  WHERE class_deleted_at IS NULL;

-- (Opsional) kalau kamu sering cari case-insensitive dan tidak selalu pakai unique index di atas:
-- (biasanya tidak perlu karena sudah covered oleh uq_classes_slug_per_masjid_active)
-- CREATE INDEX IF NOT EXISTS idx_classes_slug_lower_active
--   ON classes (lower(class_slug))
--   WHERE class_deleted_at IS NULL;

-- =========================================================
-- TRIGGER: auto-update updated_at
-- =========================================================
CREATE OR REPLACE FUNCTION fn_classes_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_updated_at := now();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_classes_touch_updated_at') THEN
    DROP TRIGGER trg_classes_touch_updated_at ON classes;
  END IF;

  CREATE TRIGGER trg_classes_touch_updated_at
    BEFORE UPDATE ON classes
    FOR EACH ROW
    EXECUTE FUNCTION fn_classes_touch_updated_at();
END$$;

-- =========================================================
-- (Opsional) Composite unique kalau suatu saat perlu FK komposit ke classes
-- =========================================================
-- ALTER TABLE classes
--   ADD CONSTRAINT uq_classes_id_masjid UNIQUE (class_id, class_masjid_id);



-- =========================================================
-- CLASS_SECTIONS (refactor)
-- =========================================================

-- 1) Fresh install table (pakai NOT NULL + CASCADE yang direkomendasikan)
-- =========================================================
-- CLASS_SECTIONS (refactor, fully idempotent)
-- =========================================================

-- 1) Fresh install table
CREATE TABLE IF NOT EXISTS class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_sections_class_id UUID NOT NULL
    REFERENCES classes(class_id) ON DELETE CASCADE,

  class_sections_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_sections_slug VARCHAR(160) NOT NULL,

  class_sections_teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,

  class_sections_name VARCHAR(100) NOT NULL,  -- "A", "B", "Pagi"
  class_sections_code VARCHAR(50),
  class_sections_capacity INT
    CHECK (class_sections_capacity IS NULL OR class_sections_capacity >= 0),
  class_sections_schedule JSONB,

  class_sections_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_updated_at TIMESTAMPTZ,
  class_sections_deleted_at TIMESTAMPTZ
);

-- 2) Cleanup legacy UNIQUE di kolom slug (jika ada)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_sections_class_sections_slug_key') THEN
    ALTER TABLE class_sections DROP CONSTRAINT class_sections_class_sections_slug_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_sections_slug_key') THEN
    ALTER TABLE class_sections DROP CONSTRAINT class_sections_slug_key;
  END IF;
END$$;

-- 2b) Backfill masjid_id (bila NULL di skema lama), lalu coba set NOT NULL
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM class_sections WHERE class_sections_masjid_id IS NULL) THEN
    UPDATE class_sections cs
       SET class_sections_masjid_id = c.class_masjid_id
      FROM classes c
     WHERE c.class_id = cs.class_sections_class_id
       AND cs.class_sections_masjid_id IS NULL;

    BEGIN
      ALTER TABLE class_sections
        ALTER COLUMN class_sections_masjid_id SET NOT NULL;
    EXCEPTION WHEN others THEN
      -- biarkan nullable jika masih ada baris yang gagal diisi
      NULL;
    END;
  END IF;
END$$;

-- 3) UNIQUE: nama section per class (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name
  ON class_sections (class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

-- 4) UNIQUE: slug per masjid (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_slug_per_masjid_active
  ON class_sections (class_sections_masjid_id, lower(class_sections_slug))
  WHERE class_sections_deleted_at IS NULL;

-- 5) Composite UNIQUE (class_sections_id, class_sections_masjid_id)
--    Gunakan pola "attach constraint ke index" agar tidak bentrok jika index sudah ada.
-- 5a) Pastikan ada unique index (pakai nama yang sama agar bisa di-attach)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sections_id_masjid
  ON class_sections (class_sections_id, class_sections_masjid_id);

-- 5b) Attach sebagai constraint jika constraint-nya belum ada
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_masjid
      UNIQUE USING INDEX uq_class_sections_id_masjid;
  END IF;
END$$;

-- 6) Index umum
CREATE INDEX IF NOT EXISTS idx_sections_class
  ON class_sections(class_sections_class_id);

CREATE INDEX IF NOT EXISTS idx_sections_active
  ON class_sections(class_sections_is_active);

CREATE INDEX IF NOT EXISTS idx_sections_masjid
  ON class_sections(class_sections_masjid_id);

CREATE INDEX IF NOT EXISTS idx_sections_created_at
  ON class_sections(class_sections_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sections_slug
  ON class_sections(class_sections_slug);

CREATE INDEX IF NOT EXISTS idx_sections_teacher
  ON class_sections(class_sections_teacher_id);

-- 7) Trigger: auto-update updated_at
CREATE OR REPLACE FUNCTION fn_class_sections_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_sections_updated_at := now();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sections_touch_updated_at') THEN
    DROP TRIGGER trg_class_sections_touch_updated_at ON class_sections;
  END IF;

  CREATE TRIGGER trg_class_sections_touch_updated_at
    BEFORE UPDATE ON class_sections
    FOR EACH ROW
    EXECUTE FUNCTION fn_class_sections_touch_updated_at();
END$$;

-- 
-- 
-- 

