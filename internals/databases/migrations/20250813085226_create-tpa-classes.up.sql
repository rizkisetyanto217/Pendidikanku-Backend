-- =========================================================
-- Supabase/Postgres setup (jika belum)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- untuk gen_random_uuid()

-- =========================================================
-- TABEL: classes  (fresh install friendly)
-- =========================================================
CREATE TABLE IF NOT EXISTS classes (
  class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_name VARCHAR(120) NOT NULL,
  class_slug VARCHAR(160) NOT NULL,

  class_description TEXT,
  class_level TEXT,
  class_image_url TEXT,

  class_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_deleted_at TIMESTAMPTZ
);

-- UNIQUE (class_id, class_masjid_id)
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

-- Hapus UNIQUE global class_slug jika ada
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'classes_class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT classes_class_slug_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_slug_key') THEN
    ALTER TABLE classes DROP CONSTRAINT class_slug_key;
  END IF;
END$$;

-- INDEXES classes
CREATE INDEX IF NOT EXISTS idx_classes_masjid      ON classes(class_masjid_id);
CREATE INDEX IF NOT EXISTS idx_classes_active      ON classes(class_is_active);
CREATE INDEX IF NOT EXISTS idx_classes_created_at  ON classes(class_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_classes_slug        ON classes(class_slug);

-- UNIQUE slug per masjid (soft-delete aware, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_active
  ON classes (class_masjid_id, lower(class_slug))
  WHERE class_deleted_at IS NULL;

-- TRIGGER updated_at (classes)
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
-- CLASS_SECTIONS (refactor, fully idempotent)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_sections_class_id UUID NOT NULL
    REFERENCES classes(class_id) ON DELETE CASCADE,

  class_sections_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_sections_slug VARCHAR(160) NOT NULL,

  class_sections_teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,

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

-- Bersihkan UNIQUE slug lawas jika ada
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_sections_class_sections_slug_key') THEN
    ALTER TABLE class_sections DROP CONSTRAINT class_sections_class_sections_slug_key;
  ELSIF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_sections_slug_key') THEN
    ALTER TABLE class_sections DROP CONSTRAINT class_sections_slug_key;
  END IF;
END$$;

-- Backfill masjid_id bila ada NULL (skema lama), lalu coba NOT NULL
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
      NULL;
    END;
  END IF;
END$$;

-- UNIQUE nama section per class (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name
  ON class_sections (class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

-- UNIQUE slug per masjid (soft-delete aware, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_slug_per_masjid_active
  ON class_sections (class_sections_masjid_id, lower(class_sections_slug))
  WHERE class_sections_deleted_at IS NULL;

-- Composite UNIQUE (id, masjid_id) via index + attach constraint
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sections_id_masjid
  ON class_sections (class_sections_id, class_sections_masjid_id);

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

-- Index umum
CREATE INDEX IF NOT EXISTS idx_sections_class       ON class_sections(class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_sections_active      ON class_sections(class_sections_is_active);
CREATE INDEX IF NOT EXISTS idx_sections_masjid      ON class_sections(class_sections_masjid_id);
CREATE INDEX IF NOT EXISTS idx_sections_created_at  ON class_sections(class_sections_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sections_slug        ON class_sections(class_sections_slug);
CREATE INDEX IF NOT EXISTS idx_sections_teacher     ON class_sections(class_sections_teacher_id);

-- TRIGGER updated_at (class_sections)
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



-- =========================================================
-- ENUM: class_price_type
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_price_type') THEN
    CREATE TYPE class_price_type AS ENUM ('ONE_TIME','RECURRING');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_pricing_options (kolom eksplisit)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_pricing_options (
  class_pricing_options_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_pricing_options_class_id          UUID NOT NULL REFERENCES classes(class_id) ON DELETE CASCADE,

  class_pricing_options_label             VARCHAR(80) NOT NULL,
  class_pricing_options_price_type        class_price_type NOT NULL,
  class_pricing_options_amount_idr        INT NOT NULL CHECK (class_pricing_options_amount_idr >= 0),

  -- ONE_TIME  -> NULL
  -- RECURRING -> 1,3,6,12
  class_pricing_options_recurrence_months INT,

  class_pricing_options_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_pricing_options_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_pricing_options_deleted_at        TIMESTAMPTZ
);

-- =========================================================
-- RENAME kolom lama cpo_* -> nama panjang (idempotent & aman)
-- =========================================================
DO $$
BEGIN
  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_id';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_id'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_id TO class_pricing_options_id;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_class_id';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_class_id'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_class_id TO class_pricing_options_class_id;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_label';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_label'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_label TO class_pricing_options_label;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_price_type';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_price_type'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_price_type TO class_pricing_options_price_type;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_amount_idr';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_amount_idr'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_amount_idr TO class_pricing_options_amount_idr;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_recurrence_months';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_recurrence_months'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_recurrence_months TO class_pricing_options_recurrence_months;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_created_at';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_created_at'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_created_at TO class_pricing_options_created_at;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_updated_at';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_updated_at'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_updated_at TO class_pricing_options_updated_at;
  END IF;

  PERFORM 1 FROM information_schema.columns
   WHERE table_name='class_pricing_options' AND column_name='cpo_deleted_at';
  IF FOUND AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_deleted_at'
  ) THEN
    ALTER TABLE class_pricing_options RENAME COLUMN cpo_deleted_at TO class_pricing_options_deleted_at;
  END IF;
END$$;

-- =========================================================
-- Bersihkan kolom lama yang sudah tidak dipakai (currency, is_default)
-- =========================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='class_pricing_options' AND column_name='cpo_currency') THEN
    ALTER TABLE class_pricing_options DROP COLUMN cpo_currency;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_currency') THEN
    ALTER TABLE class_pricing_options DROP COLUMN class_pricing_options_currency;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='class_pricing_options' AND column_name='cpo_is_default') THEN
    ALTER TABLE class_pricing_options DROP COLUMN cpo_is_default;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='class_pricing_options' AND column_name='class_pricing_options_is_default') THEN
    ALTER TABLE class_pricing_options DROP COLUMN class_pricing_options_is_default;
  END IF;
END$$;

-- =========================================================
-- CHECK constraint (kombinasi price_type <-> recurrence)
-- =========================================================
DO $$
DECLARE
  has_old bool;
  has_new bool;
BEGIN
  SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='ck_cpo_combo')
    INTO has_old;
  SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='ck_class_pricing_options_combo')
    INTO has_new;

  IF has_old AND NOT has_new THEN
    ALTER TABLE class_pricing_options
      RENAME CONSTRAINT ck_cpo_combo TO ck_class_pricing_options_combo;
  ELSIF NOT has_old AND NOT has_new THEN
    ALTER TABLE class_pricing_options
      ADD CONSTRAINT ck_class_pricing_options_combo CHECK (
        (class_pricing_options_price_type = 'ONE_TIME'  AND class_pricing_options_recurrence_months IS NULL)
        OR
        (class_pricing_options_price_type = 'RECURRING' AND class_pricing_options_recurrence_months IN (1,3,6,12))
      );
  END IF;
END$$;

-- =========================================================
-- INDEXES (tanpa UNIQUE di label — duplikasi diperbolehkan)
-- =========================================================
-- Bersihkan index lama (unik/nama pendek) bila ada
DROP INDEX IF EXISTS uq_class_pricing_options_label_per_class;
DROP INDEX IF EXISTS uq_cpo_label_per_class;

-- Index untuk pencarian umum
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_class_id
  ON class_pricing_options (class_pricing_options_class_id);

CREATE INDEX IF NOT EXISTS idx_class_pricing_options_created_at
  ON class_pricing_options (class_pricing_options_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_class_pricing_options_class_type_created_at
  ON class_pricing_options (class_pricing_options_class_id,
                            class_pricing_options_price_type,
                            class_pricing_options_created_at DESC)
  WHERE class_pricing_options_deleted_at IS NULL;

-- (Opsional) index non-unik supaya query by (class,label) tetap cepat
CREATE INDEX IF NOT EXISTS idx_class_pricing_options_label_per_class
  ON class_pricing_options (class_pricing_options_class_id, lower(class_pricing_options_label))
  WHERE class_pricing_options_deleted_at IS NULL;

-- =========================================================
-- TRIGGER updated_at (drop legacy, buat baru)
-- =========================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cpo_touch_updated_at') THEN
    DROP TRIGGER trg_cpo_touch_updated_at ON class_pricing_options;
  END IF;
END$$;

DROP FUNCTION IF EXISTS fn_cpo_touch_updated_at();

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

-- =========================================================
-- VIEWs (pakai nama & kolom baru)
-- =========================================================
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