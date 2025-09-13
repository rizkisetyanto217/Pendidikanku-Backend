-- =========================================
-- PRASYARAT
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================
-- ENUM: TIPE FILE MASJID (lengkap)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'masjid_url_type_enum') THEN
    CREATE TYPE masjid_url_type_enum AS ENUM (
      'logo',
      'stempel',
      'ttd_ketua',
      'banner',
      'profile_cover',
      'gallery',
      'qr',
      'other',
      'bg_behind_main',
      'main',
      'linktree_bg'
    );
  END IF;
END$$;

-- Pastikan semua nilai ada (idempotent)
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'logo';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'stempel';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'ttd_ketua';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'banner';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'profile_cover';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'gallery';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'qr';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'other';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'bg_behind_main';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'main';
ALTER TYPE masjid_url_type_enum ADD VALUE IF NOT EXISTS 'linktree_bg';

-- =========================================
-- TABEL: masjid_urls (lengkap; termasuk updated_at & deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS masjid_urls (
  masjid_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- relasi utama
  masjid_url_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- klasifikasi & file
  masjid_url_type     masjid_url_type_enum NOT NULL,
  masjid_url_file_url TEXT NOT NULL,

  -- trash handling
  masjid_url_trash_url            TEXT,
  masjid_url_delete_pending_until TIMESTAMPTZ,

  -- flag status
  masjid_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_url_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  masjid_url_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_url_deleted_at TIMESTAMPTZ,

  -- konsistensi pasangan trash_url + delete_pending_until
  CONSTRAINT masjid_urls_trash_pair_chk CHECK (
    (masjid_url_trash_url IS NULL AND masjid_url_delete_pending_until IS NULL)
    OR
    (masjid_url_trash_url IS NOT NULL AND masjid_url_delete_pending_until IS NOT NULL)
  )
);

-- =========================================
-- INDEXES
-- =========================================
-- Relasi & hanya yang belum dihapus
CREATE INDEX IF NOT EXISTS idx_masjid_urls_masjid
  ON masjid_urls (masjid_url_masjid_id)
  WHERE masjid_url_deleted_at IS NULL;

-- Anti-duplikat URL (case-insensitive) per masjid untuk row "alive"
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_urls_file_ci
  ON masjid_urls (masjid_url_masjid_id, LOWER(masjid_url_file_url))
  WHERE masjid_url_deleted_at IS NULL;

-- Hanya satu PRIMARY per (masjid, type) yang aktif & belum dihapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_urls_primary_per_type
  ON masjid_urls (masjid_url_masjid_id, masjid_url_type)
  WHERE masjid_url_is_active = TRUE
    AND masjid_url_is_primary = TRUE
    AND masjid_url_deleted_at IS NULL;

-- Tipe "singleton" (maks 1 baris per masjid) termasuk background & linktree
CREATE UNIQUE INDEX IF NOT EXISTS uq_masjid_urls_singleton_types
  ON masjid_urls (masjid_url_masjid_id, masjid_url_type)
  WHERE masjid_url_type IN (
    'logo','stempel','ttd_ketua','banner','profile_cover',
    'bg_behind_main','main','linktree_bg'
  )
    AND masjid_url_deleted_at IS NULL;

-- Reaper-friendly: cari yang punya due date hapus (alive saja)
CREATE INDEX IF NOT EXISTS idx_masjid_urls_delete_due
  ON masjid_urls (masjid_url_delete_pending_until)
  WHERE masjid_url_delete_pending_until IS NOT NULL
    AND masjid_url_deleted_at IS NULL;

-- =========================================
-- TRIGGERS
-- =========================================

-- pastikan single is_primary per (masjid, type) untuk row alive & active
CREATE OR REPLACE FUNCTION ensure_single_primary_masjid_url() RETURNS trigger AS $$
BEGIN
  IF NEW.masjid_url_is_primary IS TRUE THEN
    UPDATE masjid_urls
       SET masjid_url_is_primary = FALSE,
           masjid_url_updated_at = now()
     WHERE masjid_url_masjid_id = NEW.masjid_url_masjid_id
       AND masjid_url_type      = NEW.masjid_url_type
       AND masjid_url_id       <> NEW.masjid_url_id
       AND masjid_url_is_primary = TRUE
       AND masjid_url_is_active  = TRUE
       AND masjid_url_deleted_at IS NULL;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_masjid_urls_single_primary_ins ON masjid_urls;
CREATE TRIGGER trg_masjid_urls_single_primary_ins
AFTER INSERT ON masjid_urls
FOR EACH ROW
EXECUTE FUNCTION ensure_single_primary_masjid_url();

DROP TRIGGER IF EXISTS trg_masjid_urls_single_primary_upd ON masjid_urls;
CREATE TRIGGER trg_masjid_urls_single_primary_upd
AFTER UPDATE OF masjid_url_is_primary, masjid_url_type, masjid_url_masjid_id, masjid_url_is_active ON masjid_urls
FOR EACH ROW
WHEN (NEW.masjid_url_is_primary IS TRUE)
EXECUTE FUNCTION ensure_single_primary_masjid_url();

-- =========================================
-- VIEW: primary (ikut tampilkan created_at & updated_at)
-- =========================================
CREATE OR REPLACE VIEW v_masjid_primary_urls AS
SELECT
  masjid_url_masjid_id            AS masjid_id,
  masjid_url_type                 AS type,
  masjid_url_file_url             AS file_url,
  masjid_url_trash_url            AS trash_url,
  masjid_url_delete_pending_until AS delete_pending_until,
  masjid_url_created_at           AS created_at,
  masjid_url_updated_at           AS updated_at
FROM masjid_urls
WHERE masjid_url_is_active = TRUE
  AND masjid_url_is_primary = TRUE
  AND masjid_url_deleted_at IS NULL;
