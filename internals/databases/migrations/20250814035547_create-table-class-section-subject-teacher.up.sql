-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =========================================================
-- TABLE: class_section_subject_teachers
-- =========================================================
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teacher_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_section_subject_teacher_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_section_subject_teacher_section_id UUID NOT NULL,
  class_section_subject_teacher_class_subject_id UUID NOT NULL,
  class_section_subject_teacher_teacher_id UUID NOT NULL,

  class_section_subject_teacher_slug VARCHAR(160),
  class_section_subject_teacher_description TEXT,
  class_section_subject_teacher_room_id UUID,
  class_section_subject_teacher_group_url TEXT,

  class_section_subject_teacher_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teacher_deleted_at TIMESTAMPTZ,

  -- Tenant-safe FKs
  CONSTRAINT fk_csst_section_tenant FOREIGN KEY (class_section_subject_teacher_section_id, class_section_subject_teacher_masjid_id)
    REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_class_subject_tenant FOREIGN KEY (class_section_subject_teacher_class_subject_id, class_section_subject_teacher_masjid_id)
    REFERENCES class_subjects (class_subject_id, class_subjects_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_csst_teacher_tenant FOREIGN KEY (class_section_subject_teacher_teacher_id, class_section_subject_teacher_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  CONSTRAINT fk_csst_room_tenant FOREIGN KEY (class_section_subject_teacher_room_id, class_section_subject_teacher_masjid_id)
    REFERENCES class_rooms (class_room_id, class_rooms_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- Pair unik id+tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_tenant
  ON class_section_subject_teachers (class_section_subject_teacher_id, class_section_subject_teacher_masjid_id);

-- Unik kombinasi guru per (tenant×section×subject×teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_class_subject_id,
    class_section_subject_teacher_teacher_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL;

-- Opsional: hanya 1 guru aktif per section×subject
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_one_active_per_section_subject_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_class_subject_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_is_active = TRUE;

-- Index umum
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

-- Feed cepat
CREATE INDEX IF NOT EXISTS ix_csst_by_teacher_subject_active
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_teacher_id,
    class_section_subject_teacher_class_subject_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_csst_by_section_teacher_active
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    class_section_subject_teacher_section_id,
    class_section_subject_teacher_teacher_id
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_is_active = TRUE;

-- Slug
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_slug_per_tenant_alive
  ON class_section_subject_teachers (
    class_section_subject_teacher_masjid_id,
    lower(class_section_subject_teacher_slug)
  )
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_csst_slug_trgm_alive
  ON class_section_subject_teachers
  USING GIN (lower(class_section_subject_teacher_slug) gin_trgm_ops)
  WHERE class_section_subject_teacher_deleted_at IS NULL
    AND class_section_subject_teacher_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_csst_created_at
  ON class_section_subject_teachers USING BRIN (class_section_subject_teacher_created_at);



-- =========================================================
-- TABLE: user_class_section_subject_teachers
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_section_subject_teachers (
  user_class_section_subject_teacher_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_section_subject_teacher_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_section_subject_teacher_section_id UUID NOT NULL,
  user_class_section_subject_teacher_class_subject_id UUID NOT NULL,
  user_class_section_subject_teacher_teacher_id UUID NOT NULL,

  user_class_section_subject_teacher_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  user_class_section_subject_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_subject_teacher_deleted_at TIMESTAMPTZ,

  CONSTRAINT fk_ucsst_section_tenant FOREIGN KEY (user_class_section_subject_teacher_section_id, user_class_section_subject_teacher_masjid_id)
    REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_ucsst_class_subject_tenant FOREIGN KEY (user_class_section_subject_teacher_class_subject_id, user_class_section_subject_teacher_masjid_id)
    REFERENCES class_subjects (class_subject_id, class_subjects_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_ucsst_teacher_tenant FOREIGN KEY (user_class_section_subject_teacher_teacher_id, user_class_section_subject_teacher_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);

-- Pair unik id+tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_id_tenant
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_id, user_class_section_subject_teacher_masjid_id);

-- Unik kombinasi guru per (tenant×section×subject×teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_unique_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    user_class_section_subject_teacher_section_id,
    user_class_section_subject_teacher_class_subject_id,
    user_class_section_subject_teacher_teacher_id
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

-- Opsional: hanya 1 guru aktif per section×subject
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsst_one_active_per_section_subject_alive
  ON user_class_section_subject_teachers (
    user_class_section_subject_teacher_masjid_id,
    user_class_section_subject_teacher_section_id,
    user_class_section_subject_teacher_class_subject_id
  )
  WHERE user_class_section_subject_teacher_deleted_at IS NULL
    AND user_class_section_subject_teacher_is_active = TRUE;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_ucsst_masjid_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_masjid_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_section_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_section_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_class_subject_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_class_subject_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_teacher_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_teacher_id)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsst_active_alive
  ON user_class_section_subject_teachers (user_class_section_subject_teacher_is_active)
  WHERE user_class_section_subject_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_ucsst_created_at
  ON user_class_section_subject_teachers USING BRIN (user_class_section_subject_teacher_created_at);