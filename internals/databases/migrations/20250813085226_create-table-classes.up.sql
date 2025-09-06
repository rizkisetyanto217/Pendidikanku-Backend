BEGIN;

-- =========================================================
-- EXTENSIONS
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- ENUMS
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_cycle_enum') THEN
    CREATE TYPE billing_cycle_enum AS ENUM ('one_time','monthly','quarterly','semester','yearly');
  END IF;
END$$;

-- =========================================================
-- PARENT: class_parent
-- =========================================================
CREATE TABLE IF NOT EXISTS class_parent (
  class_parent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_parent_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_parent_name VARCHAR(120) NOT NULL,
  class_parent_slug VARCHAR(160) NOT NULL,
  class_parent_code VARCHAR(40),

  class_parent_description TEXT,
  class_parent_level TEXT,
  class_parent_image_url TEXT,

  class_parent_mode VARCHAR(20),
  class_parent_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_parent_trash_url TEXT,
  class_parent_delete_pending_until TIMESTAMPTZ,

  class_parent_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_deleted_at TIMESTAMPTZ
);

-- Indexes & Uniques
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parent_slug_per_masjid_active
  ON class_parent (class_parent_masjid_id, lower(class_parent_slug))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_delete_pending_until IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parent_code_per_masjid_active
  ON class_parent (class_parent_masjid_id, lower(class_parent_code))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_delete_pending_until IS NULL
    AND class_parent_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_class_parent_masjid     ON class_parent (class_parent_masjid_id);
CREATE INDEX IF NOT EXISTS idx_class_parent_active     ON class_parent (class_parent_is_active);
CREATE INDEX IF NOT EXISTS idx_class_parent_created_at ON class_parent (class_parent_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_class_parent_slug       ON class_parent (class_parent_slug);
CREATE INDEX IF NOT EXISTS idx_class_parent_code       ON class_parent (class_parent_code);
CREATE INDEX IF NOT EXISTS idx_class_parent_mode_lower ON class_parent (lower(class_parent_mode));

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_parent_id_masjid') THEN
    ALTER TABLE class_parent
      ADD CONSTRAINT uq_class_parent_id_masjid UNIQUE (class_parent_id, class_parent_masjid_id);
  END IF;
END$$;

-- =========================================================
-- CHILD: classes
-- =========================================================
CREATE TABLE IF NOT EXISTS classes (
  class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_parent_id UUID NOT NULL,

  class_name VARCHAR(140) NOT NULL,
  class_slug VARCHAR(160) NOT NULL,
  class_code VARCHAR(50),

  class_start_date DATE,
  class_end_date DATE,

  -- Registrasi / Term
  class_term_id UUID,
  class_is_open BOOLEAN NOT NULL DEFAULT TRUE,
  class_registration_opens_at  TIMESTAMPTZ,
  class_registration_closes_at TIMESTAMPTZ,
  CONSTRAINT ck_class_reg_window
  CHECK (
    class_registration_opens_at IS NULL
    OR class_registration_closes_at IS NULL
    OR class_registration_closes_at >= class_registration_opens_at
  ),

  class_quota_total INT CHECK (class_quota_total IS NULL OR class_quota_total >= 0),
  class_quota_taken INT NOT NULL DEFAULT 0 CHECK (class_quota_taken >= 0),

  -- Pricing (ringkas, 1:1)
  class_registration_fee_idr BIGINT,
  class_tuition_fee_idr      BIGINT,
  class_billing_cycle        billing_cycle_enum NOT NULL DEFAULT 'monthly',
  class_provider_product_id  TEXT,
  class_provider_price_id    TEXT,

  class_notes TEXT,

  -- Status cukup pakai is_active
  class_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_trash_url TEXT,
  class_delete_pending_until TIMESTAMPTZ,

  class_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_deleted_at TIMESTAMPTZ
);

-- Tenant-safe FK ke class_parent
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_classes_parent_same_masjid') THEN
    ALTER TABLE classes
      ADD CONSTRAINT fk_classes_parent_same_masjid
      FOREIGN KEY (class_parent_id, class_masjid_id)
      REFERENCES class_parent (class_parent_id, class_parent_masjid_id)
      ON DELETE CASCADE;
  END IF;
END$$;

-- Composite-unique di classes
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid') THEN
    ALTER TABLE classes
      ADD CONSTRAINT uq_classes_id_masjid UNIQUE (class_id, class_masjid_id);
  END IF;
END$$;

-- Composite-unique di academic_terms (untuk FK komposit)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid') THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_masjid UNIQUE (academic_terms_id, academic_terms_masjid_id);
  END IF;
END$$;

-- FK tenant-safe: term
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_classes_term_masjid_pair') THEN
    ALTER TABLE classes
      ADD CONSTRAINT fk_classes_term_masjid_pair
      FOREIGN KEY (class_term_id, class_masjid_id)
      REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
      ON UPDATE CASCADE
      ON DELETE RESTRICT;
  END IF;
END$$;

-- Indexes & Uniques
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_active
  ON classes (class_masjid_id, lower(class_slug))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_code_per_masjid_active
  ON classes (class_masjid_id, lower(class_code))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL
    AND class_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_classes_masjid      ON classes (class_masjid_id);
CREATE INDEX IF NOT EXISTS idx_classes_parent      ON classes (class_parent_id);
CREATE INDEX IF NOT EXISTS idx_classes_active      ON classes (class_is_active);
CREATE INDEX IF NOT EXISTS idx_classes_created_at  ON classes (class_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_classes_slug        ON classes (class_slug);
CREATE INDEX IF NOT EXISTS idx_classes_code        ON classes (class_code);

-- Index tambahan registrasi/kuota
CREATE INDEX IF NOT EXISTS ix_classes_tenant_term_open_live
  ON classes (class_masjid_id, class_term_id, class_is_open)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_classes_reg_window_live
  ON classes (class_masjid_id, class_registration_opens_at, class_registration_closes_at)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_classes_notes_trgm_live
  ON classes USING GIN (LOWER(class_notes) gin_trgm_ops)
  WHERE class_deleted_at IS NULL;

-- Guard harga non-negatif
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_classes_pricing_nonneg') THEN
    ALTER TABLE classes
      ADD CONSTRAINT ck_classes_pricing_nonneg
      CHECK (
        (class_registration_fee_idr IS NULL OR class_registration_fee_idr >= 0) AND
        (class_tuition_fee_idr      IS NULL OR class_tuition_fee_idr >= 0)
      );
  END IF;
END$$;

-- =========================================================
-- FUNCTIONS & TRIGGERS
-- =========================================================

-- Quota guard
CREATE OR REPLACE FUNCTION fn_classes_quota_nonnegative()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.class_quota_total IS NOT NULL
     AND NEW.class_quota_taken > NEW.class_quota_total THEN
    RAISE EXCEPTION 'Quota exceeded: taken(%) > total(%)',
      NEW.class_quota_taken, NEW.class_quota_total;
  END IF;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_classes_quota_nonnegative ON classes;
CREATE TRIGGER trg_classes_quota_nonnegative
BEFORE INSERT OR UPDATE ON classes
FOR EACH ROW EXECUTE FUNCTION fn_classes_quota_nonnegative();

-- Claim quota
CREATE OR REPLACE FUNCTION class_claim(p_class_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
  v_now TIMESTAMPTZ := now();
  v_rows INT;
BEGIN
  UPDATE classes
     SET class_quota_taken = class_quota_taken + 1,
         class_updated_at  = v_now
   WHERE class_id = p_class_id
     AND class_deleted_at IS NULL
     AND class_is_open = TRUE
     AND (class_registration_opens_at IS NULL OR v_now >= class_registration_opens_at)
     AND (class_registration_closes_at IS NULL OR v_now <= class_registration_closes_at)
     AND (class_quota_total IS NULL OR class_quota_taken < class_quota_total);

  GET DIAGNOSTICS v_rows = ROW_COUNT;
  RETURN v_rows = 1;
END;
$$ LANGUAGE plpgsql;

-- Release quota
CREATE OR REPLACE FUNCTION class_release(p_class_id UUID)
RETURNS VOID AS $$
DECLARE
  v_now TIMESTAMPTZ := now();
BEGIN
  UPDATE classes
     SET class_quota_taken = GREATEST(class_quota_taken - 1, 0),
         class_updated_at  = v_now
   WHERE class_id = p_class_id
     AND class_deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

COMMIT;
