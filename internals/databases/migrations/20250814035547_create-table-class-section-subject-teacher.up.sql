-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- TABLE: class_section_subject_teachers (CSST)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  -- PK
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & konteks
  class_section_subject_teachers_masjid_id         UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_section_subject_teachers_section_id        UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  class_section_subject_teachers_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

  -- refer ke masjid_teachers (BUKAN users.id)
  class_section_subject_teachers_teacher_id        UUID NOT NULL
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE RESTRICT,

  -- Optional slug
  class_section_subject_teachers_slug VARCHAR(160),

  class_section_subject_teachers_description       TEXT,

  -- Override ruangan (opsional; default dari class_sections.class_rooms_id)
  class_section_subject_teachers_room_id           UUID
    REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- Link grup (WA/Telegram/DS)
  class_section_subject_teachers_group_url         TEXT,

  -- Status & audit
  class_section_subject_teachers_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- CONSTRAINTS / UNIQUENESS (soft-delete aware)
-- =========================================================

-- Cegah duplikasi mapping guru per kombinasi (tenant × section × subject × teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_class_subjects_id,
    class_section_subject_teachers_teacher_id
  )
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- (Opsional kuat) Hanya 1 guru AKTIF per (tenant × section × subject)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_one_active_per_section_subject_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_class_subjects_id
  )
  WHERE class_section_subject_teachers_deleted_at IS NULL
    AND class_section_subject_teachers_is_active = TRUE;

-- (Opsional) Pair unik id + tenant (buat FK komposit di tempat lain)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_tenant
  ON class_section_subject_teachers (
    class_section_subject_teachers_id,
    class_section_subject_teachers_masjid_id
  );

-- =========================================================
-- INDEXES (akses cepat)
-- =========================================================

-- Lookup umum (soft-delete aware)
CREATE INDEX IF NOT EXISTS idx_csst_masjid_alive
  ON class_section_subject_teachers (class_section_subject_teachers_masjid_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_section_alive
  ON class_section_subject_teachers (class_section_subject_teachers_section_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_subjects_alive
  ON class_section_subject_teachers (class_section_subject_teachers_class_subjects_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_teacher_alive
  ON class_section_subject_teachers (class_section_subject_teachers_teacher_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- 🔥 Composite partial indexes (buat resolver publish/feed super cepat)
-- Jalur utama: per guru (opsional filter subject)
CREATE INDEX IF NOT EXISTS ix_csst_by_teacher_subject_active
ON class_section_subject_teachers (
  class_section_subject_teachers_masjid_id,
  class_section_subject_teachers_teacher_id,
  class_section_subject_teachers_class_subjects_id
)
WHERE class_section_subject_teachers_deleted_at IS NULL
  AND class_section_subject_teachers_is_active = TRUE;

-- Jalur alternatif: per section (rekap siapa gurunya sebuah kelas)
CREATE INDEX IF NOT EXISTS ix_csst_by_section_teacher_active
ON class_section_subject_teachers (
  class_section_subject_teachers_masjid_id,
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_teacher_id
)
WHERE class_section_subject_teachers_deleted_at IS NULL
  AND class_section_subject_teachers_is_active = TRUE;

-- (Opsional) index boolean biasanya low-selectivity, boleh di-skip
-- CREATE INDEX IF NOT EXISTS idx_csst_active_alive
--   ON class_section_subject_teachers (class_section_subject_teachers_is_active)
--   WHERE class_section_subject_teachers_deleted_at IS NULL;

-- Lookup ruangan override (opsional)
CREATE INDEX IF NOT EXISTS idx_csst_room_alive
  ON class_section_subject_teachers (class_section_subject_teachers_room_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- Scan waktu besar (opsional)
CREATE INDEX IF NOT EXISTS brin_csst_created_at
  ON class_section_subject_teachers USING BRIN (class_section_subject_teachers_created_at);

-- SLUG (unik per tenant; soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_slug_per_tenant_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_masjid_id,
    lower(class_section_subject_teachers_slug)
  )
  WHERE class_section_subject_teachers_deleted_at IS NULL
    AND class_section_subject_teachers_slug IS NOT NULL;

-- Pencarian cepat slug
CREATE INDEX IF NOT EXISTS gin_csst_slug_trgm_alive
  ON class_section_subject_teachers
  USING GIN (lower(class_section_subject_teachers_slug) gin_trgm_ops)
  WHERE class_section_subject_teachers_deleted_at IS NULL
    AND class_section_subject_teachers_slug IS NOT NULL;



-- ==== Path B: kalau tabel barunya belum ada, create ====
CREATE TABLE IF NOT EXISTS user_class_section_subject_teachers (
  user_class_section_subject_teacher_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_section_subject_teacher_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_section_subject_teacher_section_id       UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  user_class_section_subject_teacher_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

  user_class_section_subject_teacher_teacher_id       UUID NOT NULL
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE RESTRICT,

  user_class_section_subject_teacher_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  user_class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_deleted_at TIMESTAMPTZ
);

-- Uniq & indeks (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_unique_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    user_class_section_subject_teacher_section_id,
    user_class_section_subject_teacher_class_subjects_id,
    user_class_section_subject_teacher_teacher_id
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

-- Jika HANYA SATU guru aktif per (tenant×section×subject):
-- (hapus index ini jika co-teaching aktif diizinkan)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_one_active_per_section_subject_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    user_class_section_subject_teacher_section_id,
    user_class_section_subject_teacher_class_subjects_id
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL
    AND user_class_section_subject_teacher_is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_ucsst_masjid_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_masjid_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_section_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_section_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_class_subjects_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_class_subjects_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_teacher_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_teacher_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_active_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_is_active)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_ucsst_created_at
  ON user_class_section_subject_teachers USING BRIN (user_class_section_subject_teacher_created_at);
