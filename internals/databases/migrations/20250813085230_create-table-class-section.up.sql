BEGIN;

-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE & fuzzy)
CREATE EXTENSION IF NOT EXISTS btree_gist; -- optional untuk kombinasi tertentu

-- =========================================================
-- PASTIKAN academic_terms punya UNIQUE (id, masjid_id)
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
      UNIQUE (academic_term_id, academic_term_masjid_id);
  END IF;
END$$;

-- =========================================================
-- TABLE: class_sections — snapshots & hints
-- =========================================================
CREATE TABLE IF NOT EXISTS class_sections (
  class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi inti
  class_section_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_section_class_id             UUID NOT NULL,
  class_section_teacher_id           UUID,
  class_section_assistant_teacher_id UUID,
  class_section_class_room_id        UUID,
  class_section_leader_student_id    UUID,  -- ketua kelas (masjid_students)

  -- Identitas
  class_section_slug  VARCHAR(160) NOT NULL,
  class_section_name  VARCHAR(100) NOT NULL,
  class_section_code  VARCHAR(50),

  -- Jadwal sederhana
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

  class_section_teacher_code_hash BYTEA,
  class_section_teacher_code_set_at TIMESTAMPTZ,
  class_section_student_code_hash BYTEA,
  class_section_student_code_set_at TIMESTAMPTZ,

  -- Status & audit
  class_section_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_deleted_at TIMESTAMPTZ,

  -- ===================== SNAPSHOTS & HINTS =====================

  -- snapshots dari class & parent/teacher/leader
  class_section_class_slug_snapshot             VARCHAR(160),
  class_section_parent_name_snapshot            VARCHAR(120),
  class_section_teacher_name_snapshot           VARCHAR(120),
  class_section_assistant_teacher_name_snapshot VARCHAR(120),
  class_section_leader_student_name_snapshot    VARCHAR(120),

  -- kontak (snapshot)
  class_section_teacher_contact_phone_snapshot           VARCHAR(20),
  class_section_assistant_teacher_contact_phone_snapshot VARCHAR(20),
  class_section_leader_student_contact_phone_snapshot    VARCHAR(20),

  -- ROOM snapshots
  class_section_room_name_snapshot      VARCHAR(120),
  class_section_room_slug_snapshot      VARCHAR(160),
  class_section_room_location_snapshot  VARCHAR(160),

  -- housekeeping snapshot
  class_section_snapshot_updated_at TIMESTAMPTZ,

  -- TERM (lean snapshots)
  class_section_term_id UUID,
  class_section_term_name_snapshot       VARCHAR(120),
  class_section_term_slug_snapshot       VARCHAR(160),
  class_section_term_year_label_snapshot VARCHAR(20),

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

  -- ==================== FK KOMPOSIT (tenant-safe) ===============
  CONSTRAINT fk_section_class_same_masjid
    FOREIGN KEY (class_section_class_id, class_section_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_section_teacher_same_masjid
    FOREIGN KEY (class_section_teacher_id, class_section_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_assistant_teacher_same_masjid
    FOREIGN KEY (class_section_assistant_teacher_id, class_section_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_leader_student_same_masjid
    FOREIGN KEY (class_section_leader_student_id, class_section_masjid_id)
    REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_csec_term_same_masjid
    FOREIGN KEY (class_section_term_id, class_section_masjid_id)
    REFERENCES academic_terms (academic_term_id, academic_term_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Pair unik untuk join multi-tenant aman
  CONSTRAINT uq_class_section_id_masjid
    UNIQUE (class_section_id, class_section_masjid_id)
);

-- =========================================================
-- INDEXING & OPTIMIZATION
-- =========================================================

-- 1) Unik slug per masjid (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_slug_per_masjid_alive
  ON class_sections (class_section_masjid_id, lower(class_section_slug))
  WHERE class_section_deleted_at IS NULL;

-- 2) (Opsional) Unik code per class (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_code_per_class_alive
  ON class_sections (class_section_class_id, lower(class_section_code))
  WHERE class_section_deleted_at IS NULL AND class_section_code IS NOT NULL;

-- 3) FK-friendly composites (untuk join cepat; hanya baris hidup)
CREATE INDEX IF NOT EXISTS idx_section_class_masjid_alive
  ON class_sections (class_section_class_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_teacher_masjid_alive
  ON class_sections (class_section_teacher_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_assistant_teacher_masjid_alive
  ON class_sections (class_section_assistant_teacher_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_room_masjid_alive
  ON class_sections (class_section_class_room_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_leader_student_masjid_alive
  ON class_sections (class_section_leader_student_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_term_masjid_alive
  ON class_sections (class_section_term_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

-- 4) Listing umum (tenanted)
CREATE INDEX IF NOT EXISTS ix_section_masjid_active_created
  ON class_sections (class_section_masjid_id, class_section_is_active, class_section_created_at DESC)
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

-- (Opsional) filter cepat by term slug snapshot
CREATE INDEX IF NOT EXISTS ix_csec_term_slug_snap
  ON class_sections (lower(class_section_term_slug_snapshot))
  WHERE class_section_deleted_at IS NULL AND class_section_term_slug_snapshot IS NOT NULL;

-- 6) URL group lookup (kalau dipakai)
CREATE INDEX IF NOT EXISTS idx_section_group_url_alive
  ON class_sections (class_section_group_url)
  WHERE class_section_deleted_at IS NULL AND class_section_group_url IS NOT NULL;

-- 7) BRIN untuk time-range queries (hemat storage, scan cepat waktu)
CREATE INDEX IF NOT EXISTS brin_section_created_at
  ON class_sections USING BRIN (class_section_created_at);

-- 8) Purge kandidat image lama (due)
CREATE INDEX IF NOT EXISTS idx_section_image_purge_due
  ON class_sections (class_section_image_delete_pending_until)
  WHERE class_section_image_object_key_old IS NOT NULL;

-- 9) Lookup dasar per tenant (fallback umum)
CREATE INDEX IF NOT EXISTS idx_section_masjid
  ON class_sections (class_section_masjid_id);

COMMIT;
