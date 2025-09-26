BEGIN;

-- =========================================================
-- EXTENSIONS
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram (untuk GIN ILIKE opsional)

-- =========================================================
-- ENUMS (guarded)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_cycle_enum') THEN
    CREATE TYPE billing_cycle_enum AS ENUM ('one_time','monthly','quarterly','semester','yearly');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('online','offline','hybrid');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_status_enum') THEN
    CREATE TYPE class_status_enum AS ENUM ('active','inactive','completed');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_parents
-- =========================================================
CREATE TABLE IF NOT EXISTS class_parents (
  class_parent_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_parent_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_parent_name   VARCHAR(120) NOT NULL,
  class_parent_code   VARCHAR(40),
  class_parent_slug   VARCHAR(160),

  class_parent_description   TEXT,
  class_parent_level         SMALLINT,  -- 0..100, opsional
  class_parent_is_active     BOOLEAN NOT NULL DEFAULT TRUE,
  class_parent_total_classes INT     NOT NULL DEFAULT 0,

  -- Prasyarat/usia (fleksibel)
  class_parent_requirements  JSONB  NOT NULL DEFAULT '{}'::jsonb,

  -- Single image (2-slot + retensi 30 hari)
  class_parent_image_url                   TEXT,
  class_parent_image_object_key            TEXT,
  class_parent_image_url_old               TEXT,
  class_parent_image_object_key_old        TEXT,
  class_parent_image_delete_pending_until  TIMESTAMPTZ,

  -- Audit
  class_parent_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_deleted_at TIMESTAMPTZ,

  -- tenant-safe pair
  UNIQUE (class_parent_id, class_parent_masjid_id),

  -- Guards
  CONSTRAINT ck_class_parents_level_range
    CHECK (class_parent_level IS NULL OR class_parent_level BETWEEN 0 AND 100)
);

-- Indexes (class_parents)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parents_code_per_masjid_alive
  ON class_parents (class_parent_masjid_id, LOWER(class_parent_code))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_code IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parents_slug_per_masjid_alive
  ON class_parents (class_parent_masjid_id, LOWER(class_parent_slug))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_class_parents_masjid
  ON class_parents (class_parent_masjid_id);

CREATE INDEX IF NOT EXISTS idx_class_parents_active_alive
  ON class_parents (class_parent_is_active)
  WHERE class_parent_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_parents_created_at
  ON class_parents (class_parent_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_class_parents_level_alive
  ON class_parents (class_parent_level)
  WHERE class_parent_deleted_at IS NULL;

-- Purge kandidat image lama
CREATE INDEX IF NOT EXISTS idx_class_parents_image_purge_due
  ON class_parents (class_parent_image_delete_pending_until)
  WHERE class_parent_image_object_key_old IS NOT NULL;



-- =========================================================
-- TABLE: classes
-- =========================================================
CREATE TABLE IF NOT EXISTS classes (
  class_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_parent_id UUID NOT NULL,

  class_slug      VARCHAR(160) NOT NULL,

  class_start_date DATE,
  class_end_date   DATE,

  -- Registrasi / Term
  class_term_id UUID,
  class_registration_opens_at  TIMESTAMPTZ,
  class_registration_closes_at TIMESTAMPTZ,
  CONSTRAINT ck_class_reg_window CHECK (
    class_registration_opens_at IS NULL
    OR class_registration_closes_at IS NULL
    OR class_registration_closes_at >= class_registration_opens_at
  ),

  -- Kuota
  class_quota_total INT CHECK (class_quota_total IS NULL OR class_quota_total >= 0),
  class_quota_taken INT NOT NULL DEFAULT 0 CHECK (class_quota_taken >= 0),
  CONSTRAINT ck_class_quota_le_total
    CHECK (class_quota_total IS NULL OR class_quota_taken <= class_quota_total),

  -- Pricing
  class_registration_fee_idr BIGINT,
  class_tuition_fee_idr      BIGINT,
  class_billing_cycle        billing_cycle_enum NOT NULL DEFAULT 'monthly',
  class_provider_product_id  TEXT,
  class_provider_price_id    TEXT,

  -- Catatan
  class_notes TEXT,

  -- Mode & Status
  class_delivery_mode class_delivery_mode_enum,
  class_status        class_status_enum NOT NULL DEFAULT 'active',
  class_completed_at  TIMESTAMPTZ,

  -- Single image (2-slot + retensi 30 hari)
  class_image_url                   TEXT,
  class_image_object_key            TEXT,
  class_image_url_old               TEXT,
  class_image_object_key_old        TEXT,
  class_image_delete_pending_until  TIMESTAMPTZ,

  -- Snapshot Class Parent
  class_code_parent_snapshot   VARCHAR(40),
  class_name_parent_snapshot   VARCHAR(80),
  class_slug_parent_snapshot   VARCHAR(160),
  class_level_parent_snapshot  SMALLINT,

  -- Snapshot Class Term
  class_academic_year_term_snapshot   VARCHAR(40),
  class_name_term_snapshot VARCHAR(100),
  class_slug_term_snapshot VARCHAR(160),
  class_angkatan_term_snapshot VARCHAR(40),

  -- Audit
  class_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_deleted_at TIMESTAMPTZ,

  -- tenant-safe pair
  UNIQUE (class_id, class_masjid_id),

  -- FKs komposit (parent & term pada tenant yang sama)
  CONSTRAINT fk_classes_parent_same_masjid
    FOREIGN KEY (class_parent_id, class_masjid_id)
    REFERENCES class_parents (class_parent_id, class_parent_masjid_id)
    ON DELETE CASCADE,

  CONSTRAINT fk_classes_term_masjid_pair
    FOREIGN KEY (class_term_id, class_masjid_id)
    REFERENCES academic_terms (academic_term_id, academic_term_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- Indexes (classes)
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_alive
  ON classes (class_masjid_id, LOWER(class_slug))
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_masjid ON classes (class_masjid_id);
CREATE INDEX IF NOT EXISTS idx_classes_parent ON classes (class_parent_id);

CREATE INDEX IF NOT EXISTS idx_classes_status_alive
  ON classes (class_status)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_created_at
  ON classes (class_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_classes_slug
  ON classes (class_slug);

CREATE INDEX IF NOT EXISTS idx_classes_delivery_mode_alive
  ON classes (class_delivery_mode)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_reg_window_alive
  ON classes (class_masjid_id, class_registration_opens_at, class_registration_closes_at)
  WHERE class_deleted_at IS NULL;

-- Trigram pada notes (opsional)
CREATE INDEX IF NOT EXISTS gin_classes_notes_trgm_alive
  ON classes USING GIN (LOWER(class_notes) gin_trgm_ops)
  WHERE class_deleted_at IS NULL;

-- Purge kandidat image lama
CREATE INDEX IF NOT EXISTS idx_classes_iage_purge_due
  ON classes (class_image_delete_pending_until)
  WHERE class_image_object_key_old IS NOT NULL;

COMMIT;
