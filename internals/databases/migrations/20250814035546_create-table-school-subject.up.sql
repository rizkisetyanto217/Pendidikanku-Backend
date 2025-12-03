-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index (ILIKE search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- optional


-- =========================================================
-- TABLE: subjects
-- =========================================================
CREATE TABLE IF NOT EXISTS subjects (
  subject_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_school_id  UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  subject_code       VARCHAR(40)  NOT NULL,
  subject_name       VARCHAR(120) NOT NULL,
  subject_desc       TEXT,
  subject_slug       VARCHAR(160) NOT NULL,

  -- Single image (2-slot + retensi 30 hari)
  subject_image_url                   TEXT,
  subject_image_object_key            TEXT,
  subject_image_url_old               TEXT,
  subject_image_object_key_old        TEXT,
  subject_image_delete_pending_until  TIMESTAMPTZ,

  subject_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  subject_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subject_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subject_deleted_at TIMESTAMPTZ
);

-- Pair unik untuk tenant-safe lookup
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_id_school
  ON subjects (subject_id, subject_school_id);

-- Indexing subjects
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_code_per_school_alive
  ON subjects (subject_school_id, lower(subject_code))
  WHERE subject_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_slug_per_school_alive
  ON subjects (subject_school_id, lower(subject_slug))
  WHERE subject_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subject_school_alive
  ON subjects (subject_school_id)
  WHERE subject_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subject_active_alive
  ON subjects (subject_is_active)
  WHERE subject_deleted_at IS NULL;

-- Pencarian cepat by name
CREATE INDEX IF NOT EXISTS gin_subject_name_trgm_alive
  ON subjects USING GIN (lower(subject_name) gin_trgm_ops)
  WHERE subject_deleted_at IS NULL;

-- Kandidat purge image lama
CREATE INDEX IF NOT EXISTS idx_subject_image_purge_due
  ON subjects (subject_image_delete_pending_until)
  WHERE subject_image_object_key_old IS NOT NULL;


  
-- +migrate Up
-- =========================================================
-- TABLE: class_subjects
-- Relasi per tenant antara Class Parent ↔ Subject
-- Snapshot cache diambil dari subjects & class_parents
-- =========================================================

CREATE TABLE IF NOT EXISTS class_subjects (
  class_subject_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_subject_school_id     UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Optional slug (unique alive per tenant)
  class_subject_slug          VARCHAR(160),

  -- Display / akademik
  class_subject_order_index       INT,
  class_subject_hours_per_week    INT,
  class_subject_min_passing_score INT,
  class_subject_weight_on_report  INT,
  class_subject_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subject_desc              TEXT,

  -- Bobot (SMALLINT)
  class_subject_weight_assignment SMALLINT,
  class_subject_weight_quiz       SMALLINT,
  class_subject_weight_mid        SMALLINT,
  class_subject_weight_final      SMALLINT,
  class_subject_min_attendance_percent SMALLINT,

  /* =====================================================
       SUBJECT CACHE  + FK (TENANT-SAFE)
     ===================================================== */
  class_subject_subject_id            UUID NOT NULL,
  class_subject_subject_name_cache    VARCHAR(160),
  class_subject_subject_code_cache    VARCHAR(80),
  class_subject_subject_slug_cache    VARCHAR(160),
  class_subject_subject_url_cache     TEXT,

  /* =====================================================
       CLASS PARENT CACHE + FK (TENANT-SAFE)
     ===================================================== */
  class_subject_class_parent_id          UUID NOT NULL,
  class_subject_class_parent_code_cache  VARCHAR(80),
  class_subject_class_parent_slug_cache  VARCHAR(160),
  class_subject_class_parent_level_cache SMALLINT,
  class_subject_class_parent_url_cache   TEXT,
  class_subject_class_parent_name_cache  VARCHAR(160),

  /* =====================================================
       Lifecycle
     ===================================================== */
  class_subject_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_subject_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_deleted_at  TIMESTAMPTZ,

  /* =====================================================
       Tenant-safe identity pair
     ===================================================== */
  UNIQUE (class_subject_id, class_subject_school_id),

  /* =====================================================
       Guard nilai bobot
     ===================================================== */
  CONSTRAINT ck_class_subject_weights_nonneg CHECK (
    (class_subject_weight_assignment IS NULL OR class_subject_weight_assignment >= 0) AND
    (class_subject_weight_quiz       IS NULL OR class_subject_weight_quiz       >= 0) AND
    (class_subject_weight_mid        IS NULL OR class_subject_weight_mid        >= 0) AND
    (class_subject_weight_final      IS NULL OR class_subject_weight_final      >= 0)
  ),

  /* =====================================================
       FK KOMPOSIT TENANT-SAFE
     ===================================================== */
  CONSTRAINT fk_class_subject_subject_same_school
    FOREIGN KEY (class_subject_subject_id, class_subject_school_id)
    REFERENCES subjects (subject_id, subject_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_class_subject_parent_same_school
    FOREIGN KEY (class_subject_class_parent_id, class_subject_school_id)
    REFERENCES class_parents (class_parent_id, class_parent_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- =========================================================
-- INDEXES
-- =========================================================

-- active only
CREATE INDEX IF NOT EXISTS idx_class_subject_active_alive
  ON class_subjects (class_subject_is_active)
  WHERE class_subject_deleted_at IS NULL;

-- tenant
CREATE INDEX IF NOT EXISTS idx_class_subjects_school
  ON class_subjects (class_subject_school_id);

-- filter per parent
CREATE INDEX IF NOT EXISTS idx_class_subjects_parent
  ON class_subjects (class_subject_class_parent_id);

-- unique parent + subject per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subject_per_parent_subject_alive
  ON class_subjects (
    class_subject_school_id,
    class_subject_class_parent_id,
    class_subject_subject_id
  )
  WHERE class_subject_deleted_at IS NULL;

-- slug unik per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subject_slug_per_tenant_alive
  ON class_subjects (class_subject_school_id, LOWER(class_subject_slug))
  WHERE class_subject_deleted_at IS NULL
    AND class_subject_slug IS NOT NULL;

-- trigram search untuk slug
CREATE INDEX IF NOT EXISTS gin_class_subject_slug_trgm_alive
  ON class_subjects USING GIN (LOWER(class_subject_slug) gin_trgm_ops)
  WHERE class_subject_deleted_at IS NULL
    AND class_subject_slug IS NOT NULL;
