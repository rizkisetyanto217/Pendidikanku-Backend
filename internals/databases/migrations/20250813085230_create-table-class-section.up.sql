BEGIN;

-- Extensions yang diperlukan
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram index (ILIKE cepat)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- CLASS SECTIONS (fresh create) — plural table, singular columns
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

  -- Leader (ketua kelas) dari masjid_students
  class_section_leader_student_id    UUID,

  -- Identitas
  class_section_slug  VARCHAR(160) NOT NULL,
  class_section_name  VARCHAR(100) NOT NULL,
  class_section_code  VARCHAR(50),

  -- Jadwal simple
  class_section_schedule TEXT,

  -- Kapasitas & counter
  class_section_capacity       INT,
  class_section_total_students INT NOT NULL DEFAULT 0,

  -- Meeting / Group
  class_section_group_url  TEXT,

  -- Image (2-slot + retensi 30 hari)
  class_section_image_url                   TEXT,
  class_section_image_object_key            TEXT,
  class_section_image_url_old               TEXT,
  class_section_image_object_key_old        TEXT,
  class_section_image_delete_pending_until  TIMESTAMPTZ,

  -- Status & audit
  class_section_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_section_deleted_at TIMESTAMPTZ,

  -- ================= CHECK guards =================
  CONSTRAINT ck_section_capacity_nonneg
    CHECK (class_section_capacity IS NULL OR class_section_capacity >= 0),
  CONSTRAINT ck_section_total_nonneg
    CHECK (class_section_total_students >= 0),
  CONSTRAINT ck_section_total_le_capacity
    CHECK (class_section_capacity IS NULL OR class_section_total_students <= class_section_capacity),
  CONSTRAINT ck_section_group_url_scheme
    CHECK (class_section_group_url IS NULL OR class_section_group_url ~* '^(https?)://'),

  -- ================= FK KOMPOSIT (tenant-safe) =================
  -- Catatan: pasangan kolom yang direferensikan harus sudah UNIQUE/PK di tabel tujuan.
  CONSTRAINT fk_section_class_same_masjid
    FOREIGN KEY (class_section_class_id, class_section_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_section_teacher_same_masjid
    FOREIGN KEY (class_section_teacher_id, class_section_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- NEW: Assistant teacher (komposit ke masjid_teachers)
  CONSTRAINT fk_section_assistant_teacher_same_masjid
    FOREIGN KEY (class_section_assistant_teacher_id, class_section_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_section_leader_student_same_masjid
    FOREIGN KEY (class_section_leader_student_id, class_section_masjid_id)
    REFERENCES masjid_students (masjid_student_id, masjid_student_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Pair unik untuk join multi-tenant aman
  CONSTRAINT uq_class_section_id_masjid
    UNIQUE (class_section_id, class_section_masjid_id)
);

-- =========================================================
-- INDEXES (performant & minimal)
-- =========================================================

-- Unik: slug per masjid (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_slug_per_masjid_alive
  ON class_sections (class_section_masjid_id, LOWER(class_section_slug))
  WHERE class_section_deleted_at IS NULL;

-- (Opsional) Unik: code per class (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_section_code_per_class_alive
  ON class_sections (class_section_class_id, LOWER(class_section_code))
  WHERE class_section_deleted_at IS NULL AND class_section_code IS NOT NULL;

-- FK-friendly composite indexes
CREATE INDEX IF NOT EXISTS idx_section_class_masjid_alive
  ON class_sections (class_section_class_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_teacher_masjid_alive
  ON class_sections (class_section_teacher_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

-- NEW: assistant teacher lookup
CREATE INDEX IF NOT EXISTS idx_section_assistant_teacher_masjid_alive
  ON class_sections (class_section_assistant_teacher_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_section_room_masjid_alive
  ON class_sections (class_section_class_room_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

-- Leader student lookup
CREATE INDEX IF NOT EXISTS idx_section_leader_student_masjid_alive
  ON class_sections (class_section_leader_student_id, class_section_masjid_id)
  WHERE class_section_deleted_at IS NULL;

-- Lookup dasar
CREATE INDEX IF NOT EXISTS idx_section_masjid
  ON class_sections (class_section_masjid_id);

-- Listing umum
CREATE INDEX IF NOT EXISTS ix_section_masjid_active_created
  ON class_sections (class_section_masjid_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_class_active_created
  ON class_sections (class_section_class_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- (Opsional) Listing by teacher/assistant/room
CREATE INDEX IF NOT EXISTS ix_section_teacher_active_created
  ON class_sections (class_section_teacher_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_assistant_active_created
  ON class_sections (class_section_assistant_teacher_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_section_room_active_created
  ON class_sections (class_section_class_room_id, class_section_is_active, class_section_created_at DESC)
  WHERE class_section_deleted_at IS NULL;

-- Pencarian teks cepat (ILIKE) pada name/slug
CREATE INDEX IF NOT EXISTS gin_section_name_trgm_alive
  ON class_sections USING GIN (LOWER(class_section_name) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_section_slug_trgm_alive
  ON class_sections USING GIN (LOWER(class_section_slug) gin_trgm_ops)
  WHERE class_section_deleted_at IS NULL;

-- URL group lookup (opsional)
CREATE INDEX IF NOT EXISTS idx_section_group_url_alive
  ON class_sections (class_section_group_url)
  WHERE class_section_deleted_at IS NULL AND class_section_group_url IS NOT NULL;

-- BRIN waktu (hemat storage)
CREATE INDEX IF NOT EXISTS brin_section_created_at
  ON class_sections USING BRIN (class_section_created_at);

-- Purge kandidat image lama (due)
CREATE INDEX IF NOT EXISTS idx_section_image_purge_due
  ON class_sections (class_section_image_delete_pending_until)
  WHERE class_section_image_object_key_old IS NOT NULL;

COMMIT;