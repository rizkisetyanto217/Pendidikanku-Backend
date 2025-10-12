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
  subject_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

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
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_id_masjid
  ON subjects (subject_id, subject_masjid_id);

-- Indexing subjects
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_code_per_masjid_alive
  ON subjects (subject_masjid_id, lower(subject_code))
  WHERE subject_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_slug_per_masjid_alive
  ON subjects (subject_masjid_id, lower(subject_slug))
  WHERE subject_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subject_masjid_alive
  ON subjects (subject_masjid_id)
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





-- =========================================================
-- TABLE: class_subjects (RELASI KE class_parents, BUKAN classes)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subject_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subject_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_subject_parent_id   UUID NOT NULL REFERENCES class_parents(class_parent_id) ON DELETE CASCADE,
  class_subject_subject_id  UUID NOT NULL REFERENCES subjects(subject_id) ON DELETE RESTRICT,

  -- >>> SLUG <<<
  class_subject_slug        VARCHAR(160),

  class_subject_order_index       INT,
  class_subject_hours_per_week    INT,
  class_subject_min_passing_score INT,
  class_subject_weight_on_report  INT,
  class_subject_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subject_desc              TEXT,

  -- Bobot penilaian (opsional)
  class_subject_weight_assignment SMALLINT,
  class_subject_weight_quiz       SMALLINT,
  class_subject_weight_mid        SMALLINT,
  class_subject_weight_final      SMALLINT,
  class_subject_min_attendance_percent SMALLINT,

  -- ============================
  -- Snapshots dari subjects
  -- ============================
  class_subject_subject_name_snapshot VARCHAR(160),
  class_subject_subject_code_snapshot VARCHAR(80),
  class_subject_subject_slug_snapshot VARCHAR(160),
  class_subject_subject_url_snapshot  TEXT,

  -- ============================
  -- Snapshots dari class_parent
  -- ============================
  class_subject_parent_code_snapshot  VARCHAR(80),
  class_subject_parent_slug_snapshot  VARCHAR(160),
  class_subject_parent_level_snapshot SMALLINT,
  class_subject_parent_url_snapshot   TEXT,
  class_subject_parent_name_snapshot  VARCHAR(160),

  -- ============================
  -- Audit & lifecycle
  -- ============================
  class_subject_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_subject_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_deleted_at  TIMESTAMPTZ,

  -- (opsional) tenant-safe pair agar join-by-tenant aman
  UNIQUE (class_subject_id, class_subject_masjid_id),

  -- (opsional) guard bobot tidak negatif
  CONSTRAINT ck_class_subject_weights_nonneg CHECK (
    (class_subject_weight_assignment IS NULL OR class_subject_weight_assignment >= 0) AND
    (class_subject_weight_quiz       IS NULL OR class_subject_weight_quiz       >= 0) AND
    (class_subject_weight_mid        IS NULL OR class_subject_weight_mid        >= 0) AND
    (class_subject_weight_final      IS NULL OR class_subject_weight_final      >= 0)
  )
);

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_class_subject_active_alive
  ON class_subjects (class_subject_is_active)
  WHERE class_subject_deleted_at IS NULL;

-- Filter cepat per tenant/parent/subject
CREATE INDEX IF NOT EXISTS idx_class_subjects_masjid
  ON class_subjects (class_subject_masjid_id);

CREATE INDEX IF NOT EXISTS idx_class_subjects_parent
  ON class_subjects (class_subject_parent_id);

-- Unik kombinasi (hindari duplikat subject di parent yang sama, tenant-aware, hanya untuk baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subject_per_parent_subject_alive
  ON class_subjects (class_subject_masjid_id, class_subject_parent_id, class_subject_subject_id)
  WHERE class_subject_deleted_at IS NULL;

-- Unik SLUG per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subject_slug_per_tenant_alive
  ON class_subjects (class_subject_masjid_id, lower(class_subject_slug))
  WHERE class_subject_deleted_at IS NULL
    AND class_subject_slug IS NOT NULL;

-- Pencarian slug cepat
CREATE INDEX IF NOT EXISTS gin_class_subject_slug_trgm_alive
  ON class_subjects USING GIN (lower(class_subject_slug) gin_trgm_ops)
  WHERE class_subject_deleted_at IS NULL
    AND class_subject_slug IS NOT NULL;