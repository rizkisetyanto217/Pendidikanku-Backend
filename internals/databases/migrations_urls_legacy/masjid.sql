-- =========================================
-- PRASYARAT
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================
-- ENUM: TIPE FILE MASJID (lengkap)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'school_url_type_enum') THEN
    CREATE TYPE school_url_type_enum AS ENUM (
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
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'logo';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'stempel';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'ttd_ketua';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'banner';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'profile_cover';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'gallery';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'qr';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'other';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'bg_behind_main';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'main';
ALTER TYPE school_url_type_enum ADD VALUE IF NOT EXISTS 'linktree_bg';

-- =========================================
-- TABEL: school_urls (lengkap; termasuk updated_at & deleted_at)
-- =========================================
CREATE TABLE IF NOT EXISTS school_urls (
  school_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- relasi utama
  school_url_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- klasifikasi & file
  school_url_type     school_url_type_enum NOT NULL,
  school_url_file_url TEXT NOT NULL,

  -- trash handling
  school_url_trash_url            TEXT,
  school_url_delete_pending_until TIMESTAMPTZ,

  -- flag status
  school_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  school_url_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  school_url_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_url_deleted_at TIMESTAMPTZ,

  -- konsistensi pasangan trash_url + delete_pending_until
  CONSTRAINT school_urls_trash_pair_chk CHECK (
    (school_url_trash_url IS NULL AND school_url_delete_pending_until IS NULL)
    OR
    (school_url_trash_url IS NOT NULL AND school_url_delete_pending_until IS NOT NULL)
  )
);

-- =========================================
-- INDEXES
-- =========================================
-- Relasi & hanya yang belum dihapus
CREATE INDEX IF NOT EXISTS idx_school_urls_school
  ON school_urls (school_url_school_id)
  WHERE school_url_deleted_at IS NULL;

-- Anti-duplikat URL (case-insensitive) per school untuk row "alive"
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_urls_file_ci
  ON school_urls (school_url_school_id, LOWER(school_url_file_url))
  WHERE school_url_deleted_at IS NULL;

-- Hanya satu PRIMARY per (school, type) yang aktif & belum dihapus
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_urls_primary_per_type
  ON school_urls (school_url_school_id, school_url_type)
  WHERE school_url_is_active = TRUE
    AND school_url_is_primary = TRUE
    AND school_url_deleted_at IS NULL;

-- Tipe "singleton" (maks 1 baris per school) termasuk background & linktree
CREATE UNIQUE INDEX IF NOT EXISTS uq_school_urls_singleton_types
  ON school_urls (school_url_school_id, school_url_type)
  WHERE school_url_type IN (
    'logo','stempel','ttd_ketua','banner','profile_cover',
    'bg_behind_main','main','linktree_bg'
  )
    AND school_url_deleted_at IS NULL;

-- Reaper-friendly: cari yang punya due date hapus (alive saja)
CREATE INDEX IF NOT EXISTS idx_school_urls_delete_due
  ON school_urls (school_url_delete_pending_until)
  WHERE school_url_delete_pending_until IS NOT NULL
    AND school_url_deleted_at IS NULL;

-- =========================================
-- TRIGGERS
-- =========================================

-- pastikan single is_primary per (school, type) untuk row alive & active
CREATE OR REPLACE FUNCTION ensure_single_primary_school_url() RETURNS trigger AS $$
BEGIN
  IF NEW.school_url_is_primary IS TRUE THEN
    UPDATE school_urls
       SET school_url_is_primary = FALSE,
           school_url_updated_at = now()
     WHERE school_url_school_id = NEW.school_url_school_id
       AND school_url_type      = NEW.school_url_type
       AND school_url_id       <> NEW.school_url_id
       AND school_url_is_primary = TRUE
       AND school_url_is_active  = TRUE
       AND school_url_deleted_at IS NULL;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_school_urls_single_primary_ins ON school_urls;
CREATE TRIGGER trg_school_urls_single_primary_ins
AFTER INSERT ON school_urls
FOR EACH ROW
EXECUTE FUNCTION ensure_single_primary_school_url();

DROP TRIGGER IF EXISTS trg_school_urls_single_primary_upd ON school_urls;
CREATE TRIGGER trg_school_urls_single_primary_upd
AFTER UPDATE OF school_url_is_primary, school_url_type, school_url_school_id, school_url_is_active ON school_urls
FOR EACH ROW
WHEN (NEW.school_url_is_primary IS TRUE)
EXECUTE FUNCTION ensure_single_primary_school_url();

-- =========================================
-- VIEW: primary (ikut tampilkan created_at & updated_at)
-- =========================================
CREATE OR REPLACE VIEW v_school_primary_urls AS
SELECT
  school_url_school_id            AS school_id,
  school_url_type                 AS type,
  school_url_file_url             AS file_url,
  school_url_trash_url            AS trash_url,
  school_url_delete_pending_until AS delete_pending_until,
  school_url_created_at           AS created_at,
  school_url_updated_at           AS updated_at
FROM school_urls
WHERE school_url_is_active = TRUE
  AND school_url_is_primary = TRUE
  AND school_url_deleted_at IS NULL;
