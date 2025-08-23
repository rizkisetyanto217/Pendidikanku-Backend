BEGIN;

-- =========================================================
-- Extensions (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- range/GiST helpers

/* ===================================================================
   A. ACADEMIC TERMS
   =================================================================== */

-- touch updated_at helper
CREATE OR REPLACE FUNCTION fn_touch_academic_terms_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.academic_terms_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- table
CREATE TABLE IF NOT EXISTS academic_terms (
  academic_terms_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_terms_masjid_id     UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  academic_terms_academic_year TEXT NOT NULL,  -- contoh: '2026/2027'
  academic_terms_name          TEXT NOT NULL,  -- 'Ganjil' | 'Genap' | 'Pendek' | dst.

  academic_terms_start_date    TIMESTAMP NOT NULL,
  academic_terms_end_date      TIMESTAMP NOT NULL,
  academic_terms_is_active     BOOLEAN   NOT NULL DEFAULT TRUE,

  -- half-open range [start, end)
  academic_terms_period        DATERANGE GENERATED ALWAYS AS
    (daterange(academic_terms_start_date::date, academic_terms_end_date::date, '[)')) STORED,

  academic_terms_created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  academic_terms_updated_at    TIMESTAMP          DEFAULT CURRENT_TIMESTAMP,
  academic_terms_deleted_at    TIMESTAMP,

  CHECK (academic_terms_end_date >= academic_terms_start_date)
);

-- trigger
DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;
CREATE TRIGGER trg_touch_academic_terms
BEFORE UPDATE ON academic_terms
FOR EACH ROW EXECUTE FUNCTION fn_touch_academic_terms_updated_at();

-- cleanup legacy constraints / indexes (if any)
DROP INDEX IF EXISTS uq_academic_terms_tenant_year_name_live;
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_academic_terms_tenant_year_name_live'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT uq_academic_terms_tenant_year_name_live;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'ex_academic_terms_no_overlap_per_tenant'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT ex_academic_terms_no_overlap_per_tenant;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_academic_terms_one_active_per_tenant'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    ALTER TABLE academic_terms
      DROP CONSTRAINT uq_academic_terms_one_active_per_tenant;
  END IF;
END$$;
DROP INDEX IF EXISTS uq_academic_terms_one_active_per_tenant;

-- indexes (non-unique)
CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_dates
  ON academic_terms (academic_terms_masjid_id, academic_terms_start_date, academic_terms_end_date)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_period_gist
  ON academic_terms USING GIST (academic_terms_period)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
  ON academic_terms (academic_terms_masjid_id)
  WHERE academic_terms_is_active = TRUE
    AND academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_name_trgm
  ON academic_terms USING GIN (lower(academic_terms_name) gin_trgm_ops)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_year
  ON academic_terms (academic_terms_masjid_id, academic_terms_academic_year)
  WHERE academic_terms_deleted_at IS NULL;

DROP INDEX IF EXISTS ix_academic_terms_year_trgm;
CREATE INDEX IF NOT EXISTS ix_academic_terms_year_trgm_lower
  ON academic_terms USING GIN (lower(academic_terms_academic_year) gin_trgm_ops)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_created_at
  ON academic_terms (academic_terms_masjid_id, academic_terms_created_at)
  WHERE academic_terms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_updated_at
  ON academic_terms (academic_terms_masjid_id, academic_terms_updated_at)
  WHERE academic_terms_deleted_at IS NULL;

-- composite-unique untuk FK komposit (tenant-safe)
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



/* ===================================================================
   B. CLASS TERM OPENINGS (pembukaan program per term)
   =================================================================== */

-- helper updated_at
CREATE OR REPLACE FUNCTION fn_touch_class_term_openings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_term_openings_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- table
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

-- pastikan classes punya composite-unique (id, masjid_id) untuk FK komposit
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

-- FK komposit (tenant-safe): class & term harus milik masjid yang sama
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

-- trigger updated_at
DROP TRIGGER IF EXISTS trg_touch_class_term_openings ON class_term_openings;
CREATE TRIGGER trg_touch_class_term_openings
BEFORE UPDATE ON class_term_openings
FOR EACH ROW EXECUTE FUNCTION fn_touch_class_term_openings_updated_at();

-- cegah kuota over-taken
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

-- indexes
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

-- fungsi klaim / release kuota (atomic update)
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

/* ===================================================================
   C. USER_CLASSES: tambah relasi ke term (opsional, rekomendasi)
   =================================================================== */

-- add column if missing
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='user_classes' AND column_name='user_classes_term_id'
  ) THEN
    ALTER TABLE user_classes
      ADD COLUMN user_classes_term_id UUID;
  END IF;
END$$;

-- FK komposit (tenant-safe): term milik masjid yang sama
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_term_masjid_pair'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_term_masjid_pair
      FOREIGN KEY (user_classes_term_id, user_classes_masjid_id)
      REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- indexes
CREATE INDEX IF NOT EXISTS ix_uc_masjid_term_active
  ON user_classes (user_classes_masjid_id, user_classes_term_id, user_classes_status)
  WHERE user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS ix_uc_term
  ON user_classes (user_classes_term_id);

COMMIT;
