-- 20250829_01_classes.up.sql (REVISED & IDEMPOTENT, schema default)

-- =========================================================
-- CLASSES (tenant-safe, indexes, triggers)
-- =========================================================
BEGIN;

-- Ext untuk gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1) TABLE
CREATE TABLE IF NOT EXISTS classes (
  class_id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id             UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_name                  VARCHAR(120) NOT NULL,
  class_slug                  VARCHAR(160) NOT NULL,
  class_code                  VARCHAR(40),

  class_description           TEXT,
  class_level                 TEXT,
  class_image_url             TEXT,

  class_trash_url             TEXT,
  class_delete_pending_until  TIMESTAMPTZ,

  class_mode                  VARCHAR(20),
  class_is_active             BOOLEAN NOT NULL DEFAULT TRUE,

  class_created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_deleted_at            TIMESTAMPTZ
);

-- guard kolom lama
ALTER TABLE classes
  ADD COLUMN IF NOT EXISTS class_code VARCHAR(40),
  ADD COLUMN IF NOT EXISTS class_trash_url TEXT,
  ADD COLUMN IF NOT EXISTS class_delete_pending_until TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS class_mode VARCHAR(20);

DO $$
BEGIN
  BEGIN ALTER TABLE classes ALTER COLUMN class_mode DROP NOT NULL; EXCEPTION WHEN others THEN NULL; END;
  BEGIN ALTER TABLE classes ALTER COLUMN class_mode DROP DEFAULT;   EXCEPTION WHEN others THEN NULL; END;
END$$;

-- bersihkan unique lama
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_slug_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_slug_key;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_code_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_code_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_code_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_code_key;
  END IF;
END$$;

-- unique komposit
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid') THEN
    ALTER TABLE classes
      ADD CONSTRAINT uq_classes_id_masjid UNIQUE (class_id, class_masjid_id);
  END IF;
END$$;

-- index
CREATE INDEX IF NOT EXISTS idx_classes_masjid         ON classes (class_masjid_id);
CREATE INDEX IF NOT EXISTS idx_classes_active         ON classes (class_is_active);
CREATE INDEX IF NOT EXISTS idx_classes_created_at     ON classes (class_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_classes_slug           ON classes (class_slug);
CREATE INDEX IF NOT EXISTS idx_classes_code           ON classes (class_code);
CREATE INDEX IF NOT EXISTS idx_classes_mode_lower     ON classes (LOWER(class_mode));

CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_active
  ON classes (class_masjid_id, LOWER(class_slug))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_code_per_masjid_active
  ON classes (class_masjid_id, LOWER(class_code))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL
    AND class_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_classes_visible
  ON classes (class_masjid_id, class_is_active, class_created_at DESC)
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_pending_until
  ON classes (class_delete_pending_until)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_masjid_slug_visible
  ON classes (class_masjid_id, LOWER(class_slug))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_masjid_code_visible
  ON classes (class_masjid_id, LOWER(class_code))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL
    AND class_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_classes_masjid_mode_visible
  ON classes (class_masjid_id, class_mode, class_created_at DESC)
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

-- trigger
CREATE OR REPLACE FUNCTION fn_classes_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_updated_at := NOW();
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

COMMIT;

-- =========================================================
-- CLASS_SECTIONS
-- =========================================================
BEGIN;

CREATE TABLE IF NOT EXISTS class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_sections_class_id UUID NOT NULL
    REFERENCES classes(class_id) ON DELETE CASCADE,

  class_sections_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_sections_slug VARCHAR(160) NOT NULL,
  class_sections_teacher_id UUID,

  class_sections_name VARCHAR(100) NOT NULL,
  class_sections_code VARCHAR(50),
  class_sections_capacity INT
    CHECK (class_sections_capacity IS NULL OR class_sections_capacity >= 0),
  class_sections_schedule JSONB,

  class_sections_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_sections_deleted_at TIMESTAMPTZ
);

-- FK ke masjid_teachers, deteksi kolom PK
DO $$
DECLARE
  fk_name text;
  target_col text;
BEGIN
  -- cek FK lama
  SELECT tc.constraint_name
    INTO fk_name
  FROM information_schema.table_constraints tc
  JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
   AND tc.table_name = kcu.table_name
  WHERE tc.table_name = 'class_sections'
    AND tc.constraint_type = 'FOREIGN KEY'
    AND kcu.column_name = 'class_sections_teacher_id'
  LIMIT 1;

  -- deteksi nama kolom PK di masjid_teachers
  SELECT column_name
    INTO target_col
  FROM information_schema.columns
  WHERE table_name = 'masjid_teachers'
    AND column_name IN ('masjid_teachers_id','masjid_teacher_id','id')
  ORDER BY CASE column_name
             WHEN 'masjid_teachers_id' THEN 1
             WHEN 'masjid_teacher_id' THEN 2
             WHEN 'id' THEN 3
           END
  LIMIT 1;

  IF target_col IS NULL THEN
    RAISE EXCEPTION 'Tidak menemukan kolom PK di masjid_teachers';
  END IF;

  -- jika FK lama salah arah, drop
  IF fk_name IS NOT NULL THEN
    PERFORM 1
    FROM information_schema.referential_constraints rc
    JOIN information_schema.constraint_column_usage ccu
      ON rc.unique_constraint_name = ccu.constraint_name
    WHERE rc.constraint_name = fk_name
      AND ccu.table_name = 'masjid_teachers'
      AND ccu.column_name = target_col;

    IF NOT FOUND THEN
      EXECUTE format('ALTER TABLE class_sections DROP CONSTRAINT %I', fk_name);
      fk_name := NULL;
    END IF;
  END IF;

  -- buat FK benar kalau belum ada
  IF fk_name IS NULL THEN
    EXECUTE format($sql$
      ALTER TABLE class_sections
      ADD CONSTRAINT fk_class_sections_teacher
      FOREIGN KEY (class_sections_teacher_id)
      REFERENCES masjid_teachers(%I)
      ON DELETE SET NULL
    $sql$, target_col);
  END IF;
END$$;

-- unique + index
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name
  ON class_sections (class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_slug_per_masjid_active
  ON class_sections (class_sections_masjid_id, lower(class_sections_slug))
  WHERE class_sections_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sections_id_masjid
  ON class_sections (class_sections_id, class_sections_masjid_id);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_masjid') THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_masjid
      UNIQUE USING INDEX uq_class_sections_id_masjid;
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_sections_class       ON class_sections(class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_sections_active      ON class_sections(class_sections_is_active);
CREATE INDEX IF NOT EXISTS idx_sections_masjid      ON class_sections(class_sections_masjid_id);
CREATE INDEX IF NOT EXISTS idx_sections_created_at  ON class_sections(class_sections_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sections_slug        ON class_sections(class_sections_slug);
CREATE INDEX IF NOT EXISTS idx_sections_teacher     ON class_sections(class_sections_teacher_id);

-- trigger
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

COMMIT;

-- =========================================================
-- CLASS_PRICING_OPTIONS
-- =========================================================
BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_price_type') THEN
    CREATE TYPE class_price_type AS ENUM ('ONE_TIME','RECURRING');
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS class_pricing_options (
  class_pricing_options_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_pricing_options_class_id          UUID NOT NULL REFERENCES classes(class_id) ON DELETE CASCADE,

  class_pricing_options_label             VARCHAR(80) NOT NULL,
  class_pricing_options_price_type        class_price_type NOT NULL,
  class_pricing_options_amount_idr        INT NOT NULL CHECK (class_pricing_options_amount_idr >= 0),
  class_pricing_options_recurrence_months INT,

  class_pricing_options_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_pricing_options_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_pricing_options_deleted_at        TIMESTAMPTZ
);

-- index
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_class_id
  ON class_pricing_options (class_pricing_options_class_id);
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_created_at
  ON class_pricing_options (class_pricing_options_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_class_type_created_at
  ON class_pricing_options (
    class_pricing_options_class_id,
    class_pricing_options_price_type,
    class_pricing_options_created_at DESC
  )
  WHERE class_pricing_options_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_label_per_class
  ON class_pricing_options (class_pricing_options_class_id, lower(class_pricing_options_label))
  WHERE class_pricing_options_deleted_at IS NULL;

-- trigger
CREATE OR REPLACE FUNCTION fn_class_pricing_options_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_pricing_options_updated_at := now();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_pricing_options_touch_updated_at') THEN
    DROP TRIGGER trg_class_pricing_options_touch_updated_at ON class_pricing_options;
  END IF;

  CREATE TRIGGER trg_class_pricing_options_touch_updated_at
    BEFORE UPDATE ON class_pricing_options
    FOR EACH ROW
    EXECUTE FUNCTION fn_class_pricing_options_touch_updated_at();
END$$;

-- views
DROP VIEW IF EXISTS v_cpo_latest_per_type;
CREATE OR REPLACE VIEW v_class_pricing_options_latest_per_type AS
SELECT DISTINCT ON (class_pricing_options_class_id, class_pricing_options_price_type) *
FROM class_pricing_options
WHERE class_pricing_options_deleted_at IS NULL
ORDER BY class_pricing_options_class_id,
         class_pricing_options_price_type,
         class_pricing_options_created_at DESC,
         class_pricing_options_id DESC;

CREATE OR REPLACE VIEW v_class_pricing_options_active AS
SELECT *
FROM class_pricing_options
WHERE class_pricing_options_deleted_at IS NULL;

COMMIT;
