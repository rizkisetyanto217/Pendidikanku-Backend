BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE & fuzzy)
CREATE EXTENSION IF NOT EXISTS btree_gist; -- optional untuk kombinasi tertentu

-- =========================================================
-- PASTIKAN academic_terms punya UNIQUE (id, school_id)
-- (agar FK komposit tenant-safe bisa dibuat)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'uq_academic_terms_id_tenant'
  ) THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_tenant
      UNIQUE (academic_term_id, academic_term_school_id);
  END IF;
END$$;

-- =========================================================
-- ENUM: cara siswa terdaftar ke CSST (section×subject×teacher)
-- =========================================================
DO $$ BEGIN
  CREATE TYPE class_section_csst_enrollment_mode AS ENUM ('self_select','assigned','hybrid');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================================================
-- TABLE: class_sections — snapshots JSONB, hints & caches
-- =========================================================
CREATE TABLE IF NOT EXISTS class_sections (
  class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi inti
  class_section_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  class_section_class_id             UUID NOT NULL,
  class_section_teacher_id           UUID,
  class_section_assistant_teacher_id UUID,
  class_section_class_room_id        UUID,
  class_section_leader_student_id    UUID,  -- ketua kelas (school_students)

  -- Identitas
  class_section_slug  VARCHAR(160) NOT NULL,
  class_section_name  VARCHAR(100) NOT NULL,
  class_section_code  VARCHAR(50),

  -- Jadwal sederhana (biarkan TEXT untuk sekarang)
  class_section_schedule TEXT,

  -- Kapasitas & counter dasar
  class_section_capacity       INT,
  class_section_total_students INT NOT NULL DEFAULT 0,

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

  -- ===================== SNAPSHOTS (JSONB) =====================
  -- Class
  class_section_class_snapshot JSONB,
  class_section_class_slug_snap TEXT GENERATED ALWAYS AS ((class_section_class_snapshot->>'slug')) STORED,

  -- Parent
  class_section_parent_snapshot JSONB,
  class_section_parent_name_snap  TEXT GENERATED ALWAYS AS ((class_section_parent_snapshot->>'name')) STORED,
  class_section_parent_code_snap  TEXT GENERATED ALWAYS AS ((class_section_parent_snapshot->>'code')) STORED,
  class_section_parent_slug_snap  TEXT GENERATED ALWAYS AS ((class_section_parent_snapshot->>'slug')) STORED,
  class_section_parent_level_snap TEXT GENERATED ALWAYS AS ((class_section_parent_snapshot->>'level')) STORED,

  -- People
  class_section_teacher_snapshot           JSONB,
  class_section_assistant_teacher_snapshot JSONB,
  class_section_leader_student_snapshot    JSONB,
  class_section_teacher_name_snap           TEXT GENERATED ALWAYS AS ((class_section_teacher_snapshot->>'name')) STORED,
  class_section_assistant_teacher_name_snap TEXT GENERATED ALWAYS AS ((class_section_assistant_teacher_snapshot->>'name')) STORED,
  class_section_leader_student_name_snap    TEXT GENERATED ALWAYS AS ((class_section_leader_student_snapshot->>'name')) STORED,

  -- Room
  class_section_room_snapshot JSONB,
  class_section_room_name_snap     TEXT GENERATED ALWAYS AS ((class_section_room_snapshot->>'name')) STORED,
  class_section_room_slug_snap     TEXT GENERATED ALWAYS AS ((class_section_room_snapshot->>'slug')) STORED,
  class_section_room_location_snap TEXT GENERATED ALWAYS AS ((class_section_room_snapshot->>'location')) STORED,

  -- TERM
  class_section_term_id UUID,
  class_section_term_snapshot JSONB,
  class_section_term_name_snap       TEXT GENERATED ALWAYS AS ((class_section_term_snapshot->>'name')) STORED,
  class_section_term_slug_snap       TEXT GENERATED ALWAYS AS ((class_section_term_snapshot->>'slug')) STORED,
  class_section_term_year_label_snap TEXT GENERATED ALWAYS AS ((class_section_term_snapshot->>'year_label')) STORED,

  -- ===================== CSST: cache dan PENGATURAN =====================
  -- cache daftar assignment (section×subject×teacher)
  class_sections_csst JSONB NOT NULL DEFAULT '[]'::jsonb,
  class_sections_csst_count INT GENERATED ALWAYS AS (jsonb_array_length(class_sections_csst)) STORED,
  class_sections_csst_active_count INT GENERATED ALWAYS AS (
    jsonb_array_length(jsonb_path_query_array(class_sections_csst, '$ ? (@.is_active == true)'))
  ) STORED,

  -- >>> Pengaturan cara siswa masuk ke CSST <<<
  -- 'self_select' : siswa bebas memilih sendiri mapel/CSST
  -- 'assigned'    : wali/teacher yang menentukan
  -- 'hybrid'      : kombinasi (boleh pilih, tapi bisa di-lock/override)
  class_section_csst_enrollment_mode class_section_csst_enrollment_mode
    NOT NULL DEFAULT 'self_select',

  -- jika self-select/hybrid: apakah butuh approval guru/wali sebelum aktif?
  class_section_csst_self_select_requires_approval BOOLEAN NOT NULL DEFAULT FALSE,

  -- batas maksimal CSST yang boleh dipilih per siswa untuk section ini (NULL = tidak dibatasi)
  class_section_csst_max_subjects_per_student INT,

  -- tenggat waktu siswa boleh ganti pilihan (NULL = tidak dibatasi)
  class_section_csst_switch_deadline TIMESTAMPTZ,

  -- (Opsional) fitur kelas sebagai object konfigurasi umum
  class_section_features JSONB NOT NULL DEFAULT '{}'::jsonb,

  -- housekeeping snapshot
  class_section_snapshot_updated_at TIMESTAMPTZ,

  -- Status & audit
  class_section_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_deleted_at TIMESTAMPTZ,

  -- ========================= CHECKS ============================
  CONSTRAINT ck_section_capacity_nonneg
    CHECK (class_section_capacity IS NULL OR class_section_capacity >= 0),

  CONSTRAINT ck_section_total_nonneg
    CHECK (class_section_total_students >= 0),

  CONSTRAINT ck_section_total_le_capacity
    CHECK (
      class_section_capacity IS NULL
      OR class_section_total_students <= class_section_capacity
    ),

  CONSTRAINT ck_section_group_url_scheme
    CHECK (class_section_group_url IS NULL OR class_section_group_url ~* '^(https?)://'),

  -- JSON types
  CONSTRAINT ck_jsonb_class_obj    CHECK (class_section_class_snapshot             IS NULL OR jsonb_typeof(class_section_class_snapshot)            = 'object'),
  CONSTRAINT ck_jsonb_parent_obj   CHECK (class_section_parent_snapshot            IS NULL OR jsonb_typeof(class_section_parent_snapshot)          = 'object'),
  CONSTRAINT ck_jsonb_teacher_obj  CHECK (class_section_teacher_snapshot           IS NULL OR jsonb_typeof(class_section_teacher_snapshot)         = 'object'),
  CONSTRAINT ck_jsonb_asst_obj     CHECK (class_section_assistant_teacher_snapshot IS NULL OR jsonb_typeof(class_section_assistant_teacher_snapshot)= 'object'),
  CONSTRAINT ck_jsonb_leader_obj   CHECK (class_section_leader_student_snapshot    IS NULL OR jsonb_typeof(class_section_leader_student_snapshot)  = 'object'),
  CONSTRAINT ck_jsonb_room_obj     CHECK (class_section_room_snapshot              IS NULL OR jsonb_typeof(class_section_room_snapshot)            = 'object'),
  CONSTRAINT ck_jsonb_term_obj     CHECK (class_section_term_snapshot              IS NULL OR jsonb_typeof(class_section_term_snapshot)            = 'object'),
  CONSTRAINT ck_csec_csst_is_array CHECK (jsonb_typeof(class_sections_csst) = 'array'),
  CONSTRAINT ck_csec_features_obj  CHECK (jsonb_typeof(class_section_features) = 'object'),

  -- numeric constraints (baru)
  CONSTRAINT ck_csst_max_subjects_nonneg
    CHECK (class_section_csst_max_subjects_per_student IS NULL OR class_section_csst_max_subjects_per_student >= 0),

  -- ==================== FK KOMPOSIT (tenant-safe) ===============
  CONSTRAINT fk_section_class_same_school
    FOREIGN KEY (class_section_class_id, class_section_school_id)
    REFERENCES classes (class_id, class_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_section_teacher_same_school
    FOREIGN KEY (class_section_teacher_id, class_section_school_id)
    REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_assistant_teacher_same_school
    FOREIGN KEY (class_section_assistant_teacher_id, class_section_school_id)
    REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_leader_student_same_school
    FOREIGN KEY (class_section_leader_student_id, class_section_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_csec_term_same_school
    FOREIGN KEY (class_section_term_id, class_section_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Pair unik untuk join multi-tenant aman
  CONSTRAINT uq_class_section_id_school
    UNIQUE (class_section_id, class_section_school_id)
);

-- =========================================================
-- INDEXING & OPTIMIZATION
-- =========================================================

-- 1) Unik slug per school (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_slug_per_school_alive
  ON class_sections (class_section_school_id, lower(class_section_slug))
  WHERE class_section_deleted_at IS NULL;

-- 2) (Opsional) Unik code per class (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_code_per_class_alive
  ON class_sections (class_section_class_id, lower(class_section_code))
  WHERE class_section_deleted_at IS NULL AND class_section_code IS NOT NULL;

-- 3) FK-friendly composites (untuk join cepat; hanya baris hidup)
CREATE INDEX IF NOT EXISTS idx_section_class_school_alive
  ON class_sections (class_section_class_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_teacher_school_alive
  ON class_sections (class_section_teacher_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_assistant_teacher_school_alive
  ON class_sections (class_section_assistant_teacher_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_room_school_alive
  ON class_sections (class_section_class_room_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_leader_student_school_alive
  ON class_sections (class_section_leader_student_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_term_school_alive
  ON class_sections (class_section_term_id, class_section_school_id)
  WHERE class_section_deleted_at IS NULL;

-- 4) Listing umum (tenanted)
CREATE INDEX IF NOT EXISTS ix_section_school_active_created
  ON class_sections (class_section_school_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_class_active_created
  ON class_sections (class_section_class_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- (Opsional) listing per peran/room
CREATE INDEX IF NOT EXISTS ix_section_teacher_active_created
  ON class_sections (class_section_teacher_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_assistant_active_created
  ON class_sections (class_section_assistant_teacher_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_room_active_created
  ON class_sections (class_section_class_room_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- 5) Pencarian teks cepat (ILIKE/fuzzy) pada name/slug
CREATE INDEX IF NOT EXISTS gin_section_name_trgm_alive
  ON class_sections USING GIN ((lower(class_section_name)) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_section_slug_trgm_alive
  ON class_sections USING GIN ((lower(class_section_slug)) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

-- 6) JSONB indexes (generic + targeted lookups)
CREATE INDEX IF NOT EXISTS gin_csec_class_snapshot   ON class_sections USING GIN (class_section_class_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_parent_snapshot  ON class_sections USING GIN (class_section_parent_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_teacher_snapshot ON class_sections USING GIN (class_section_teacher_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_asst_snapshot    ON class_sections USING GIN (class_section_assistant_teacher_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_leader_snapshot  ON class_sections USING GIN (class_section_leader_student_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_room_snapshot    ON class_sections USING GIN (class_section_room_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_term_snapshot    ON class_sections USING GIN (class_section_term_snapshot jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_csst             ON class_sections USING GIN (class_sections_csst jsonb_path_ops);
CREATE INDEX IF NOT EXISTS gin_csec_features         ON class_sections USING GIN (class_section_features jsonb_path_ops);

--    b) B-Tree expression indexes untuk query umum (slug/name)
CREATE INDEX IF NOT EXISTS ix_csec_class_slug_snap
  ON class_sections (lower(class_section_class_slug_snap))
  WHERE class_section_deleted_at IS NULL AND class_section_class_slug_snap IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_csec_parent_slug_snap
  ON class_sections (lower(class_section_parent_slug_snap))
  WHERE class_section_deleted_at IS NULL AND class_section_parent_slug_snap IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_csec_term_slug_snap
  ON class_sections (lower(class_section_term_slug_snap))
  WHERE class_section_deleted_at IS NULL AND class_section_term_slug_snap IS NOT NULL;

-- 7) URL group lookup (kalau dipakai)
CREATE INDEX IF NOT EXISTS idx_section_group_url_alive
  ON class_sections (class_section_group_url)
  WHERE class_section_deleted_at IS NULL AND class_section_group_url IS NOT NULL;

-- 8) BRIN untuk time-range queries (hemat storage, scan cepat waktu)
CREATE INDEX IF NOT EXISTS brin_section_created_at
  ON class_sections USING BRIN (class_section_created_at);

-- 9) Purge kandidat image lama (due)
CREATE INDEX IF NOT EXISTS idx_section_image_purge_due
  ON class_sections (class_section_image_delete_pending_until)
  WHERE class_section_image_object_key_old IS NOT NULL;

-- 10) Lookup dasar per tenant (fallback umum)
CREATE INDEX IF NOT EXISTS idx_section_school
  ON class_sections (class_section_school_id);

-- 11) Query cepat berdasarkan mode enrolment CSST
CREATE INDEX IF NOT EXISTS ix_section_csst_enrollment_mode_alive
  ON class_sections (class_section_csst_enrollment_mode, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

COMMIT;
