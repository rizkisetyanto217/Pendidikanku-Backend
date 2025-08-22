-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS announcement_themes (
  announcement_themes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_themes_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_themes_name VARCHAR(80) NOT NULL,
  announcement_themes_slug VARCHAR(120) NOT NULL,
  announcement_themes_color VARCHAR(20),
  announcement_themes_description TEXT, 
  announcement_themes_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  announcement_themes_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  announcement_themes_updated_at TIMESTAMP,
  announcement_themes_deleted_at TIMESTAMP,

  CONSTRAINT ck_announcement_themes_slug
    CHECK (announcement_themes_slug ~ '^[a-z0-9-]+$')
);

-- Unik per tenant untuk baris yang belum terhapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_name_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_name)
  WHERE announcement_themes_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_slug_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_slug)
  WHERE announcement_themes_deleted_at IS NULL;

-- Bantu query (hanya yang live)
CREATE INDEX IF NOT EXISTS ix_announcement_themes_tenant_active_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_is_active)
  WHERE announcement_themes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_announcement_themes_name_trgm_live
  ON announcement_themes USING GIN (announcement_themes_name gin_trgm_ops)
  WHERE announcement_themes_deleted_at IS NULL;

-- Untuk FK komposit tenant-safe dari tabel lain
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_id_tenant
  ON announcement_themes (announcement_themes_id, announcement_themes_masjid_id);



-- 2) Announcements (global jika section NULL, dibuat oleh user admin/teacher)
CREATE TABLE IF NOT EXISTS announcements (
  announcement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- pembuat (admin/teacher)
  announcement_created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

  -- target 1 section (NULL = GLOBAL → tampil ke semua user dalam masjid)
  announcement_class_section_id UUID NULL
    REFERENCES class_sections(class_sections_id) ON DELETE SET NULL,

  -- tema (tenant-safe)
  announcement_theme_id UUID NULL,
  CONSTRAINT fk_ann_theme_same_tenant
    FOREIGN KEY (announcement_theme_id, announcement_masjid_id)
    REFERENCES announcement_themes (announcement_themes_id, announcement_themes_masjid_id)
    ON DELETE SET NULL,

  announcement_title   VARCHAR(200) NOT NULL,
  announcement_date    DATE NOT NULL,
  announcement_content TEXT NOT NULL,

  announcement_attachment_url TEXT,

  -- 🔥 cukup 1 toggle aktif/non-aktif
  announcement_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  announcement_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  announcement_updated_at TIMESTAMP,
  announcement_deleted_at TIMESTAMP,

  -- FTS
  announcement_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(announcement_title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(announcement_content, '')), 'B')
  ) STORED
);

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