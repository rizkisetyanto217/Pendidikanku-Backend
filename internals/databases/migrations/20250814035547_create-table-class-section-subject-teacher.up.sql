-- =========================================================
-- TABLE: class_section_subject_teachers
-- =========================================================

-- (opsional, bila belum ada di migrasi awal)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  -- PK
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & konteks
  class_section_subject_teachers_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_section_subject_teachers_section_id       UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  class_section_subject_teachers_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

  -- ✅ refer ke masjid_teachers.masjid_teacher_id (BUKAN users.id)
  class_section_subject_teachers_teacher_id       UUID NOT NULL
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE RESTRICT,

  -- Status & audit
  class_section_subject_teachers_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- Indexes (soft-delete aware + akses cepat)
-- =========================================================

-- Cegah duplikasi mapping guru pada kombinasi (tenant × section × class_subject × teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_unique_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_class_subjects_id,
    class_section_subject_teachers_teacher_id
  )
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- (Opsional) Hanya boleh 1 guru AKTIF per (tenant × section × class_subject)
-- Jika kamu izinkan co-teaching aktif, hapus index ini.
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_one_active_per_section_subject_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_class_subjects_id
  )
  WHERE class_section_subject_teachers_deleted_at IS NULL
    AND class_section_subject_teachers_is_active = TRUE;

-- Lookups umum
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

CREATE INDEX IF NOT EXISTS idx_csst_active_alive
  ON class_section_subject_teachers (class_section_subject_teachers_is_active)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- Scan waktu besar (opsional)
CREATE INDEX IF NOT EXISTS brin_csst_created_at
  ON class_section_subject_teachers USING BRIN (class_section_subject_teachers_created_at);




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
