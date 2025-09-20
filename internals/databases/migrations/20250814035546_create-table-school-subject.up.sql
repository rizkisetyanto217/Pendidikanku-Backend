BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index (ILIKE search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi btree + GIN (opsional)

-- =========================================================
-- TABLE: subjects
-- =========================================================
CREATE TABLE IF NOT EXISTS subjects (
  subjects_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  subjects_code       VARCHAR(40)  NOT NULL,
  subjects_name       VARCHAR(120) NOT NULL,
  subjects_desc       TEXT,
  subjects_slug       VARCHAR(160) NOT NULL,

  -- Single image (2-slot + retensi 30 hari)
  subjects_image_url                   TEXT,
  subjects_image_object_key            TEXT,
  subjects_image_url_old               TEXT,
  subjects_image_object_key_old        TEXT,
  subjects_image_delete_pending_until  TIMESTAMPTZ,

  subjects_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  subjects_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_deleted_at TIMESTAMPTZ
);

-- Indexing subjects
CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_code_per_masjid_alive
  ON subjects (subjects_masjid_id, lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_slug_per_masjid_alive
  ON subjects (subjects_masjid_id, lower(subjects_slug))
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_masjid_alive
  ON subjects (subjects_masjid_id)
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_active_alive
  ON subjects (subjects_is_active)
  WHERE subjects_deleted_at IS NULL;

-- Pencarian cepat by name (opsional)
CREATE INDEX IF NOT EXISTS gin_subjects_name_trgm_alive
  ON subjects USING GIN (subjects_name gin_trgm_ops)
  WHERE subjects_deleted_at IS NULL;

-- Kandidat purge image lama
CREATE INDEX IF NOT EXISTS idx_subjects_image_purge_due
  ON subjects (subjects_image_delete_pending_until)
  WHERE subjects_image_object_key_old IS NOT NULL;




-- =========================================================
-- TABLE: class_subjects
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subjects_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id)   ON DELETE CASCADE,
  class_subjects_class_id   UUID NOT NULL REFERENCES classes(class_id)     ON DELETE CASCADE,
  class_subjects_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,

  class_subjects_order_index       INT,
  class_subjects_hours_per_week    INT,
  class_subjects_min_passing_score INT,
  class_subjects_weight_on_report  INT,
  class_subjects_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subjects_desc              TEXT,

  -- Bobot penilaian (opsional)
  class_subjects_weight_assignment SMALLINT,
  class_subjects_weight_quiz       SMALLINT,
  class_subjects_weight_mid        SMALLINT,
  class_subjects_weight_final      SMALLINT,
  class_subjects_min_attendance_percent SMALLINT,

  -- Single image (2-slot + retensi 30 hari)
  class_subjects_image_url                   TEXT,
  class_subjects_image_object_key            TEXT,
  class_subjects_image_url_old               TEXT,
  class_subjects_image_object_key_old        TEXT,
  class_subjects_image_delete_pending_until  TIMESTAMPTZ,

  class_subjects_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_subjects_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_deleted_at  TIMESTAMPTZ
);

-- FK komposit ke classes (tenant-safe lookup umum)
CREATE INDEX IF NOT EXISTS idx_cs_class_masjid
  ON class_subjects (class_subjects_class_id, class_subjects_masjid_id)
  WHERE class_subjects_deleted_at IS NULL;


-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_class_subjects_subject_alive
  ON class_subjects (class_subjects_subject_id)
  WHERE class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_subjects_active_alive
  ON class_subjects (class_subjects_is_active)
  WHERE class_subjects_deleted_at IS NULL;

-- Kandidat purge image lama
CREATE INDEX IF NOT EXISTS idx_class_subjects_image_purge_due
  ON class_subjects (class_subjects_image_delete_pending_until)
  WHERE class_subjects_image_object_key_old IS NOT NULL;

COMMIT;