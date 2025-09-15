-- =========================================================
-- UP Migration — User Notes (+ URLs)  — NO triggers/functions
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
  user_note_type_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  user_note_type_is_shared  BOOLEAN NOT NULL DEFAULT FALSE,

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
  user_note_labels   TEXT[] DEFAULT '{}',
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

-- ---------------------------------------------------------
-- Indexes (tanpa fungsi/trigger) — user_notes
-- ---------------------------------------------------------
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

CREATE INDEX IF NOT EXISTS idx_user_notes_student_created
  ON user_notes(user_note_student_id, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_class_scope_created
  ON user_notes(user_note_class_section_id, user_note_scope, user_note_created_at DESC)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_type_id
  ON user_notes(user_note_type_id)
  WHERE user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_labels_gin
  ON user_notes USING GIN (user_note_labels);

CREATE INDEX IF NOT EXISTS idx_user_notes_pinned_partial
  ON user_notes(user_note_masjid_id, user_note_created_at DESC)
  WHERE user_note_is_pinned = TRUE AND user_note_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_notes_due_date
  ON user_notes(user_note_masjid_id, user_note_due_date)
  WHERE user_note_deleted_at IS NULL AND user_note_due_date IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_user_notes_title_trgm
  ON user_notes USING GIN (user_note_title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS gin_user_notes_content_trgm
  ON user_notes USING GIN (user_note_content gin_trgm_ops);

-- =========================================================
-- 3) CHILD: user_note_urls (multi-lampiran/tautan per catatan)
--     - Mendukung file/link arbitrer (image/video/audio/document/link/other)
--     - 2-slot retensi: *_old + delete_pending_until
-- =========================================================
-- (Opsional) Enum lokal untuk kind — hindari konflik global
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_note_url_kind_enum') THEN
    CREATE TYPE user_note_url_kind_enum AS ENUM ('image','video','audio','document','link','other');
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS user_note_urls (
  user_note_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard (aman walau join langsung tanpa ikut parent)
  user_note_url_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- parent
  user_note_url_note_id UUID NOT NULL
    REFERENCES user_notes(user_note_id) ON DELETE CASCADE,

  -- informasi & klasifikasi
  user_note_url_kind      user_note_url_kind_enum NOT NULL DEFAULT 'link',
  user_note_url_mime      VARCHAR(120),               -- ex: image/webp, application/pdf
  user_note_url_tags      TEXT[] DEFAULT '{}',        -- tag bebas (opsional)
  user_note_url_title     VARCHAR(180),
  user_note_url_desc      TEXT,

  -- pointer file/link (utama)
  user_note_url           TEXT,                       -- bisa http(s) link atau signed url
  user_note_url_object_key TEXT,                      -- key di storage (Supabase/S3)

  -- 2-slot retensi (opsional)
  user_note_url_old            TEXT,
  user_note_url_object_key_old TEXT,
  user_note_url_delete_pending_until TIMESTAMPTZ,

  -- behaviour UI
  user_note_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  user_note_url_sort       INT NOT NULL DEFAULT 0,

  -- audit
  user_note_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_note_url_deleted_at TIMESTAMPTZ,

  -- konsistensi minimal: jika pakai object_key_old maka url_old harus ada (dan sebaliknya)
  CONSTRAINT chk_user_note_url_old_pair
    CHECK (
      (user_note_url_old IS NULL AND user_note_url_object_key_old IS NULL)
      OR (user_note_url_old IS NOT NULL AND user_note_url_object_key_old IS NOT NULL)
    )
);

-- ---------------------------------------------------------
-- Indexes — user_note_urls
-- ---------------------------------------------------------
-- Listing per note + prioritas/urutan
CREATE INDEX IF NOT EXISTS idx_user_note_urls_note_sort
  ON user_note_urls(user_note_url_note_id, user_note_url_sort, user_note_url_created_at DESC)
  WHERE user_note_url_deleted_at IS NULL;

-- Satu primary aktif per note (partial unique)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_note_urls_one_primary
  ON user_note_urls(user_note_url_note_id)
  WHERE user_note_url_is_primary = TRUE AND user_note_url_deleted_at IS NULL;

-- Query label/tags
CREATE INDEX IF NOT EXISTS gin_user_note_urls_tags
  ON user_note_urls USING GIN (user_note_url_tags)
  WHERE user_note_url_deleted_at IS NULL;

-- Pencarian URL/title (trigram)
CREATE INDEX IF NOT EXISTS gin_user_note_urls_url_trgm
  ON user_note_urls USING GIN (user_note_url gin_trgm_ops)
  WHERE user_note_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_user_note_urls_title_trgm
  ON user_note_urls USING GIN (user_note_url_title gin_trgm_ops)
  WHERE user_note_url_deleted_at IS NULL;

-- Lookup per masjid/kind
CREATE INDEX IF NOT EXISTS idx_user_note_urls_masjid_kind
  ON user_note_urls(user_note_url_masjid_id, user_note_url_kind)
  WHERE user_note_url_deleted_at IS NULL;

-- Arsip waktu (ringan)
CREATE INDEX IF NOT EXISTS brin_user_note_urls_created_at
  ON user_note_urls USING BRIN (user_note_url_created_at);

COMMIT;
