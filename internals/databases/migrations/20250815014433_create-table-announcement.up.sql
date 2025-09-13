-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- announcement_themes
-- =========================================================
CREATE TABLE IF NOT EXISTS announcement_themes (
  announcement_themes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_themes_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_themes_name VARCHAR(80)  NOT NULL,
  announcement_themes_slug VARCHAR(120) NOT NULL,
  announcement_themes_color VARCHAR(20),
  announcement_themes_description TEXT,
  announcement_themes_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  announcement_themes_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_deleted_at  TIMESTAMPTZ,

  CONSTRAINT ck_announcement_themes_slug
    CHECK (announcement_themes_slug ~ '^[a-z0-9-]+$')
);

-- Unik per tenant (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_name_live
  ON announcement_themes (announcement_themes_masjid_id, lower(announcement_themes_name))
  WHERE announcement_themes_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_slug_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_slug)
  WHERE announcement_themes_deleted_at IS NULL;

-- Bantu query (live)
CREATE INDEX IF NOT EXISTS ix_announcement_themes_tenant_active_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_is_active)
  WHERE announcement_themes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_themes_name_trgm_live
  ON announcement_themes USING GIN (announcement_themes_name gin_trgm_ops)
  WHERE announcement_themes_deleted_at IS NULL;

-- ========= Tenant-safe composite UNIQUE (simple & safe) =========
DO $$
DECLARE
  has_constraint boolean;
  has_index boolean;
BEGIN
  -- 1) Sudah ada constraint?
  SELECT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'announcement_themes'::regclass
      AND conname  = 'uq_announcement_themes_id_tenant'
  ) INTO has_constraint;

  IF NOT has_constraint THEN
    -- 2) Apakah ada index lama bernama uq_announcement_themes_id_tenant ?
    SELECT EXISTS (
      SELECT 1
      FROM pg_class
      WHERE relname = 'uq_announcement_themes_id_tenant'
        AND relkind = 'i'   -- index
    ) INTO has_index;

    IF has_index THEN
      -- 3) Attach index lama sebagai UNIQUE constraint
      EXECUTE '
        ALTER TABLE announcement_themes
          ADD CONSTRAINT uq_announcement_themes_id_tenant
          UNIQUE USING INDEX uq_announcement_themes_id_tenant
      ';
    ELSE
      -- 4) Buat constraint baru (Postgres auto-bikin index)
      ALTER TABLE announcement_themes
        ADD CONSTRAINT uq_announcement_themes_id_tenant
        UNIQUE (announcement_themes_id, announcement_themes_masjid_id);
    END IF;
  END IF;
END$$;

-- =========================================================
-- Kebutuhan UNIQUE komposit di tabel lain (untuk FK tenant-safe)
-- =========================================================

-- class_sections: butuh UNIQUE (class_sections_id, class_sections_masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_masjid
      UNIQUE (class_sections_id, class_sections_masjid_id);
  END IF;
END$$;

-- masjid_teachers: untuk FK komposit (teacher_id, masjid_id) → harus unik
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='uq_masjid_teachers_id_tenant'
  ) THEN
    ALTER TABLE masjid_teachers
      ADD CONSTRAINT uq_masjid_teachers_id_tenant
      UNIQUE (masjid_teacher_id, masjid_teacher_masjid_id);
  END IF;
END$$;


-- =========================================================
-- announcements
-- =========================================================
CREATE TABLE IF NOT EXISTS announcements (
  announcement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- GANTI: sumber pembuat sekarang teacher_id, bukan user_id
  announcement_created_by_teacher_id UUID NULL,

  -- target section (NULL = global)
  announcement_class_section_id UUID NULL,

  -- tema (tenant-safe via composite FK)
  announcement_theme_id UUID NULL,

  announcement_title   VARCHAR(200) NOT NULL,
  announcement_date    DATE NOT NULL,
  announcement_content TEXT NOT NULL,

  announcement_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  announcement_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_deleted_at TIMESTAMPTZ,

  -- FTS
  announcement_search tsvector
    GENERATED ALWAYS AS (
      setweight(to_tsvector('simple', coalesce(announcement_title,   '')), 'A') ||
      setweight(to_tsvector('simple', coalesce(announcement_content, '')), 'B')
    ) STORED
);

-- Pastikan kolom teacher_id ada
ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS announcement_created_by_teacher_id UUID;

-- Backfill dari user_id (jika ada)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='announcements'
      AND column_name='announcement_created_by_user_id'
  ) THEN
    WITH mapped AS (
      SELECT a.announcement_id, mt.masjid_teacher_id
      FROM announcements a
      JOIN LATERAL (
        SELECT masjid_teacher_id
        FROM masjid_teachers
        WHERE masjid_teacher_masjid_id = a.announcement_masjid_id
          AND masjid_teacher_user_id   = a.announcement_created_by_user_id
          AND masjid_teacher_deleted_at IS NULL
        ORDER BY masjid_teacher_created_at ASC
        LIMIT 1
      ) mt ON TRUE
      WHERE a.announcement_created_by_teacher_id IS NULL
        AND a.announcement_created_by_user_id    IS NOT NULL
    )
    UPDATE announcements a
       SET announcement_created_by_teacher_id = m.masjid_teacher_id
      FROM mapped m
     WHERE a.announcement_id = m.announcement_id;
  END IF;
END$$;


-- Tenant-safe FKs
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_ann_created_by_teacher_same_tenant') THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_created_by_teacher_same_tenant
      FOREIGN KEY (announcement_created_by_teacher_id, announcement_masjid_id)
      REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_ann_theme_same_tenant') THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_theme_same_tenant
      FOREIGN KEY (announcement_theme_id, announcement_masjid_id)
      REFERENCES announcement_themes (announcement_themes_id, announcement_themes_masjid_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Drop FK lama ke class_sections (single-col) kalau ada
DO $$
DECLARE r RECORD;
BEGIN
  FOR r IN
    SELECT conname
    FROM pg_constraint
    WHERE conrelid='announcements'::regclass
      AND contype='f'
      AND pg_get_constraintdef(oid) ILIKE '%(announcement_class_section_id)%class_sections%'
      AND pg_get_constraintdef(oid) NOT ILIKE '%(announcement_class_section_id, announcement_masjid_id)%'
  LOOP
    EXECUTE format('ALTER TABLE announcements DROP CONSTRAINT %I', r.conname);
  END LOOP;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_ann_section_same_tenant') THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_section_same_tenant
      FOREIGN KEY (announcement_class_section_id, announcement_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_id_tenant
  ON announcements (announcement_id, announcement_masjid_id);

CREATE INDEX IF NOT EXISTS ix_announcements_tenant_date_live
  ON announcements (announcement_masjid_id, announcement_date DESC)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_theme_live
  ON announcements (announcement_theme_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_section_live
  ON announcements (announcement_class_section_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

DROP INDEX IF EXISTS ix_announcements_created_by_live;
CREATE INDEX IF NOT EXISTS ix_announcements_created_by_teacher_live
  ON announcements (announcement_created_at DESC, announcement_created_by_teacher_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_search_gin_live
  ON announcements USING GIN (announcement_search)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_title_trgm_live
  ON announcements USING GIN (announcement_title gin_trgm_ops)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;
