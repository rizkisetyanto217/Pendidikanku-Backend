-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- announcement_themes (timestamps -> TIMESTAMP)
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

-- Tenant-safe composite FK target
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_id_tenant
  ON announcement_themes (announcement_themes_id, announcement_themes_masjid_id);

-- Trigger updated_at
CREATE OR REPLACE FUNCTION fn_announcement_themes_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.announcement_themes_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcement_themes_touch_updated_at') THEN
    DROP TRIGGER trg_announcement_themes_touch_updated_at ON announcement_themes;
  END IF;
  CREATE TRIGGER trg_announcement_themes_touch_updated_at
    BEFORE UPDATE ON announcement_themes
    FOR EACH ROW
    EXECUTE FUNCTION fn_announcement_themes_touch_updated_at();
END$$;



-- =========================================================
-- announcements (timestamps -> TIMESTAMP)
-- =========================================================
CREATE TABLE IF NOT EXISTS announcements (
  announcement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

  -- target 1 section (NULL = GLOBAL)
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
  announcement_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(announcement_title,   '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(announcement_content, '')), 'B')
  ) STORED
);

-- Tenant-safe composite FK ke THEME
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_ann_theme_same_tenant'
  ) THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_theme_same_tenant
      FOREIGN KEY (announcement_theme_id, announcement_masjid_id)
      REFERENCES announcement_themes (announcement_themes_id, announcement_themes_masjid_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Pastikan class_sections punya UNIQUE (id, masjid_id) untuk composite FK
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

-- Tenant-safe composite FK ke SECTION
DO $$
BEGIN
  -- drop FK single-column lama kalau ada
  PERFORM 1 FROM pg_constraint
    WHERE conrelid='announcements'::regclass AND contype='f'
      AND pg_get_constraintdef(oid) ILIKE '%(announcement_class_section_id)%class_sections%';
  IF FOUND THEN
    DO $inner$
    DECLARE r RECORD;
    BEGIN
      FOR r IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid='announcements'::regclass AND contype='f'
          AND pg_get_constraintdef(oid) ILIKE '%(announcement_class_section_id)%class_sections%'
      LOOP
        EXECUTE format('ALTER TABLE announcements DROP CONSTRAINT %I', r.conname);
      END LOOP;
    END;
    $inner$;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_ann_section_same_tenant'
  ) THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_section_same_tenant
      FOREIGN KEY (announcement_class_section_id, announcement_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Composite key (opsional) untuk join tenant-safe
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_id_tenant
  ON announcements (announcement_id, announcement_masjid_id);

-- “live only” (aktif & belum terhapus)
CREATE INDEX IF NOT EXISTS ix_announcements_tenant_date_live
  ON announcements (announcement_masjid_id, announcement_date DESC)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_theme_live
  ON announcements (announcement_theme_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_section_live
  ON announcements (announcement_class_section_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_created_by_live
  ON announcements (announcement_created_by_user_id, announcement_created_at DESC)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_search_gin_live
  ON announcements USING GIN (announcement_search)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_announcements_title_trgm_live
  ON announcements USING GIN (announcement_title gin_trgm_ops)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION fn_announcements_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.announcement_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcements_touch_updated_at') THEN
    DROP TRIGGER trg_announcements_touch_updated_at ON announcements;
  END IF;
  CREATE TRIGGER trg_announcements_touch_updated_at
    BEFORE UPDATE ON announcements
    FOR EACH ROW
    EXECUTE FUNCTION fn_announcements_touch_updated_at();
END$$;



-- +migrate Up
-- =========================================================
-- ANNOUNCEMENT URLS (child dari announcements, tanpa is_active)
-- =========================================================

CREATE TABLE IF NOT EXISTS announcement_urls (
  announcement_url_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_url_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- relasi ke announcements (tenant-safe via composite FK)
  announcement_url_announcement_id UUID NOT NULL,

  -- data url
  announcement_url_label       VARCHAR(120),
  announcement_url_href        TEXT NOT NULL,          -- URL utama
  announcement_url_trash_url   TEXT,                   -- URL lama dipindah ke trash
  announcement_url_delete_pending_until TIMESTAMPTZ,   -- jadwal penghapusan permanen

  announcement_url_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_deleted_at  TIMESTAMPTZ
);

-- Composite FK ke announcements (tenant-safe)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_au_announcement_same_tenant'
  ) THEN
    ALTER TABLE announcement_urls
      ADD CONSTRAINT fk_au_announcement_same_tenant
      FOREIGN KEY (announcement_url_announcement_id, announcement_url_masjid_id)
      REFERENCES announcements (announcement_id, announcement_masjid_id)
      ON UPDATE CASCADE
      ON DELETE CASCADE;
  END IF;
END$$;

-- Composite key bantuan untuk join tenant-safe
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_urls_id_tenant
  ON announcement_urls (announcement_url_id, announcement_url_masjid_id);

-- Cegah duplikat URL aktif di satu pengumuman (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_urls_announcement_href_live
  ON announcement_urls (
    announcement_url_announcement_id,
    lower(announcement_url_href)
  )
  WHERE announcement_url_deleted_at IS NULL;

-- Indeks query umum
CREATE INDEX IF NOT EXISTS ix_announcement_urls_announcement_live
  ON announcement_urls (announcement_url_announcement_id)
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_masjid_live
  ON announcement_urls (announcement_url_masjid_id)
  WHERE announcement_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_urls_label_trgm_live
  ON announcement_urls USING GIN (announcement_url_label gin_trgm_ops)
  WHERE announcement_url_deleted_at IS NULL;

-- Indeks tambahan untuk monitoring penghapusan pending
CREATE INDEX IF NOT EXISTS ix_announcement_urls_delete_pending
  ON announcement_urls (announcement_url_delete_pending_until)
  WHERE announcement_url_deleted_at IS NULL;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION fn_announcement_urls_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.announcement_url_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_announcement_urls_touch_updated_at') THEN
    DROP TRIGGER trg_announcement_urls_touch_updated_at ON announcement_urls;
  END IF;
  CREATE TRIGGER trg_announcement_urls_touch_updated_at
    BEFORE UPDATE ON announcement_urls
    FOR EACH ROW
    EXECUTE FUNCTION fn_announcement_urls_touch_updated_at();
END$$;
