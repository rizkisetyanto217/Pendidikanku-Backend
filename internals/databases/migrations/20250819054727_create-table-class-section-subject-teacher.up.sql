-- +migrate Up
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ENUM: class_delivery_mode_enum
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('offline','online','hybrid');
  END IF;
END$$;

-- ======================================================================
-- PREREQ tenant-safe uniques (class_sections, class_subjects, school_teachers,
--                             class_rooms, school_students, academic_terms)
-- ======================================================================

DO $$
BEGIN
  -- class_sections
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_class_sections_id_tenant'
      AND conrelid = 'class_sections'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_class_sections_id_tenant'
        AND i.indrelid = 'class_sections'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE class_sections
               ADD CONSTRAINT uq_class_sections_id_tenant
               UNIQUE USING INDEX uq_class_sections_id_tenant';
    ELSE
      ALTER TABLE class_sections
        ADD CONSTRAINT uq_class_sections_id_tenant
        UNIQUE (class_section_id, class_section_school_id);
    END IF;
  END IF;

  -- class_subjects
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_class_subjects_id_tenant'
      AND conrelid = 'class_subjects'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_class_subjects_id_tenant'
        AND i.indrelid = 'class_subjects'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE class_subjects
               ADD CONSTRAINT uq_class_subjects_id_tenant
               UNIQUE USING INDEX uq_class_subjects_id_tenant';
    ELSE
      ALTER TABLE class_subjects
        ADD CONSTRAINT uq_class_subjects_id_tenant
        UNIQUE (class_subject_id, class_subject_school_id);
    END IF;
  END IF;

  -- school_teachers
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_school_teachers_id_tenant'
      AND conrelid = 'school_teachers'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_school_teachers_id_tenant'
        AND i.indrelid = 'school_teachers'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE school_teachers
               ADD CONSTRAINT uq_school_teachers_id_tenant
               UNIQUE USING INDEX uq_school_teachers_id_tenant';
    ELSE
      ALTER TABLE school_teachers
        ADD CONSTRAINT uq_school_teachers_id_tenant
        UNIQUE (school_teacher_id, school_teacher_school_id);
    END IF;
  END IF;

  -- class_rooms
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_class_rooms_id_tenant'
      AND conrelid = 'class_rooms'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_class_rooms_id_tenant'
        AND i.indrelid = 'class_rooms'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE class_rooms
               ADD CONSTRAINT uq_class_rooms_id_tenant
               UNIQUE USING INDEX uq_class_rooms_id_tenant';
    ELSE
      ALTER TABLE class_rooms
        ADD CONSTRAINT uq_class_rooms_id_tenant
        UNIQUE (class_room_id, class_room_school_id);
    END IF;
  END IF;

  -- school_students
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_school_students_id_tenant'
      AND conrelid = 'school_students'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_school_students_id_tenant'
        AND i.indrelid = 'school_students'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE school_students
               ADD CONSTRAINT uq_school_students_id_tenant
               UNIQUE USING INDEX uq_school_students_id_tenant';
    ELSE
      ALTER TABLE school_students
        ADD CONSTRAINT uq_school_students_id_tenant
        UNIQUE (school_student_id, school_student_school_id);
    END IF;
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'uq_academic_terms_id_tenant'
      AND conrelid = 'academic_terms'::regclass
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM pg_class c
      JOIN pg_index i ON i.indexrelid = c.oid
      WHERE c.relname = 'uq_academic_terms_id_tenant'
        AND i.indrelid = 'academic_terms'::regclass
        AND i.indisunique
    ) THEN
      EXECUTE 'ALTER TABLE academic_terms
               ADD CONSTRAINT uq_academic_terms_id_tenant
               UNIQUE USING INDEX uq_academic_terms_id_tenant';
    ELSE
      ALTER TABLE academic_terms
        ADD CONSTRAINT uq_academic_terms_id_tenant
        UNIQUE (academic_term_id, academic_term_school_id);
    END IF;
  END IF;
END$$;

-- ======================================================================
-- TABLE: class_section_subject_teachers (CSST)
-- ======================================================================

CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  csst_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  csst_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Identitas & fasilitas
  csst_slug        VARCHAR(160),
  csst_description TEXT,
  csst_group_url   TEXT,

  -- Agregat & quota
  csst_total_attendance      INT NOT NULL DEFAULT 0,
  csst_total_meetings_target INT,

  -- quota_total / quota_taken
  csst_quota_total INT,
  csst_quota_taken INT NOT NULL DEFAULT 0,

  -- Total assessments (by type)
  csst_total_assessments               INT NOT NULL DEFAULT 0,
  csst_total_assessments_training      INT NOT NULL DEFAULT 0,
  csst_total_assessments_daily_exam    INT NOT NULL DEFAULT 0,
  csst_total_assessments_exam          INT NOT NULL DEFAULT 0,
  csst_total_students_passed           INT NOT NULL DEFAULT 0,

  csst_delivery_mode class_delivery_mode_enum NOT NULL DEFAULT 'offline',

  csst_school_attendance_entry_mode_cache attendance_entry_mode_enum,

  /* =======================
     SECTION cache
     ======================= */
  csst_class_section_id         UUID NOT NULL,
  csst_class_section_slug_cache VARCHAR(160),
  csst_class_section_name_cache VARCHAR(160),
  csst_class_section_code_cache VARCHAR(50),
  csst_class_section_url_cache  TEXT,

  /* =======================
     ROOM cache
     ======================= */
  csst_class_room_id             UUID,
  csst_class_room_slug_cache     VARCHAR(160),
  csst_class_room_cache          JSONB,
  csst_class_room_name_cache     TEXT GENERATED ALWAYS AS ((csst_class_room_cache->>'name')) STORED,
  csst_class_room_slug_cache_gen TEXT GENERATED ALWAYS AS ((csst_class_room_cache->>'slug')) STORED,
  csst_class_room_location_cache TEXT GENERATED ALWAYS AS ((csst_class_room_cache->>'location')) STORED,

  /* =======================
     PEOPLE cache (teacher & assistant)
     ======================= */
  csst_school_teacher_id         UUID,
  csst_school_teacher_slug_cache VARCHAR(160),
  csst_school_teacher_cache      JSONB,

  csst_assistant_school_teacher_id          UUID,
  csst_assistant_school_teacher_slug_cache  VARCHAR(160),
  csst_assistant_school_teacher_cache       JSONB,

  csst_school_teacher_name_cache            TEXT GENERATED ALWAYS AS ((csst_school_teacher_cache->>'name')) STORED,
  csst_assistant_school_teacher_name_cache  TEXT GENERATED ALWAYS AS ((csst_assistant_school_teacher_cache->>'name')) STORED,

  /* =======================
     SUBJECT cache (via CLASS_SUBJECT)
     ======================= */
  csst_total_books      INT NOT NULL DEFAULT 0,
  csst_class_subject_id UUID NOT NULL,
  csst_subject_id       UUID,
  csst_subject_name_cache VARCHAR(160),
  csst_subject_code_cache VARCHAR(80),
  csst_subject_slug_cache VARCHAR(160),

  /* =======================
     ACADEMIC_TERM cache
     ======================= */
  csst_academic_term_id UUID,
  csst_academic_term_name_cache     VARCHAR(160),
  csst_academic_term_slug_cache     VARCHAR(160),
  csst_academic_year_cache          VARCHAR(160),
  csst_academic_term_angkatan_cache INT,

  /* =======================
     KKM cache per CSST
     ======================= */
  csst_min_passing_score_class_subject_cache INT,

  -- Status & audit
  csst_status       class_status_enum NOT NULL DEFAULT 'active',
  csst_completed_at TIMESTAMPTZ,
  csst_created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  csst_updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  csst_deleted_at   TIMESTAMPTZ,

  -- CHECKS
  CONSTRAINT ck_csst_capacity_nonneg
    CHECK (csst_quota_total IS NULL OR csst_quota_total >= 0),
  CONSTRAINT ck_csst_enrolled_nonneg
    CHECK (csst_quota_taken >= 0),
  CONSTRAINT ck_csst_counts_nonneg
    CHECK (
      csst_total_attendance             >= 0 AND
      csst_total_assessments            >= 0 AND
      csst_total_assessments_training   >= 0 AND
      csst_total_assessments_daily_exam >= 0 AND
      csst_total_assessments_exam       >= 0 AND
      csst_total_students_passed        >= 0 AND
      csst_total_books                  >= 0
    ),
  CONSTRAINT ck_csst_room_cache_is_object
    CHECK (csst_class_room_cache IS NULL OR jsonb_typeof(csst_class_room_cache) = 'object'),
  CONSTRAINT ck_csst_teacher_cache_is_object
    CHECK (csst_school_teacher_cache IS NULL OR jsonb_typeof(csst_school_teacher_cache) = 'object'),
  CONSTRAINT ck_csst_asst_teacher_cache_is_object
    CHECK (csst_assistant_school_teacher_cache IS NULL OR jsonb_typeof(csst_assistant_school_teacher_cache) = 'object'),

  -- TENANT-SAFE FKs
  CONSTRAINT fk_csst_section_tenant FOREIGN KEY (
    csst_class_section_id,
    csst_school_id
  ) REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_class_subject_tenant FOREIGN KEY (
    csst_class_subject_id,
    csst_school_id
  ) REFERENCES class_subjects (class_subject_id, class_subject_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_teacher_tenant FOREIGN KEY (
    csst_school_teacher_id,
    csst_school_id
  ) REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_csst_assistant_teacher_tenant FOREIGN KEY (
    csst_assistant_school_teacher_id,
    csst_school_id
  ) REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_csst_room_tenant FOREIGN KEY (
    csst_class_room_id,
    csst_school_id
  ) REFERENCES class_rooms (class_room_id, class_room_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- ======================================================================
-- INDEXES
-- ======================================================================

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_tenant
  ON class_section_subject_teachers (csst_id, csst_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    csst_school_id,
    csst_class_section_id,
    csst_class_subject_id,
    csst_school_teacher_id
  )
  WHERE csst_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_one_active_per_section_subject_alive
  ON class_section_subject_teachers (
    csst_school_id,
    csst_class_section_id,
    csst_class_subject_id
  )
  WHERE csst_deleted_at IS NULL
    AND csst_status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_slug_per_tenant_alive
  ON class_section_subject_teachers (
    csst_school_id,
    LOWER(csst_slug)
  )
  WHERE csst_deleted_at IS NULL
    AND csst_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_csst_school_alive
  ON class_section_subject_teachers (csst_school_id)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_section_alive
  ON class_section_subject_teachers (csst_class_section_id)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_subject_alive
  ON class_section_subject_teachers (csst_class_subject_id)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_school_teacher_alive
  ON class_section_subject_teachers (csst_school_teacher_id)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_room_alive
  ON class_section_subject_teachers (csst_class_room_id)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_csst_slug_trgm_alive
  ON class_section_subject_teachers
  USING GIN (LOWER(csst_slug) gin_trgm_ops)
  WHERE csst_deleted_at IS NULL
    AND csst_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_csst_created_at
  ON class_section_subject_teachers USING BRIN (csst_created_at);

CREATE INDEX IF NOT EXISTS idx_csst_capacity_alive
  ON class_section_subject_teachers (csst_quota_total)
  WHERE csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_enrolled_count_alive
  ON class_section_subject_teachers (csst_quota_taken)
  WHERE csst_deleted_at IS NULL;

-- SUBJECT cache search
CREATE INDEX IF NOT EXISTS gin_csst_subject_name_cache_trgm_alive
  ON class_section_subject_teachers
  USING GIN (LOWER(csst_subject_name_cache) gin_trgm_ops)
  WHERE csst_deleted_at IS NULL
    AND csst_subject_name_cache IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_csst_subject_code_cache_alive
  ON class_section_subject_teachers (LOWER(csst_subject_code_cache))
  WHERE csst_deleted_at IS NULL
    AND csst_subject_code_cache IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_csst_subject_slug_cache_trgm_alive
  ON class_section_subject_teachers
  USING GIN (LOWER(csst_subject_slug_cache) gin_trgm_ops)
  WHERE csst_deleted_at IS NULL
    AND csst_subject_slug_cache IS NOT NULL;

-- ======================================================================
-- COMMENT
-- ======================================================================

COMMENT ON COLUMN class_section_subject_teachers.csst_min_passing_score_class_subject_cache IS
  'KKM bawaan dari class_subjects.class_subject_min_passing_score (pure cache, jangan di-edit manual)';

COMMENT ON COLUMN class_subjects.class_subject_min_passing_score IS
  'KKM dasar per kombinasi (class_parent + subject); dicache ke CSST sebagai csst_min_passing_score_class_subject_cache';

COMMIT;

-- +migrate Up
BEGIN;

-- =========================================================
-- TABLE: student_class_section_subject_teachers (SCSST)
-- =========================================================
CREATE TABLE IF NOT EXISTS student_class_section_subject_teachers (
  student_csst_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  student_csst_school_id    UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Anchor
  student_csst_student_id   UUID NOT NULL,
  student_csst_csst_id      UUID NOT NULL,

  -- Status mapping
  student_csst_is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  student_csst_from         DATE,
  student_csst_to           DATE,

  -- Nilai ringkas
  student_csst_score_total      NUMERIC(6,2),
  student_csst_score_max_total  NUMERIC(6,2) DEFAULT 100,
  student_csst_score_percent    NUMERIC(5,2)
    GENERATED ALWAYS AS (
      CASE
        WHEN student_csst_score_total IS NULL
          OR student_csst_score_max_total IS NULL
          OR student_csst_score_max_total = 0
        THEN NULL
        ELSE ROUND(
          (student_csst_score_total
           / student_csst_score_max_total) * 100.0, 2
        )
      END
    ) STORED,
  student_csst_grade_letter     VARCHAR(8),
  student_csst_grade_point      NUMERIC(3,2),
  student_csst_is_passed        BOOLEAN,

  -- Cache user profile & siswa (pendek)
  student_csst_user_profile_name_cache          VARCHAR(80),
  student_csst_user_profile_avatar_url_cache    VARCHAR(255),
  student_csst_user_profile_whatsapp_url_cache        VARCHAR(50),
  student_csst_user_profile_parent_name_cache   VARCHAR(80),
  student_csst_user_profile_parent_wa_url_cache VARCHAR(50),
  student_csst_user_profile_gender_cache        VARCHAR(20),
  student_csst_school_student_code_cache  VARCHAR(50),

  -- Riwayat intervensi/remedial
  student_csst_edits_history JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_student_csst_edits_history_is_array CHECK (
    jsonb_typeof(student_csst_edits_history) = 'array'
  ),

  -- NOTES
  student_csst_student_notes                 TEXT,
  student_csst_student_notes_updated_at      TIMESTAMPTZ,

  -- Catatan dari wali kelas (homeroom)
  student_csst_homeroom_notes                TEXT,
  student_csst_homeroom_notes_updated_at     TIMESTAMPTZ,

  -- Catatan dari guru mapel (subject teacher)
  student_csst_subject_teacher_notes         TEXT,
  student_csst_subject_teacher_notes_updated_at TIMESTAMPTZ,

  -- Admin & meta
  student_csst_slug VARCHAR(160),
  student_csst_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_student_csst_meta_is_object CHECK (
    jsonb_typeof(student_csst_meta) = 'object'
  ),

  -- Audit
  student_csst_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_csst_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_csst_deleted_at TIMESTAMPTZ,

  /* ===== Tenant-safe FKs ===== */
  CONSTRAINT fk_student_csst_student_tenant FOREIGN KEY (
    student_csst_student_id,
    student_csst_school_id
  ) REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_student_csst_csst_tenant FOREIGN KEY (
    student_csst_csst_id,
    student_csst_school_id
  ) REFERENCES class_section_subject_teachers (
        csst_id,
        csst_school_id
      )
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- =========================================================
-- INDEXES (SCSST)
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_student_csst_id_tenant
  ON student_class_section_subject_teachers (
    student_csst_id,
    student_csst_school_id
  );

CREATE UNIQUE INDEX IF NOT EXISTS uq_student_csst_one_active_per_student_csst_alive
  ON student_class_section_subject_teachers (
    student_csst_school_id,
    student_csst_student_id,
    student_csst_csst_id
  )
  WHERE student_csst_deleted_at IS NULL
    AND student_csst_is_active = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS uq_student_csst_slug_per_tenant_alive
  ON student_class_section_subject_teachers (
    student_csst_school_id,
    LOWER(student_csst_slug)
  )
  WHERE student_csst_deleted_at IS NULL
    AND student_csst_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_student_csst_school_alive
  ON student_class_section_subject_teachers (student_csst_school_id)
  WHERE student_csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_csst_student_alive
  ON student_class_section_subject_teachers (student_csst_student_id)
  WHERE student_csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_csst_csst_alive
  ON student_class_section_subject_teachers (student_csst_csst_id)
  WHERE student_csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_csst_active_alive
  ON student_class_section_subject_teachers (student_csst_is_active)
  WHERE student_csst_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_student_csst_created_at
  ON student_class_section_subject_teachers USING BRIN (student_csst_created_at);

CREATE INDEX IF NOT EXISTS gin_student_csst_edits_history
  ON student_class_section_subject_teachers
  USING GIN (student_csst_edits_history jsonb_path_ops);

COMMIT;
