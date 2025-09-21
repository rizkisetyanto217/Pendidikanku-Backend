-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- announcement_themes
-- =========================================================
-- =========================================================
-- TABLE: announcement_themes
-- =========================================================
CREATE TABLE IF NOT EXISTS announcement_themes (
  announcement_themes_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_themes_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_themes_name       VARCHAR(80)  NOT NULL,
  announcement_themes_slug       VARCHAR(120) NOT NULL,
  announcement_themes_color      VARCHAR(20),
  announcement_themes_costum_color VARCHAR(20),
  announcement_themes_description TEXT,
  announcement_themes_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- Icon (single file, 2-slot + retensi)
  announcement_themes_icon_url                   TEXT,
  announcement_themes_icon_object_key            TEXT,
  announcement_themes_icon_url_old               TEXT,
  announcement_themes_icon_object_key_old        TEXT,
  announcement_themes_icon_delete_pending_until  TIMESTAMPTZ,

  announcement_themes_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_deleted_at  TIMESTAMPTZ
);

-- =========================================================
-- INDEXING / OPTIMIZATION (announcement_themes)
-- =========================================================

-- Unik per tenant (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_name_per_tenant_alive
  ON announcement_themes (announcement_themes_masjid_id, lower(announcement_themes_name))
  WHERE announcement_themes_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_slug_per_tenant_alive
  ON announcement_themes (announcement_themes_masjid_id, lower(announcement_themes_slug))
  WHERE announcement_themes_deleted_at IS NULL;

-- Listing umum: tenant + active (live only)
CREATE INDEX IF NOT EXISTS ix_announcement_themes_tenant_active_alive
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_is_active)
  WHERE announcement_themes_deleted_at IS NULL;

-- Pencarian cepat nama (ILIKE)
CREATE INDEX IF NOT EXISTS gin_announcement_themes_name_trgm_alive
  ON announcement_themes USING GIN (announcement_themes_name gin_trgm_ops)
  WHERE announcement_themes_deleted_at IS NULL;

-- Kandidat purge icon lama
CREATE INDEX IF NOT EXISTS ix_announcement_themes_icon_purge_due
  ON announcement_themes (announcement_themes_icon_delete_pending_until)
  WHERE announcement_themes_icon_object_key_old IS NOT NULL;

-- Pair unik id + tenant (untuk FK komposit di tempat lain) — sebagai UNIQUE INDEX saja
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_id_tenant
  ON announcement_themes (announcement_themes_id, announcement_themes_masjid_id);

-- =========================================================
-- UNIQUE INDEX KOMPOSIT DI TABEL LAIN (untuk FK tenant-safe)
-- =========================================================

-- class_sections: (class_sections_id, class_sections_masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sections_id_tenant
  ON class_sections (class_sections_id, class_sections_masjid_id);

-- masjid_teachers: (masjid_teacher_id, masjid_teacher_masjid_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_teachers_id_tenant
  ON masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id);



-- =========================================================
-- TABLE: announcements
-- =========================================================
CREATE TABLE IF NOT EXISTS announcements (
  announcement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- sumber pembuat: teacher (bukan user)
  announcement_created_by_teacher_id UUID NULL,

  -- target section (NULL = global)
  announcement_class_section_id UUID NULL,

  -- tema (tenant-safe via composite FK)
  announcement_theme_id UUID NULL,

  -- SLUG (opsional; unik per tenant saat alive)
  announcement_slug VARCHAR(160),

  announcement_title   VARCHAR(200) NOT NULL,
  announcement_date    DATE NOT NULL,
  announcement_content TEXT NOT NULL,

  announcement_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,

  announcement_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_deleted_at TIMESTAMPTZ,

  -- FTS (judul + konten)
  announcement_search tsvector
    GENERATED ALWAYS AS (
      setweight(to_tsvector('simple', coalesce(announcement_title,   '')), 'A') ||
      setweight(to_tsvector('simple', coalesce(announcement_content, '')), 'B')
    ) STORED,

  -- =========================
  -- Tenant-safe FKs (komposit)
  -- =========================

  -- Teacher (SET NULL kalau teacher dihapus)
  CONSTRAINT fk_ann_created_by_teacher_same_tenant
    FOREIGN KEY (announcement_created_by_teacher_id, announcement_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Section (SET NULL jika section dihapus)
  CONSTRAINT fk_ann_section_same_tenant
    FOREIGN KEY (announcement_class_section_id, announcement_masjid_id)
    REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Theme (SET NULL jika theme dihapus)
  CONSTRAINT fk_ann_theme_same_tenant
    FOREIGN KEY (announcement_theme_id, announcement_masjid_id)
    REFERENCES announcement_themes (announcement_themes_id, announcement_themes_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =========================================================
-- Indexes / Optimizations
-- =========================================================

-- Pair unik (id + tenant)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_id_tenant
  ON announcements (announcement_id, announcement_masjid_id);

-- SLUG unik per tenant (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_slug_per_tenant_alive
  ON announcements (announcement_masjid_id, lower(announcement_slug))
  WHERE announcement_deleted_at IS NULL
    AND announcement_slug IS NOT NULL;

-- (opsional) pencarian slug cepat (case-insensitive, alive only)
CREATE INDEX IF NOT EXISTS gin_announcements_slug_trgm_live
  ON announcements USING GIN (lower(announcement_slug) gin_trgm_ops)
  WHERE announcement_deleted_at IS NULL
    AND announcement_slug IS NOT NULL;

-- Listing umum per tenant (live only) urut tanggal
CREATE INDEX IF NOT EXISTS ix_announcements_tenant_date_live
  ON announcements (announcement_masjid_id, announcement_date DESC)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Filter per tema (live only)
CREATE INDEX IF NOT EXISTS ix_announcements_theme_live
  ON announcements (announcement_theme_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Filter per section (live only)
CREATE INDEX IF NOT EXISTS ix_announcements_section_live
  ON announcements (announcement_class_section_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Filter per pembuat (live only)
CREATE INDEX IF NOT EXISTS ix_announcements_created_by_teacher_live
  ON announcements (announcement_created_at DESC, announcement_created_by_teacher_id)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Full-text search (live only)
CREATE INDEX IF NOT EXISTS ix_announcements_search_gin_live
  ON announcements USING GIN (announcement_search)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- Pencarian judul cepat (trigram, live only)
CREATE INDEX IF NOT EXISTS ix_announcements_title_trgm_live
  ON announcements USING GIN (announcement_title gin_trgm_ops)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;

-- (opsional) Scan waktu besar
CREATE INDEX IF NOT EXISTS brin_announcements_created_at
  ON announcements USING BRIN (announcement_created_at);


-- =========================================================
-- TABLE: announcement_urls
-- =========================================================
CREATE TABLE IF NOT EXISTS announcement_urls (
  announcement_url_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  announcement_url_masjid_id     UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  announcement_url_announcement_id UUID NOT NULL
    REFERENCES announcements(announcement_id) ON DELETE CASCADE,

  -- Jenis/peran aset
  -- contoh: 'banner','image','video','attachment','link'
  announcement_url_kind          VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  announcement_url_href          TEXT,        -- URL publik
  announcement_url_object_key    TEXT,        -- object key aktif di storage
  announcement_url_object_key_old TEXT,       -- object key lama (retensi in-place replace)
  
  -- Tampilan
  announcement_url_label         VARCHAR(160),
  announcement_url_order         INT NOT NULL DEFAULT 0,
  announcement_url_is_primary    BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  announcement_url_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_url_deleted_at    TIMESTAMPTZ,           -- soft delete (versi-per-baris)
  announcement_url_delete_pending_until TIMESTAMPTZ     -- tenggat purge (baris aktif dgn *_old atau baris soft-deleted)
);

-- =========================================================
-- INDEXING / OPTIMIZATION
-- =========================================================

-- Lookup per announcement (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_ann_urls_by_owner_live
  ON announcement_urls (
    announcement_url_announcement_id,
    announcement_url_kind,
    announcement_url_is_primary DESC,
    announcement_url_order,
    announcement_url_created_at
  )
  WHERE announcement_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_ann_urls_by_masjid_live
  ON announcement_urls (announcement_url_masjid_id)
  WHERE announcement_url_deleted_at IS NULL;

-- Satu primary per (announcement, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ann_urls_primary_per_kind_alive
  ON announcement_urls (announcement_url_announcement_id, announcement_url_kind)
  WHERE announcement_url_deleted_at IS NULL
    AND announcement_url_is_primary = TRUE;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_ann_urls_purge_due
  ON announcement_urls (announcement_url_delete_pending_until)
  WHERE announcement_url_delete_pending_until IS NOT NULL
    AND (
      (announcement_url_deleted_at IS NULL  AND announcement_url_object_key_old IS NOT NULL) OR
      (announcement_url_deleted_at IS NOT NULL AND announcement_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label
CREATE INDEX IF NOT EXISTS gin_ann_urls_label_trgm_live
  ON announcement_urls USING GIN (announcement_url_label gin_trgm_ops)
  WHERE announcement_url_deleted_at IS NULL;
