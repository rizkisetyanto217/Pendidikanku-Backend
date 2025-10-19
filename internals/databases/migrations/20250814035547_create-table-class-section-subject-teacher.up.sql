-- +migrate Up
-- =========================================================
-- EXTENSIONS (safe to repeat)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;



-- =========================================================
-- PREREQUISITES: UNIQUE INDEX untuk target FK komposit
-- (harus ada agar FK (id, masjid_id) valid)
-- =========================================================

-- class_sections(id, masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sections_id_tenant
  ON class_sections (class_section_id, class_section_masjid_id);

-- class_subjects(id, masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subjects_id_tenant
  ON class_subjects (class_subject_id, class_subject_masjid_id);

-- masjid_teachers(id, masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_teachers_id_tenant
  ON masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id);

-- class_rooms(id, masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_id_tenant
  ON class_rooms (class_room_id, class_room_masjid_id);


BEGIN;


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
-- TABLE: class_section_subject_teachers (CSST)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teacher_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_section_subject_teacher_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- target penugasan
  class_section_subject_teacher_section_id UUID NOT NULL,
  class_section_subject_teacher_class_subject_id UUID NOT NULL,
  class_section_subject_teacher_teacher_id UUID NOT NULL,

  class_section_subject_teacher_name VARCHAR(160) NOT NULL,

  -- identitas & fasilitas
  class_section_subject_teacher_slug VARCHAR(160),
  class_section_subject_teacher_description TEXT,
  class_section_subject_teacher_room_id UUID,
  class_section_subject_teacher_group_url TEXT,

  -- agregat & kapasitas
  class_section_subject_teacher_total_attendance INT NOT NULL DEFAULT 0,
  class_section_subject_teacher_capacity INT,
  class_section_subject_teacher_enrolled_count INT NOT NULL DEFAULT 0,

  -- delivery mode (mengikuti penamaan kamu sebelumnya)
  class_sections_subject_teacher_delivery_mode class_delivery_mode_enum NOT NULL DEFAULT 'offline',

  -- =======================
  -- SNAPSHOTS (JSONB)
  -- =======================

  -- Room & People (existing)
  class_section_subject_teacher_room_snapshot JSONB,
  class_section_subject_teacher_teacher_snapshot JSONB,
  class_section_subject_teacher_assistant_teacher_snapshot JSONB,
  class_section_subject_teacher_books_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- Class Subject (baru, langsung di tabel)
  -- Kunci JSON yang diharapkan: name, code, slug, url
  class_section_subject_teacher_class_subject_snapshot JSONB,

  -- =======================
  -- GENERATED COLUMNS
  -- =======================

  -- Room
  class_section_subject_teacher_room_name_snap     TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_room_snapshot->>'name')) STORED,
  class_section_subject_teacher_room_slug_snap     TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_room_snapshot->>'slug')) STORED,
  class_section_subject_teacher_room_location_snap TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_room_snapshot->>'location')) STORED,

  -- People
  class_section_subject_teacher_teacher_name_snap TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_teacher_snapshot->>'name')) STORED,
  class_section_subject_teacher_assistant_teacher_name_snap TEXT GENERATED ALWAYS AS ((class_section_subject_teacher_assistant_teacher_snapshot->>'name')) STORED,

  -- Class Subject (generated dari snapshot)
  class_section_subject_teacher_class_subject_name_snap TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_snapshot->>'name')) STORED,
  class_section_subject_teacher_class_subject_code_snap TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_snapshot->>'code')) STORED,
  class_section_subject_teacher_class_subject_slug_snap TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_snapshot->>'slug')) STORED,
  class_section_subject_teacher_class_subject_url_snap  TEXT
    GENERATED ALWAYS AS ((class_section_subject_teacher_class_subject_snapshot->>'url')) STORED,

  -- status & audit
  class_section_subject_teacher_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_deleted_at TIMESTAMPTZ,

  -- =============== CHECKS ===============
  CONSTRAINT ck_csst_capacity_nonneg
    CHECK (class_section_subject_teacher_capacity IS NULL OR class_section_subject_teacher_capacity >= 0),
  CONSTRAINT ck_csst_enrolled_nonneg
    CHECK (class_section_subject_teacher_enrolled_count >= 0),

  -- =============== TENANT-SAFE FKs ===============
  CONSTRAINT fk_csst_section_tenant FOREIGN KEY (
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_masjid_id
  ) REFERENCES class_sections (class_section_id, class_section_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_class_subject_tenant FOREIGN KEY (
    class_section_subject_teacher_class_subject_id,
    class_section_subject_teacher_masjid_id
  ) REFERENCES class_subjects (class_subject_id, class_subject_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_teacher_tenant FOREIGN KEY (
    class_section_subject_teacher_teacher_id,
    class_section_subject_teacher_masjid_id
  ) REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_csst_room_tenant FOREIGN KEY (
    class_section_subject_teacher_room_id,
    class_section_subject_teacher_masjid_id
  ) REFERENCES class_rooms (class_room_id, class_room_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =========================================================
-- UNIQUE & INDEXES (idempotent)
-- =========================================================

-- PK + tenant (proteksi akses lintas tenant by habit)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_tenant
  ON class_section_subject_teachers (class_section_subject_teacher_id, class_section_subject_teacher_masjid_id);

-- Unik kombinasi penugasan "alive"
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_class_subject_id,
    class_section_subject_teacher_teacher_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL;

-- Satu aktif per section+class_subject (alive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_one_active_per_section_subject_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_class_subject_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_is_active = TRUE;

-- Slug unik per tenant (alive, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_slug_per_tenant_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    lower(class_section_subject_teacher_slug)
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_slug IS NOT NULL;

-- Search & perf indexes
CREATE INDEX IF NOT EXISTS idx_csst_masjid_alive
  ON class_section_subject_teachers (class_section_subject_teacher_masjid_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_section_alive
  ON class_section_subject_teachers (class_section_subject_teacher_section_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_subject_alive
  ON class_section_subject_teachers (class_section_subject_teacher_class_subject_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_teacher_alive
  ON class_section_subject_teachers (class_section_subject_teacher_teacher_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_room_alive
  ON class_section_subject_teachers (class_section_subject_teacher_room_id)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

-- Trigram search untuk slug (butuh pg_trgm)
CREATE INDEX IF NOT EXISTS gin_csst_slug_trgm_alive
  ON class_section_subject_teachers
  USING GIN (lower(class_section_subject_teacher_slug) gin_trgm_ops)
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_slug IS NOT NULL;

-- BRIN untuk waktu create
CREATE INDEX IF NOT EXISTS brin_csst_created_at
  ON class_section_subject_teachers USING BRIN (class_section_subject_teacher_created_at);

-- Agregat
CREATE INDEX IF NOT EXISTS idx_csst_capacity_alive
  ON class_section_subject_teachers (class_section_subject_teacher_capacity)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_enrolled_count_alive
  ON class_section_subject_teachers (class_section_subject_teacher_enrolled_count)
  WHERE class_section_subject_teacher_deleted_at IS NULL;

-- ==============================
-- Index pendukung snapshot class_subject
-- ==============================
CREATE INDEX IF NOT EXISTS idx_csst_class_subject_name_snap_alive
  ON class_section_subject_teachers (LOWER(class_section_subject_teacher_class_subject_name_snap))
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_subject_code_snap_alive
  ON class_section_subject_teachers (LOWER(class_section_subject_teacher_class_subject_code_snap))
  WHERE class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_csst_class_subject_slug_snap_trgm_alive
  ON class_section_subject_teachers
  USING GIN (LOWER(class_section_subject_teacher_class_subject_slug_snap) gin_trgm_ops)
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_class_subject_slug_snap IS NOT NULL;

COMMIT;



-- +migrate Up
BEGIN;

-- =========================================================
-- TABLE: user_class_section_subject_teachers (UC SST)
-- Fokus: mapping siswa ↔ guru (CSST) + nilai + history intervensi
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_section_subject_teachers (
  user_class_section_subject_teacher_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_section_subject_teacher_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Anchor hubungan
  user_class_section_subject_teacher_student_id UUID NOT NULL,
  user_class_section_subject_teacher_csst_id    UUID NOT NULL,

  -- Status mapping
  user_class_section_subject_teacher_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  user_class_section_subject_teacher_from DATE,
  user_class_section_subject_teacher_to   DATE,

  -- Nilai terbaru (untuk akses cepat)
  user_class_section_subject_teacher_score_total     NUMERIC(6,2),
  user_class_section_subject_teacher_score_max_total NUMERIC(6,2) DEFAULT 100,
  user_class_section_subject_teacher_score_percent   NUMERIC(5,2)
    GENERATED ALWAYS AS (
      CASE
        WHEN user_class_section_subject_teacher_score_total IS NULL
          OR user_class_section_subject_teacher_score_max_total IS NULL
          OR user_class_section_subject_teacher_score_max_total = 0
        THEN NULL
        ELSE ROUND(
          (user_class_section_subject_teacher_score_total
           / user_class_section_subject_teacher_score_max_total) * 100.0, 2
        )
      END
    ) STORED,
  user_class_section_subject_teacher_grade_letter VARCHAR(8),
  user_class_section_subject_teacher_grade_point  NUMERIC(3,2),
  user_class_section_subject_teacher_is_passed    BOOLEAN,

  -- Snapshot users_profile (per siswa saat enrol ke CSST)
  user_class_section_subject_teacher_user_profile_name_snapshot                VARCHAR(80),
  user_class_section_subject_teacher_user_profile_avatar_url_snapshot          VARCHAR(255),
  user_class_section_subject_teacher_user_profile_whatsapp_url_snapshot        VARCHAR(50),
  user_class_section_subject_teacher_user_profile_parent_name_snapshot         VARCHAR(80),
  user_class_section_subject_teacher_user_profile_parent_whatsapp_url_snapshot VARCHAR(50),

  -- Riwayat intervensi/remedial (JSONB append-only)
  user_class_section_subject_teacher_edits_history JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_ucsst_edits_history_is_array CHECK (
    jsonb_typeof(user_class_section_subject_teacher_edits_history) = 'array'
  ),

  -- Admin & meta
  user_class_section_subject_teacher_slug VARCHAR(160),
  user_class_section_subject_teacher_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_ucsst_meta_is_object CHECK (
    jsonb_typeof(user_class_section_subject_teacher_meta) = 'object'
  ),

  -- Audit & soft delete
  user_class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_deleted_at TIMESTAMPTZ,

  -- ===== Tenant-safe FKs =====
  CONSTRAINT fk_ucsst_student_tenant FOREIGN KEY (
    user_class_section_subject_teacher_student_id,
    user_class_section_subject_teacher_masjid_id
  ) REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_ucsst_csst_tenant FOREIGN KEY (
    user_class_section_subject_teacher_csst_id,
    user_class_section_subject_teacher_masjid_id
  ) REFERENCES class_section_subject_teachers (
        class_section_subject_teacher_id,
        class_section_subject_teacher_masjid_id
      )
    ON UPDATE CASCADE ON DELETE RESTRICT
);


-- =========================================================
-- INDEXES
-- =========================================================

-- Pair unik id+tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_id_tenant
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_id,
    user_class_section_subject_teacher_masjid_id
  );

-- Satu mapping aktif per (student × CSST)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_one_active_per_student_csst_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    user_class_section_subject_teacher_student_id,
    user_class_section_subject_teacher_csst_id
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL
    AND user_class_section_subject_teacher_is_active = TRUE;

-- Optional slug per tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_slug_per_tenant_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    lower(user_class_section_subject_teacher_slug)
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL
    AND user_class_section_subject_teacher_slug IS NOT NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_ucsst_masjid_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_masjid_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_student_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_student_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_csst_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_csst_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_active_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_is_active)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_ucsst_created_at
  ON user_class_section_subject_teachers USING BRIN (user_class_section_subject_teacher_created_at);

-- GIN optional untuk query di history JSONB
CREATE INDEX IF NOT EXISTS gin_ucsst_edits_history
  ON user_class_section_subject_teachers
  USING GIN (user_class_section_subject_teacher_edits_history jsonb_path_ops);

COMMIT;
