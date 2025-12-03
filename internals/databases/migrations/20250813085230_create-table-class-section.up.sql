-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE & fuzzy)
CREATE EXTENSION IF NOT EXISTS btree_gist; -- optional

-- =========================================================
-- ENUM (idempotent) - CSST enrollment mode
-- =========================================================
DO $$ BEGIN
  CREATE TYPE class_section_subject_teachers_enrollment_mode AS ENUM ('self_select','assigned','hybrid');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- ENUMS lain (kalau belum ada)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('offline','online','hybrid');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_entry_mode_enum') THEN
    CREATE TYPE attendance_entry_mode_enum AS ENUM ('teacher_only', 'student_only', 'both');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_sections (JSONB snapshots utk people & room)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_sections (
  class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant
  class_section_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Identitas
  class_section_slug  VARCHAR(160) NOT NULL,
  class_section_name  VARCHAR(100) NOT NULL,
  class_section_code  VARCHAR(50),

  -- Jadwal sederhana
  class_section_schedule TEXT,

  -- Kuota (mirip classes)
  --  - class_section_quota_total → kapasitas maksimal (limit)
  --  - class_section_quota_taken → sudah terdaftar (count)
  class_section_quota_total INT,
  class_section_quota_taken INT NOT NULL DEFAULT 0,

  -- (ACTIVE ONLY stats)
  class_section_total_students_active        INT NOT NULL DEFAULT 0,
  class_section_total_students_male          INT NOT NULL DEFAULT 0,
  class_section_total_students_female        INT NOT NULL DEFAULT 0,
  class_section_total_students_male_active   INT NOT NULL DEFAULT 0,
  class_section_total_students_female_active INT NOT NULL DEFAULT 0,
  class_section_stats JSONB,

  -- Meeting / Group
  class_section_group_url  TEXT,

  -- Image (2-slot + retensi)
  class_section_image_url                  TEXT,
  class_section_image_object_key           TEXT,
  class_section_image_url_old              TEXT,
  class_section_image_object_key_old       TEXT,
  class_section_image_delete_pending_until TIMESTAMPTZ,

  -- Join code (hash)
  class_section_teacher_code_hash BYTEA,
  class_section_teacher_code_set_at TIMESTAMPTZ,
  class_section_student_code_hash BYTEA,
  class_section_student_code_set_at TIMESTAMPTZ,

  /* =====================================================
     SNAPSHOTS (dibekukan saat update)
     ===================================================== */

  -- Class (dipakai juga utk FK ke classes)
  class_section_class_id UUID,
  class_section_class_name_cache VARCHAR(160),
  class_section_class_slug_cache VARCHAR(160),

  -- Parent
  class_section_class_parent_id UUID,
  class_section_class_parent_name_cache VARCHAR(160),
  class_section_class_parent_slug_cache VARCHAR(160),
  class_section_class_parent_level_cache SMALLINT,

  -- People (teacher/assistant/leader)
  class_section_school_teacher_id              UUID,
  class_section_school_teacher_slug_cache      VARCHAR(100),
  class_section_school_teacher_cache           JSONB,

  class_section_assistant_school_teacher_id    UUID,
  class_section_assistant_school_teacher_slug_cache  VARCHAR(100),
  class_section_assistant_school_teacher_cache       JSONB,

  class_section_leader_school_student_id       UUID,
  class_section_leader_school_student_slug_cache     VARCHAR(100),
  class_section_leader_school_student_cache          JSONB,

  -- Room
  class_section_class_room_id              UUID,
  class_section_class_room_slug_cache      VARCHAR(100),
  class_section_class_room_name_cache      VARCHAR(160),
  class_section_class_room_location_cache  TEXT,
  class_section_class_room_cache           JSONB,

  -- TERM (dipakai juga utk FK ke academic_terms)
  class_section_academic_term_id UUID,
  class_section_academic_term_name_cache TEXT,
  class_section_academic_term_slug_cache TEXT,
  class_section_academic_term_academic_year_cache TEXT,
  class_section_academic_term_angkatan_cache INT,

  /* =====================================================
     SETTINGS untuk CSST
     ===================================================== */
  class_section_subject_teachers_enrollment_mode class_section_subject_teachers_enrollment_mode
    NOT NULL DEFAULT 'self_select',
  class_section_subject_teachers_self_select_requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
  class_section_subject_teachers_max_subjects_per_student INT,

  -- TOTAL CSST (ALL + ACTIVE)
  class_section_total_class_class_section_subject_teachers         INTEGER NOT NULL DEFAULT 0,
  class_section_total_class_class_section_subject_teachers_active  INTEGER NOT NULL DEFAULT 0,

  -- Status & audit
  class_section_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_deleted_at TIMESTAMPTZ,

  /* ========================= CHECKS ============================ */

  -- Kuota non-negatif & taken ≤ total (kalau total di-set)
  CONSTRAINT ck_section_quota_total_nonneg
    CHECK (class_section_quota_total IS NULL OR class_section_quota_total >= 0),

  CONSTRAINT ck_section_quota_taken_nonneg
    CHECK (class_section_quota_taken >= 0),

  CONSTRAINT ck_section_quota_taken_le_total
    CHECK (class_section_quota_total IS NULL OR class_section_quota_taken <= class_section_quota_total),

  -- URL group opsional tapi kalau ada harus http/https
  CONSTRAINT ck_section_group_url_scheme
    CHECK (class_section_group_url IS NULL OR class_section_group_url ~* '^(https?)://'),

  CONSTRAINT ck_subject_teachers_max_subjects_nonneg
    CHECK (class_section_subject_teachers_max_subjects_per_student IS NULL
           OR class_section_subject_teachers_max_subjects_per_student >= 0),

  /* ============== FK KOMPOSIT (tenant-safe) ==================== */
  CONSTRAINT fk_section_class_same_school
    FOREIGN KEY (class_section_class_id, class_section_school_id)
    REFERENCES classes (class_id, class_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_term_same_school
    FOREIGN KEY (class_section_academic_term_id, class_section_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- tenant-safe pair
  CONSTRAINT uq_class_section_id_school UNIQUE (class_section_id, class_section_school_id)
);

-- =========================================================
-- INDEXING & OPTIMIZATION
-- =========================================================

-- Slug unik per tenant (alive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_slug_per_school_alive
  ON class_sections (class_section_school_id, lower(class_section_slug))
  WHERE class_section_deleted_at IS NULL;

-- Code unik per class (alive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_code_per_class_alive
  ON class_sections (class_section_class_id, lower(class_section_code))
  WHERE class_section_deleted_at IS NULL AND class_section_code IS NOT NULL;

-- Lookup class+school (alive)
CREATE INDEX IF NOT EXISTS idx_section_class_school_alive
  ON class_sections (class_section_class_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

-- Lookup term+school (alive)
CREATE INDEX IF NOT EXISTS idx_section_term_school_alive
  ON class_sections (class_section_academic_term_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

-- Snapshot ID lookups (alive)
CREATE INDEX IF NOT EXISTS idx_section_teacher_id_alive
  ON class_sections (class_section_school_teacher_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_asst_teacher_id_alive
  ON class_sections (class_section_assistant_school_teacher_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_leader_student_id_alive
  ON class_sections (class_section_leader_school_student_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_room_id_alive
  ON class_sections (class_section_class_room_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

-- Scope tenant + active + recent
CREATE INDEX IF NOT EXISTS ix_section_school_active_created
  ON class_sections (class_section_school_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- Pencarian fuzzy nama & slug (alive)
CREATE INDEX IF NOT EXISTS gin_section_name_trgm_alive
  ON class_sections USING GIN (lower(class_section_name) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_section_slug_trgm_alive
  ON class_sections USING GIN (lower(class_section_slug) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

-- Mode enrollment (alive)
CREATE INDEX IF NOT EXISTS ix_section_subject_teachers_enrollment_mode_alive
  ON class_sections (class_section_subject_teachers_enrollment_mode, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- BRIN untuk waktu create
CREATE INDEX IF NOT EXISTS brin_section_created_at
  ON class_sections USING BRIN (class_section_created_at);

COMMIT;
