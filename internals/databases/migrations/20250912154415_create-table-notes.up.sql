-- =========================================================
-- UP Migration — User Notes (NO triggers/functions)
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- 1) MASTER: user_note_types
-- =========================================================
CREATE TABLE IF NOT EXISTS user_note_types (
  user_note_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_note_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_note_type_owner_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- opsional: jika owner adalah guru
  user_note_type_owner_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE CASCADE,

  user_note_type_code  VARCHAR(32)  NOT NULL,  -- unik per tenant + owner
  user_note_type_name  VARCHAR(80)  NOT NULL,
  user_note_type_color VARCHAR(16),
  user_note_type_sort  INT NOT NULL DEFAULT 0,
  user_note_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  user_note_type_is_shared BOOLEAN NOT NULL DEFAULT FALSE,

  user_note_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_type_deleted_at TIMESTAMPTZ,

  CONSTRAINT uq_note_type_per_tenant_owner_code
    UNIQUE (user_note_type_masjid_id, user_note_type_owner_user_id, user_note_type_code)
);

-- Indexes (lookup cepat)
CREATE INDEX IF NOT EXISTS idx_note_types_masjid
  ON user_note_types(user_note_type_masjid_id);

CREATE INDEX IF NOT EXISTS idx_note_types_owner_active_sort
  ON user_note_types(user_note_type_owner_user_id, user_note_type_is_active, user_note_type_sort)
  WHERE user_note_type_deleted_at IS NULL;

-- =========================================================
-- 2) TABEL UTAMA: user_notes
-- =========================================================
CREATE TABLE IF NOT EXISTS user_notes (
  user_note_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant scope
  user_note_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- author
  user_note_author_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_note_author_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE CASCADE,

  -- cakupan/target
  user_note_scope VARCHAR(16) NOT NULL DEFAULT 'student'
    CHECK (user_note_scope IN ('student','class','school','personal')),

  -- target (nullable sesuai scope)
  user_note_student_id UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- NOTE: kolom PK table ini biasanya "class_sections_id"
  user_note_class_section_id UUID
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- konten
  user_note_title   VARCHAR(150),
  user_note_content TEXT NOT NULL,

  -- tipe/tema (opsional)
  user_note_type_id UUID
    REFERENCES user_note_types(user_note_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- label & pengaturan
  user_note_labels  TEXT[] DEFAULT '{}',
  user_note_priority VARCHAR(8) NOT NULL DEFAULT 'med'
    CHECK (user_note_priority IN ('low','med','high')),
  user_note_is_pinned BOOLEAN NOT NULL DEFAULT FALSE,

  -- tenggat opsional
  user_note_due_date TIMESTAMPTZ,

  -- audit
  user_note_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_deleted_at TIMESTAMPTZ,

  -- Konsistensi target berdasarkan scope
  CONSTRAINT chk_user_notes_scope_targets CHECK (
    (user_note_scope = 'student'  AND user_note_student_id IS NOT NULL AND user_note_class_section_id IS NULL) OR
    (user_note_scope = 'class'    AND user_note_class_section_id IS NOT NULL AND user_note_student_id IS NULL) OR
    (user_note_scope = 'school'   AND user_note_student_id IS NULL AND user_note_class_section_id IS NULL) OR
    (user_note_scope = 'personal' AND user_note_student_id IS NULL AND user_note_class_section_id IS NULL)
  )
);

-- =========================================================
-- Indexes (tanpa fungsi/trigger)
-- =========================================================

-- scope dasar
CREATE INDEX IF NOT EXISTS idx_user_notes_masjid
  ON user_notes(user_note_masjid_id)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_author
  ON user_notes(user_note_author_user_id, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_author_teacher
  ON user_notes(user_note_author_teacher_id, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_scope
  ON user_notes(user_note_scope)
  WHERE user_note_deleted_at IS NULL;

-- lookup cepat per target
CREATE INDEX IF NOT EXISTS idx_user_notes_student_created
  ON user_notes(user_note_student_id, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_class_scope_created
  ON user_notes(user_note_class_section_id, user_note_scope, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

-- tipe & label
CREATE INDEX IF NOT EXISTS idx_user_notes_type_id
  ON user_notes(user_note_type_id)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_labels_gin
  ON user_notes USING GIN (user_note_labels);

-- pin & due date
CREATE INDEX IF NOT EXISTS idx_user_notes_pinned_partial
  ON user_notes(user_note_masjid_id, user_note_created_at DESC)
  WHERE user_note_is_pinned = TRUE AND user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_due_date
  ON user_notes(user_note_masjid_id, user_note_due_date)
  WHERE user_note_deleted_at IS NULL AND user_note_due_date IS NOT NULL;

-- pencarian cepat title/content (trigram)
CREATE INDEX IF NOT EXISTS gin_user_notes_title_trgm
  ON user_notes USING GIN (user_note_title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS gin_user_notes_content_trgm
  ON user_notes USING GIN (user_note_content gin_trgm_ops);

COMMIT;
