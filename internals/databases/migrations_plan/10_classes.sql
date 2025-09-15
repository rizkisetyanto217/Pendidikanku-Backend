BEGIN;

-- =========================================================
-- EXTENSIONS (safe repeat)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram ops (ILIKE/FTS)
CREATE EXTENSION IF NOT EXISTS btree_gin; -- optional for INCLUDE combos

-- =========================================================
-- ENUMS (guard via DO-block)
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

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_visibility_enum') THEN
    CREATE TYPE class_visibility_enum AS ENUM ('public','unlisted','private');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_category_enum') THEN
    CREATE TYPE class_category_enum AS ENUM ('umum','tahsin','tahfidz','fiqih','akhlak','sejarah','lainnya');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_parents
-- =========================================================
CREATE TABLE IF NOT EXISTS class_parents (
  class_parent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_parent_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Identitas & konten
  class_parent_slug           VARCHAR(160),
  class_parent_name           VARCHAR(120) NOT NULL,
  class_parent_code           VARCHAR(40),
  class_parent_description    TEXT,
  class_parent_icon_url       TEXT,
  class_parent_banner_url     TEXT,

  -- Hierarki & urutan
  class_parent_level          SMALLINT,
  class_parent_display_order  SMALLINT,

  -- Publishing & visibility
  class_parent_is_active      BOOLEAN     NOT NULL DEFAULT TRUE,

  -- Ringkasan/cache & audit-by-user
  class_parent_total_classes   INT        NOT NULL DEFAULT 0,

  class_parent_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_parent_deleted_at TIMESTAMPTZ,

  -- Tenant-safe pair
  UNIQUE (class_parent_id, class_parent_masjid_id),

  -- Guards
  CONSTRAINT ck_class_parents_level_range
    CHECK (class_parent_level IS NULL OR class_parent_level BETWEEN 0 AND 100),
  CONSTRAINT ck_class_parents_display_order_range
    CHECK (class_parent_display_order IS NULL OR class_parent_display_order BETWEEN 0 AND 32767),
  CONSTRAINT ck_class_parents_publish_window
    CHECK (
      class_parent_publish_at   IS NULL OR
      class_parent_unpublish_at IS NULL OR
      class_parent_unpublish_at >= class_parent_publish_at
    ),
  CONSTRAINT ck_class_parent_slug_fmt
    CHECK (class_parent_slug IS NULL OR class_parent_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

-- -------------------------
-- INDEXES: class_parents
-- -------------------------
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parents_slug_per_masjid_active
  ON class_parents (class_parent_masjid_id, LOWER(class_parent_slug))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_delete_pending_until IS NULL
    AND class_parent_slug IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_parents_code_per_masjid_active
  ON class_parents (class_parent_masjid_id, LOWER(class_parent_code))
  WHERE class_parent_deleted_at IS NULL
    AND class_parent_delete_pending_until IS NULL
    AND class_parent_code IS NOT NULL;

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

CREATE INDEX IF NOT EXISTS idx_class_parents_display_order_alive
  ON class_parents (class_parent_display_order)
  WHERE class_parent_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_parents_total_classes_alive
  ON class_parents (class_parent_total_classes)
  WHERE class_parent_deleted_at IS NULL;

-- Trigram search helpers
CREATE INDEX IF NOT EXISTS gin_class_parents_name_trgm_alive
  ON class_parents USING GIN (LOWER(class_parent_name) gin_trgm_ops)
  WHERE class_parent_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_class_parents_desc_trgm_alive
  ON class_parents USING GIN (LOWER(class_parent_description) gin_trgm_ops)
  WHERE class_parent_deleted_at IS NULL;

-- =========================================================
-- TABLE: classes
-- =========================================================
CREATE TABLE IF NOT EXISTS classes (
  class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Relasi
  class_parent_id UUID NOT NULL,
  class_term_id   UUID,

  -- Identitas & konten
  class_slug           VARCHAR(160) NOT NULL,
  class_title          VARCHAR(150),
  class_code           VARCHAR(40),
  class_summary        VARCHAR(300),
  class_description    TEXT,
  class_thumbnail_url  TEXT,
  class_thumbnail_alt  VARCHAR(160),
  class_banner_url     TEXT,
  class_canonical_url  TEXT,

  -- Registrasi / Term
  class_is_open BOOLEAN NOT NULL DEFAULT TRUE,
  class_registration_opens_at  TIMESTAMPTZ,
  class_registration_closes_at TIMESTAMPTZ,

  -- Kuota
  class_quota_total INT CHECK (class_quota_total IS NULL OR class_quota_total >= 0),
  class_quota_taken INT NOT NULL DEFAULT 0 CHECK (class_quota_taken >= 0),

  -- Pricing
  class_registration_fee_idr BIGINT,
  class_tuition_fee_idr      BIGINT,
  class_billing_cycle        billing_cycle_enum NOT NULL DEFAULT 'monthly',
  class_provider_product_id  TEXT,
  class_provider_price_id    TEXT,

  -- Diskon sederhana
  class_discount_percent      SMALLINT,
  class_discount_flat_idr     BIGINT,
  class_discount_starts_at    TIMESTAMPTZ,
  class_discount_ends_at      TIMESTAMPTZ,


  class_status       class_status_enum     NOT NULL DEFAULT 'active',
  class_status_reason TEXT,
  class_completed_at TIMESTAMPTZ,

  -- Visibility & lifecycle
  class_is_public      BOOLEAN           NOT NULL DEFAULT TRUE, -- legacy flag
  class_display_order  SMALLINT,

  -- Tagging/kategori & referensi eksternal
  class_tags         TEXT[],
  class_category     VARCHAR(60),
  class_category_enum class_category_enum,

  -- Prasyarat/usia
  class_requirements    JSONB     NOT NULL DEFAULT '{}'::jsonb,

  -- Audit
  class_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_deleted_at TIMESTAMPTZ,

  -- tenant-safe pair
  UNIQUE (class_id, class_masjid_id),

  -- Guards
  CONSTRAINT ck_class_slug_fmt
    CHECK (class_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  CONSTRAINT ck_class_code_fmt
    CHECK (class_code IS NULL OR class_code ~ '^[A-Z0-9._-]+$'),

  CONSTRAINT ck_class_reg_window CHECK (
    class_registration_opens_at IS NULL
    OR class_registration_closes_at IS NULL
    OR class_registration_closes_at >= class_registration_opens_at
  ),
  CONSTRAINT ck_class_dates_order
    CHECK (class_start_date IS NULL OR class_end_date IS NULL OR class_end_date >= class_start_date),

  CONSTRAINT ck_class_quota_le_total
    CHECK (class_quota_total IS NULL OR class_quota_taken <= class_quota_total),

  CONSTRAINT ck_classes_pricing_nonneg CHECK (
    (class_registration_fee_idr IS NULL OR class_registration_fee_idr >= 0) AND
    (class_tuition_fee_idr      IS NULL OR class_tuition_fee_idr >= 0)
  ),
  CONSTRAINT ck_classes_discount_valid CHECK (
    (class_discount_percent IS NULL OR (class_discount_percent BETWEEN 0 AND 100)) AND
    (class_discount_flat_idr IS NULL OR class_discount_flat_idr >= 0) AND
    (class_discount_starts_at IS NULL OR class_discount_ends_at IS NULL OR class_discount_ends_at >= class_discount_starts_at)
  ),

  CONSTRAINT ck_classes_waitlist_nonneg CHECK (class_waitlist_count >= 0),
  CONSTRAINT ck_classes_duration_nonneg CHECK (class_duration_minutes IS NULL OR class_duration_minutes >= 0),
  CONSTRAINT ck_classes_meeting_day_range CHECK (class_default_meeting_day IS NULL OR (class_default_meeting_day BETWEEN 0 AND 6)),
  CONSTRAINT ck_classes_lat_lng_bounds CHECK (
    (class_location_lat IS NULL OR (class_location_lat BETWEEN -90  AND 90)) AND
    (class_location_lng IS NULL OR (class_location_lng BETWEEN -180 AND 180))
  ),
  CONSTRAINT ck_classes_age_range CHECK (class_min_age IS NULL OR class_max_age IS NULL OR class_max_age >= class_min_age),

  CONSTRAINT ck_classes_completed_closed_full
    CHECK (class_status <> 'completed' OR (class_is_open = FALSE AND class_completed_at IS NOT NULL)),

  CONSTRAINT ck_classes_publish_window
    CHECK (class_publish_at IS NULL OR class_unpublish_at IS NULL OR class_unpublish_at >= class_publish_at),

  CONSTRAINT ck_classes_visibility_consistency
    CHECK (
      class_visibility IS NULL OR
      (class_visibility = 'public'  AND class_is_public = TRUE) OR
      (class_visibility IN ('unlisted','private') AND class_is_public = FALSE)
    ),

  -- FKs (komposit)
  CONSTRAINT fk_classes_parent_same_masjid
    FOREIGN KEY (class_parent_id, class_masjid_id)
    REFERENCES class_parents (class_parent_id, class_parent_masjid_id)
    ON DELETE CASCADE,

  CONSTRAINT fk_classes_term_masjid_pair
    FOREIGN KEY (class_term_id, class_masjid_id)
    REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- -------------------------
-- INDEXES: classes
-- -------------------------
-- Unik per masjid
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_slug_per_masjid_active
  ON classes (class_masjid_id, LOWER(class_slug))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_code_per_masjid_active
  ON classes (class_masjid_id, LOWER(class_code))
  WHERE class_deleted_at IS NULL
    AND class_delete_pending_until IS NULL
    AND class_code IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_external_ref_per_masjid
  ON classes (class_masjid_id, LOWER(class_external_ref))
  WHERE class_deleted_at IS NULL
    AND class_external_ref IS NOT NULL;

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_classes_masjid      ON classes (class_masjid_id);
CREATE INDEX IF NOT EXISTS idx_classes_parent      ON classes (class_parent_id);
CREATE INDEX IF NOT EXISTS idx_classes_term        ON classes (class_term_id);

-- Filter status/visibility (alive only)
CREATE INDEX IF NOT EXISTS idx_classes_status_alive
  ON classes (class_status)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_visibility_alive
  ON classes (class_visibility)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_is_open_alive
  ON classes (class_is_open)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_is_public_alive
  ON classes (class_is_public)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_delivery_mode_alive
  ON classes (class_delivery_mode)
  WHERE class_deleted_at IS NULL;

-- Waktu, urutan & slug
CREATE INDEX IF NOT EXISTS idx_classes_created_at  ON classes (class_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_classes_slug        ON classes (class_slug);
CREATE INDEX IF NOT EXISTS idx_classes_display_order_alive
  ON classes (class_display_order)
  WHERE class_deleted_at IS NULL;

-- Window pendaftaran & kombinasi tenant/term
CREATE INDEX IF NOT EXISTS ix_classes_tenant_term_open_live
  ON classes (class_masjid_id, class_term_id, class_is_open)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_classes_reg_window_live
  ON classes (class_masjid_id, class_registration_opens_at, class_registration_closes_at)
  WHERE class_deleted_at IS NULL;

-- Publish window, kategori, tags, currency, row_version
CREATE INDEX IF NOT EXISTS idx_classes_publish_window_alive
  ON classes (class_publish_at, class_unpublish_at)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_category_alive
  ON classes (class_category)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_category_enum_alive
  ON classes (class_category_enum)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_tags_gin_alive
  ON classes USING GIN (class_tags)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_currency_alive
  ON classes (class_currency)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_row_version_alive
  ON classes (class_row_version)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_discount_window_alive
  ON classes (class_discount_starts_at, class_discount_ends_at)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_canonical_alive
  ON classes (class_canonical_url)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_meeting_provider_alive
  ON classes (class_meeting_platform, class_meeting_provider_event_id)
  WHERE class_deleted_at IS NULL;

-- Covering indexes (feed/catalog & admin)
CREATE INDEX IF NOT EXISTS ix_classes_feed
  ON classes (class_masjid_id, class_publish_at DESC)
  INCLUDE (class_slug, class_title, class_thumbnail_url)
  WHERE class_deleted_at IS NULL AND class_is_public = TRUE;

CREATE INDEX IF NOT EXISTS ix_classes_term_status
  ON classes (class_masjid_id, class_term_id, class_status, class_created_at DESC)
  WHERE class_deleted_at IS NULL;

-- Trigram search helpers
CREATE INDEX IF NOT EXISTS gin_classes_title_trgm_alive
  ON classes USING GIN (LOWER(class_title) gin_trgm_ops)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_classes_summary_trgm_alive
  ON classes USING GIN (LOWER(class_summary) gin_trgm_ops)
  WHERE class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_classes_notes_trgm_alive
  ON classes USING GIN (LOWER(class_notes) gin_trgm_ops)
  WHERE class_deleted_at IS NULL;

-- FTS index
CREATE INDEX IF NOT EXISTS gin_classes_search_tsv
  ON classes USING GIN (class_search_tsv)
  WHERE class_deleted_at IS NULL;

-- =========================================================
-- FUNCTIONS & TRIGGERS
-- =========================================================

-- Auto updated_at
CREATE OR REPLACE FUNCTION set_updated_at_class_parents()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.class_parent_updated_at := now(); RETURN NEW; END$$;

CREATE OR REPLACE FUNCTION set_updated_at_classes()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.class_updated_at := now(); RETURN NEW; END$$;

DROP TRIGGER IF EXISTS trg_class_parents_updated_at ON class_parents;
CREATE TRIGGER trg_class_parents_updated_at
BEFORE UPDATE ON class_parents
FOR EACH ROW EXECUTE FUNCTION set_updated_at_class_parents();

DROP TRIGGER IF EXISTS trg_classes_updated_at ON classes;
CREATE TRIGGER trg_classes_updated_at
BEFORE UPDATE ON classes
FOR EACH ROW EXECUTE FUNCTION set_updated_at_classes();

-- Bump counter class_parent_total_classes
CREATE OR REPLACE FUNCTION bump_class_parent_total_classes()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    UPDATE class_parents
      SET class_parent_total_classes = class_parent_total_classes + 1
    WHERE class_parent_id = NEW.class_parent_id
      AND class_parent_masjid_id = NEW.class_masjid_id;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    UPDATE class_parents
      SET class_parent_total_classes = GREATEST(0, class_parent_total_classes - 1)
    WHERE class_parent_id = OLD.class_parent_id
      AND class_parent_masjid_id = OLD.class_masjid_id;
    RETURN OLD;
  ELSIF TG_OP = 'UPDATE' AND (NEW.class_parent_id,NEW.class_masjid_id) IS DISTINCT FROM (OLD.class_parent_id,OLD.class_masjid_id) THEN
    UPDATE class_parents
      SET class_parent_total_classes = GREATEST(0, class_parent_total_classes - 1)
    WHERE class_parent_id = OLD.class_parent_id
      AND class_parent_masjid_id = OLD.class_masjid_id;
    UPDATE class_parents
      SET class_parent_total_classes = class_parent_total_classes + 1
    WHERE class_parent_id = NEW.class_parent_id
      AND class_parent_masjid_id = NEW.class_masjid_id;
    RETURN NEW;
  END IF;
  RETURN NEW;
END$$;

DROP TRIGGER IF EXISTS trg_classes_bump_parent ON classes;
CREATE TRIGGER trg_classes_bump_parent
AFTER INSERT OR DELETE OR UPDATE OF class_parent_id, class_masjid_id ON classes
FOR EACH ROW WHEN (
  (TG_OP = 'INSERT' AND NEW.class_deleted_at IS NULL) OR
  (TG_OP = 'DELETE' AND OLD.class_deleted_at IS NULL) OR
  (TG_OP = 'UPDATE' AND NEW.class_deleted_at IS NULL)
)
EXECUTE FUNCTION bump_class_parent_total_classes();

-- FTS tsvector maintenance
CREATE OR REPLACE FUNCTION classes_update_tsv()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.class_search_tsv :=
    setweight(to_tsvector('simple', coalesce(NEW.class_title,'')), 'A') ||
    setweight(to_tsvector('simple', coalesce(NEW.class_summary,'')), 'B') ||
    setweight(to_tsvector('simple', coalesce(NEW.class_description,'')), 'C');
  RETURN NEW;
END$$;

DROP TRIGGER IF EXISTS trg_classes_tsv ON classes;
CREATE TRIGGER trg_classes_tsv
BEFORE INSERT OR UPDATE OF class_title, class_summary, class_description
ON classes FOR EACH ROW EXECUTE FUNCTION classes_update_tsv();

-- Optimistic locking (row_version auto-increment)
CREATE OR REPLACE FUNCTION classes_bump_row_version()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.class_row_version := COALESCE(OLD.class_row_version,1) + 1;
  RETURN NEW;
END$$;

DROP TRIGGER IF EXISTS trg_classes_row_version ON classes;
CREATE TRIGGER trg_classes_row_version
BEFORE UPDATE ON classes
FOR EACH ROW EXECUTE FUNCTION classes_bump_row_version();

-- =========================================================
-- VIEW untuk publik (simplify query FE)
-- =========================================================
CREATE OR REPLACE VIEW v_classes_public_live AS
SELECT c.*
FROM classes c
WHERE c.class_deleted_at IS NULL
  AND c.class_is_public = TRUE
  AND (c.class_publish_at IS NULL OR c.class_publish_at <= now())
  AND (c.class_unpublish_at IS NULL OR c.class_unpublish_at >= now());

-- =========================================================
-- (OPSIONAL) RLS skeleton — aktifkan jika ingin hard multi-tenant di DB
-- =========================================================
-- ALTER TABLE class_parents ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE classes ENABLE ROW LEVEL SECURITY;
-- CREATE POLICY cls_parents_tenant ON class_parents
--   USING (class_parent_masjid_id::text = current_setting('app.mjid', true));
-- CREATE POLICY classes_tenant ON classes
--   USING (class_masjid_id::text = current_setting('app.mjid', true));
-- -- Di app, setelah resolve tenant:
-- --   SET app.mjid = '<masjid_uuid>';

COMMIT;

-- =========================================================
-- (OPSIONAL) FKs ke users — jalankan TERPISAH setelah tabel users siap
-- =========================================================
-- ALTER TABLE class_parents
--   ADD CONSTRAINT fk_class_parent_created_by_users
--   FOREIGN KEY (class_parent_created_by) REFERENCES users(id) ON DELETE SET NULL;
-- ALTER TABLE class_parents
--   ADD CONSTRAINT fk_class_parent_updated_by_users
--   FOREIGN KEY (class_parent_updated_by) REFERENCES users(id) ON DELETE SET NULL;
-- ALTER TABLE class_parents
--   ADD CONSTRAINT fk_class_parent_deleted_by_users
--   FOREIGN KEY (class_parent_deleted_by) REFERENCES users(id) ON DELETE SET NULL;

-- ALTER TABLE classes
--   ADD CONSTRAINT fk_classes_created_by_users
--   FOREIGN KEY (class_created_by) REFERENCES users(id) ON DELETE SET NULL;
-- ALTER TABLE classes
--   ADD CONSTRAINT fk_classes_updated_by_users
--   FOREIGN KEY (class_updated_by) REFERENCES users(id) ON DELETE SET NULL;
-- ALTER TABLE classes
--   ADD CONSTRAINT fk_classes_deleted_by_users
--   FOREIGN KEY (class_deleted_by) REFERENCES users(id) ON DELETE SET NULL;
-- ALTER TABLE classes
--   ADD CONSTRAINT fk_classes_main_teacher
--   FOREIGN KEY (class_main_teacher_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL;
