-- +migrate Up
/* =======================================================================
   MIGRATION: CSST (class_section_subject_teachers)
              + SCSST (student_class_section_subject_teachers)
   ======================================================================= */

BEGIN;
-- +migrate Up
/* =======================================================================
   MIGRATION: CSST (class_section_subject_teachers)
              FINAL VERSION — with KKM, meetings target, graded counts
   ======================================================================= */
-- =========================================================
-- EXTENSIONS (safe to repeat)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- ENUMS (idempotent)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('offline','online','hybrid');
  END IF;
END$$;

-- =========================================================
-- PREREQUISITES: UNIQUE CONSTRAINTS (tenant-safe)
-- =========================================================
DO $$
BEGIN
  PERFORM 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_tenant';
  IF NOT FOUND THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_tenant
      UNIQUE (class_section_id, class_section_school_id);
  END IF;

  PERFORM 1 FROM pg_constraint WHERE conname = 'uq_class_subject_books_id_tenant';
  IF NOT FOUND THEN
    ALTER TABLE class_subject_books
      ADD CONSTRAINT uq_class_subject_books_id_tenant
      UNIQUE (class_subject_book_id, class_subject_book_school_id);
  END IF;

  PERFORM 1 FROM pg_constraint WHERE conname = 'uq_school_teachers_id_tenant';
  IF NOT FOUND THEN
    ALTER TABLE school_teachers
      ADD CONSTRAINT uq_school_teachers_id_tenant
      UNIQUE (school_teacher_id, school_teacher_school_id);
  END IF;

  PERFORM 1 FROM pg_constraint WHERE conname = 'uq_class_rooms_id_tenant';
  IF NOT FOUND THEN
    ALTER TABLE class_rooms
      ADD CONSTRAINT uq_class_rooms_id_tenant
      UNIQUE (class_room_id, class_room_school_id);
  END IF;
END$$;

-- =========================================================
-- TABLE: class_section_subject_teachers (CSST)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teacher_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_section_subject_teacher_school_id        UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Identitas & fasilitas
  class_section_subject_teacher_slug             VARCHAR(160),
  class_section_subject_teacher_description      TEXT,
  class_section_subject_teacher_group_url        TEXT,

  -- Agregat & kapasitas
  class_section_subject_teacher_total_attendance INT NOT NULL DEFAULT 0,
  class_section_subject_teacher_total_meetings_target INT,
  class_section_subject_teacher_capacity         INT,
  class_section_subject_teacher_enrolled_count   INT NOT NULL DEFAULT 0,

  -- Assessment counts
  class_section_subject_teacher_total_assessments           INT NOT NULL DEFAULT 0,
  class_section_subject_teacher_total_assessments_graded    INT NOT NULL DEFAULT 0,
  class_section_subject_teacher_total_assessments_ungraded  INT NOT NULL DEFAULT 0,

  -- Delivery mode
  class_section_subject_teacher_delivery_mode class_delivery_mode_enum NOT NULL DEFAULT 'offline',

  -- =======================
  -- SECTION snapshot (no JSON)
  -- =======================
  class_section_subject_teacher_class_section_id            UUID NOT NULL,
  class_section_subject_teacher_class_section_slug_snapshot VARCHAR(160),
  class_section_subject_teacher_class_section_name_snapshot VARCHAR(160),
  class_section_subject_teacher_class_section_code_snapshot VARCHAR(50),
  class_section_subject_teacher_class_section_url_snapshot  TEXT,

  -- =======================
  -- ROOM snapshot (JSONB)
  -- =======================
  class_section_subject_teacher_class_room_id                UUID,
  class_section_subject_teacher_class_room_slug_snapshot     VARCHAR(160),
  class_section_subject_teacher_class_room_snapshot          JSONB,

  class_section_subject_teacher_class_room_name_snapshot     TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_class_room_snapshot->>'name')) STORED,
  class_section_subject_teacher_class_room_slug_snapshot_gen TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_class_room_snapshot->>'slug')) STORED,
  class_section_subject_teacher_class_room_location_snapshot TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_class_room_snapshot->>'location')) STORED,

  -- =======================
  -- TEACHER snapshot
  -- =======================
  class_section_subject_teacher_school_teacher_id            UUID NOT NULL,
  class_section_subject_teacher_school_teacher_slug_snapshot VARCHAR(160),
  class_section_subject_teacher_school_teacher_snapshot      JSONB,
  class_section_subject_teacher_school_teacher_name_snapshot TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_school_teacher_snapshot->>'name')) STORED,

  -- Assistant (opsional)
  class_section_subject_teacher_assistant_school_teacher_id            UUID,
  class_section_subject_teacher_assistant_school_teacher_slug_snapshot VARCHAR(160),
  class_section_subject_teacher_assistant_school_teacher_snapshot      JSONB,
  class_section_subject_teacher_assistant_school_teacher_name_snapshot TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_assistant_school_teacher_snapshot->>'name')) STORED,

  -- =======================
  -- CLASS_SUBJECT_BOOK snapshot
  -- =======================
  class_section_subject_teacher_class_subject_book_id            UUID NOT NULL,
  class_section_subject_teacher_class_subject_book_slug_snapshot VARCHAR(160),
  class_section_subject_teacher_class_subject_book_snapshot      JSONB,

  -- BOOK derived
  class_section_subject_teacher_book_title_snapshot        TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'book'->>'title')) STORED,
  class_section_subject_teacher_book_author_snapshot       TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'book'->>'author')) STORED,
  class_section_subject_teacher_book_slug_snapshot         TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'book'->>'slug')) STORED,
  class_section_subject_teacher_book_image_url_snapshot    TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'book'->>'image_url')) STORED,

  -- SUBJECT derived
  class_section_subject_teacher_subject_id_snapshot        UUID,
  class_section_subject_teacher_subject_name_snapshot      TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'subject'->>'name')) STORED,
  class_section_subject_teacher_subject_code_snapshot      TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'subject'->>'code')) STORED,
  class_section_subject_teacher_subject_slug_snapshot      TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_book_snapshot->'subject'->>'slug')) STORED,

  -- =======================
  -- NEW: KKM SNAPSHOT
  -- =======================
  class_section_subject_teacher_min_passing_score INT,

  -- Status & audit
  class_section_subject_teacher_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_subject_teacher_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_deleted_at  TIMESTAMPTZ,

  -- Checks
  CONSTRAINT ck_csst_capacity_nonneg CHECK (class_section_subject_teacher_capacity IS NULL OR class_section_subject_teacher_capacity >= 0),
  CONSTRAINT ck_csst_enrolled_nonneg CHECK (class_section_subject_teacher_enrolled_count >= 0),
  CONSTRAINT ck_csst_min_passing_score_nonneg CHECK (class_section_subject_teacher_min_passing_score IS NULL OR class_section_subject_teacher_min_passing_score >= 0),

  -- FKs (tenant-safe)
  CONSTRAINT fk_csst_section_tenant FOREIGN KEY (
    class_section_subject_teacher_class_section_id,
    class_section_subject_teacher_school_id
  ) REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_class_subject_book_tenant FOREIGN KEY (
    class_section_subject_teacher_class_subject_book_id,
    class_section_subject_teacher_school_id
  ) REFERENCES class_subject_books (class_subject_book_id, class_subject_book_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_teacher_tenant FOREIGN KEY (
    class_section_subject_teacher_school_teacher_id,
    class_section_subject_teacher_school_id
  ) REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_csst_assistant_teacher_tenant FOREIGN KEY (
    class_section_subject_teacher_assistant_school_teacher_id,
    class_section_subject_teacher_school_id
  ) REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_csst_room_tenant FOREIGN KEY (
    class_section_subject_teacher_class_room_id,
    class_section_subject_teacher_school_id
  ) REFERENCES class_rooms (class_room_id, class_room_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =========================================================
-- INDEXES
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_tenant
  ON class_section_subject_teachers (class_section_subject_teacher_id, class_section_subject_teacher_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_school_id,
    class_section_subject_teacher_class_section_id,
    class_section_subject_teacher_class_subject_book_id,
    class_section_subject_teacher_school_teacher_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_slug_per_tenant_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_school_id,
    LOWER(class_section_subject_teacher_slug)
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_csst_school_alive
  ON class_section_subject_teachers (class_section_subject_teacher_school_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_total_assessments_alive
  ON class_section_subject_teachers (class_section_subject_teacher_total_assessments)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_total_meetings_target
  ON class_section_subject_teachers (class_section_subject_teacher_total_meetings_target)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

COMMIT;


-- =========================================================
-- TABLE: student_class_section_subject_teachers (SCSST)
-- =========================================================
CREATE TABLE IF NOT EXISTS student_class_section_subject_teachers (
  student_class_section_subject_teacher_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_class_section_subject_teacher_school_id    UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Anchor
  student_class_section_subject_teacher_student_id   UUID NOT NULL,
  student_class_section_subject_teacher_csst_id      UUID NOT NULL,

  -- Status mapping
  student_class_section_subject_teacher_is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  student_class_section_subject_teacher_from         DATE,
  student_class_section_subject_teacher_to           DATE,

  -- Nilai ringkas
  student_class_section_subject_teacher_score_total      NUMERIC(6,2),
  student_class_section_subject_teacher_score_max_total  NUMERIC(6,2) DEFAULT 100,
  student_class_section_subject_teacher_score_percent    NUMERIC(5,2)
    GENERATED ALWAYS AS (
      CASE
        WHEN student_class_section_subject_teacher_score_total IS NULL
          OR student_class_section_subject_teacher_score_max_total IS NULL
          OR student_class_section_subject_teacher_score_max_total = 0
        THEN NULL
        ELSE ROUND(
          (student_class_section_subject_teacher_score_total
           / student_class_section_subject_teacher_score_max_total) * 100.0, 2
        )
      END
    ) STORED,
  student_class_section_subject_teacher_grade_letter     VARCHAR(8),
  student_class_section_subject_teacher_grade_point      NUMERIC(3,2),
  student_class_section_subject_teacher_is_passed        BOOLEAN,

  -- Snapshot user profile
  student_class_section_subject_teacher_user_profile_name_snapshot                 VARCHAR(80),
  student_class_section_subject_teacher_user_profile_avatar_url_snapshot           VARCHAR(255),
  student_class_section_subject_teacher_user_profile_whatsapp_url_snapshot         VARCHAR(50),
  student_class_section_subject_teacher_user_profile_parent_name_snapshot          VARCHAR(80),
  student_class_section_subject_teacher_user_profile_parent_whatsapp_url_snapshot  VARCHAR(50),

  -- Riwayat intervensi/remedial
  student_class_section_subject_teacher_edits_history JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_scsst_edits_history_is_array CHECK (
    jsonb_typeof(student_class_section_subject_teacher_edits_history) = 'array'
  ),

  -- Admin & meta
  student_class_section_subject_teacher_slug VARCHAR(160),
  student_class_section_subject_teacher_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_scsst_meta_is_object CHECK (
    jsonb_typeof(student_class_section_subject_teacher_meta) = 'object'
  ),

  -- Audit
  student_class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_section_subject_teacher_deleted_at TIMESTAMPTZ,

  /* ===== Tenant-safe FKs ===== */
  CONSTRAINT fk_scsst_student_tenant FOREIGN KEY (
    student_class_section_subject_teacher_student_id,
    student_class_section_subject_teacher_school_id
  ) REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_scsst_csst_tenant FOREIGN KEY (
    student_class_section_subject_teacher_csst_id,
    student_class_section_subject_teacher_school_id
  ) REFERENCES class_section_subject_teachers (
        class_section_subject_teacher_id,
        class_section_subject_teacher_school_id
      )
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================================================
-- INDEXES (SCSST)
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsst_id_tenant
  ON student_class_section_subject_teachers (
    student_class_section_subject_teacher_id,
    student_class_section_subject_teacher_school_id
  );

CREATE UNIQUE INDEX IF NOT EXISTS uq_scsst_one_active_per_student_csst_alive
  ON student_class_section_subject_teachers (
    student_class_section_subject_teacher_school_id,
    student_class_section_subject_teacher_student_id,
    student_class_section_subject_teacher_csst_id
  )
  WHERE student_class_section_subject_teacher_deleted_at IS NULL
    AND student_class_section_subject_teacher_is_active = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS uq_scsst_slug_per_tenant_alive
  ON student_class_section_subject_teachers (
    student_class_section_subject_teacher_school_id,
    LOWER(student_class_section_subject_teacher_slug)
  )
  WHERE student_class_section_subject_teacher_deleted_at IS NULL
    AND student_class_section_subject_teacher_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_scsst_school_alive
  ON student_class_section_subject_teachers (student_class_section_subject_teacher_school_id)
  WHERE student_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsst_student_alive
  ON student_class_section_subject_teachers (student_class_section_subject_teacher_student_id)
  WHERE student_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsst_csst_alive
  ON student_class_section_subject_teachers (student_class_section_subject_teacher_csst_id)
  WHERE student_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsst_active_alive
  ON student_class_section_subject_teachers (student_class_section_subject_teacher_is_active)
  WHERE student_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_scsst_created_at
  ON student_class_section_subject_teachers USING BRIN (student_class_section_subject_teacher_created_at);

CREATE INDEX IF NOT EXISTS gin_scsst_edits_history
  ON student_class_section_subject_teachers
  USING GIN (student_class_section_subject_teacher_edits_history jsonb_path_ops);

COMMIT;
