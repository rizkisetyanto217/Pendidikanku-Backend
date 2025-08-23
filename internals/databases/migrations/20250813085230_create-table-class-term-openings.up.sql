-- 20250823_02_class_term_openings.up.sql

BEGIN;

-- Extensions yang diperlukan (notes trigram)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Function: touch updated_at
CREATE OR REPLACE FUNCTION fn_touch_class_term_openings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_term_openings_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- Table: class_term_openings
CREATE TABLE IF NOT EXISTS class_term_openings (
  class_term_openings_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_term_openings_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_term_openings_class_id  UUID NOT NULL,
  class_term_openings_term_id   UUID NOT NULL,

  class_term_openings_is_open BOOLEAN NOT NULL DEFAULT TRUE,

  class_term_openings_registration_opens_at  TIMESTAMP,
  class_term_openings_registration_closes_at TIMESTAMP,
  CONSTRAINT ck_cto_reg_window
    CHECK (
      class_term_openings_registration_opens_at IS NULL
      OR class_term_openings_registration_closes_at IS NULL
      OR class_term_openings_registration_closes_at >= class_term_openings_registration_opens_at
    ),

  class_term_openings_quota_total INT CHECK (class_term_openings_quota_total IS NULL OR class_term_openings_quota_total >= 0),
  class_term_openings_quota_taken INT NOT NULL DEFAULT 0 CHECK (class_term_openings_quota_taken >= 0),
  class_term_openings_fee_override_monthly_idr INT CHECK (class_term_openings_fee_override_monthly_idr IS NULL OR class_term_openings_fee_override_monthly_idr >= 0),

  class_term_openings_notes TEXT,

  class_term_openings_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  class_term_openings_updated_at TIMESTAMP,
  class_term_openings_deleted_at TIMESTAMP
);

-- Pastikan classes punya composite-unique (id, masjid_id) untuk FK komposit
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

-- Pastikan academic_terms punya composite-unique (id, masjid_id) untuk FK komposit
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid'
  ) THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_masjid
      UNIQUE (academic_terms_id, academic_terms_masjid_id);
  END IF;
END$$;

-- FKs (tenant-safe): class & term harus milik masjid yang sama
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_cto_class_masjid_pair'
  ) THEN
    ALTER TABLE class_term_openings
      ADD CONSTRAINT fk_cto_class_masjid_pair
      FOREIGN KEY (class_term_openings_class_id, class_term_openings_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_cto_term_masjid_pair'
  ) THEN
    ALTER TABLE class_term_openings
      ADD CONSTRAINT fk_cto_term_masjid_pair
      FOREIGN KEY (class_term_openings_term_id, class_term_openings_masjid_id)
      REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- Trigger: touch updated_at
DROP TRIGGER IF EXISTS trg_touch_class_term_openings ON class_term_openings;
CREATE TRIGGER trg_touch_class_term_openings
BEFORE UPDATE ON class_term_openings
FOR EACH ROW EXECUTE FUNCTION fn_touch_class_term_openings_updated_at();

-- Cegah kuota over-taken
CREATE OR REPLACE FUNCTION fn_cto_quota_nonnegative()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.class_term_openings_quota_total IS NOT NULL
     AND NEW.class_term_openings_quota_taken > NEW.class_term_openings_quota_total THEN
    RAISE EXCEPTION 'Quota exceeded: taken(%) > total(%)',
      NEW.class_term_openings_quota_taken, NEW.class_term_openings_quota_total;
  END IF;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cto_quota_nonnegative ON class_term_openings;
CREATE TRIGGER trg_cto_quota_nonnegative
BEFORE INSERT OR UPDATE ON class_term_openings
FOR EACH ROW EXECUTE FUNCTION fn_cto_quota_nonnegative();

-- Indexes
CREATE INDEX IF NOT EXISTS ix_cto_tenant_term_open_live
  ON class_term_openings (class_term_openings_masjid_id, class_term_openings_term_id, class_term_openings_is_open)
  WHERE class_term_openings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_cto_tenant_class_live
  ON class_term_openings (class_term_openings_masjid_id, class_term_openings_class_id)
  WHERE class_term_openings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_cto_reg_window_live
  ON class_term_openings (class_term_openings_masjid_id, class_term_openings_registration_opens_at, class_term_openings_registration_closes_at)
  WHERE class_term_openings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_cto_notes_trgm_live
  ON class_term_openings USING GIN (LOWER(class_term_openings_notes) gin_trgm_ops)
  WHERE class_term_openings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_cto_created_at_live
  ON class_term_openings (class_term_openings_masjid_id, class_term_openings_created_at DESC)
  WHERE class_term_openings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_cto_updated_at_live
  ON class_term_openings (class_term_openings_masjid_id, class_term_openings_updated_at DESC)
  WHERE class_term_openings_deleted_at IS NULL;

-- Functions: claim/release kuota
CREATE OR REPLACE FUNCTION class_term_openings_claim(p_opening_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
  v_now TIMESTAMP := CURRENT_TIMESTAMP;
  v_rows INT;
BEGIN
  UPDATE class_term_openings
     SET class_term_openings_quota_taken = class_term_openings_quota_taken + 1,
         class_term_openings_updated_at  = v_now
   WHERE class_term_openings_id = p_opening_id
     AND class_term_openings_is_open = TRUE
     AND class_term_openings_deleted_at IS NULL
     AND (
           class_term_openings_registration_opens_at IS NULL
           OR v_now >= class_term_openings_registration_opens_at
         )
     AND (
           class_term_openings_registration_closes_at IS NULL
           OR v_now <= class_term_openings_registration_closes_at
         )
     AND (
           class_term_openings_quota_total IS NULL
           OR class_term_openings_quota_taken < class_term_openings_quota_total
         );

  GET DIAGNOSTICS v_rows = ROW_COUNT;
  RETURN v_rows = 1;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION class_term_openings_release(p_opening_id UUID)
RETURNS VOID AS $$
DECLARE v_now TIMESTAMP := CURRENT_TIMESTAMP;
BEGIN
  UPDATE class_term_openings
     SET class_term_openings_quota_taken = GREATEST(class_term_openings_quota_taken - 1, 0),
         class_term_openings_updated_at  = v_now
   WHERE class_term_openings_id = p_opening_id
     AND class_term_openings_deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

COMMIT;
