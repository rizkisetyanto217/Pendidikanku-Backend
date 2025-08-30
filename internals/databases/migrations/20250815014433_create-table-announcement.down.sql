-- +migrate Down
BEGIN;

-- =========================================================
-- 1) ANNOUNCEMENT_URLS — drop trigger, index, FK, table
-- =========================================================

-- Trigger & function
DROP TRIGGER IF EXISTS trg_announcement_urls_touch_updated_at ON announcement_urls;
DROP FUNCTION IF EXISTS fn_announcement_urls_touch_updated_at();

-- Indexes
DROP INDEX IF EXISTS ix_announcement_urls_delete_pending;
DROP INDEX IF EXISTS ix_announcement_urls_label_trgm_live;
DROP INDEX IF EXISTS ix_announcement_urls_masjid_live;
DROP INDEX IF EXISTS ix_announcement_urls_announcement_live;
DROP INDEX IF EXISTS uq_announcement_urls_announcement_href_live;
DROP INDEX IF EXISTS uq_announcement_urls_id_tenant;

-- FK (composite ke announcements)
ALTER TABLE IF EXISTS announcement_urls
  DROP CONSTRAINT IF EXISTS fk_au_announcement_same_tenant;

-- Table
DROP TABLE IF EXISTS announcement_urls;


-- =========================================================
-- 2) ANNOUNCEMENTS — rollback teacher_id → user_id, drop FK/index/trigger
-- =========================================================

-- Trigger & function
DROP TRIGGER IF EXISTS trg_announcements_touch_updated_at ON announcements;
DROP FUNCTION IF EXISTS fn_announcements_touch_updated_at();

-- Hapus FK komposit yang dibuat saat Up
ALTER TABLE IF EXISTS announcements
  DROP CONSTRAINT IF EXISTS fk_ann_section_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_theme_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_created_by_teacher_same_tenant;

-- Tambahkan kembali kolom lama user_id (jika belum ada)
ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS announcement_created_by_user_id UUID;

-- Backfill user_id dari teacher_id via masjid_teachers
DO $$
BEGIN
  UPDATE announcements a
     SET announcement_created_by_user_id = mt.masjid_teacher_user_id
    FROM masjid_teachers mt
   WHERE a.announcement_created_by_teacher_id IS NOT NULL
     AND mt.masjid_teacher_id = a.announcement_created_by_teacher_id
     AND mt.masjid_teacher_masjid_id = a.announcement_masjid_id
     AND (a.announcement_created_by_user_id IS NULL
          OR a.announcement_created_by_user_id <> mt.masjid_teacher_user_id);
END$$;

-- Hapus kolom teacher_id
ALTER TABLE announcements
  DROP COLUMN IF EXISTS announcement_created_by_teacher_id;

-- Pulihkan FK lama ke users (best-effort, nama baru agar tidak bentrok)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid='announcements'::regclass
      AND conname='fk_ann_created_by_user'
  ) THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_created_by_user
      FOREIGN KEY (announcement_created_by_user_id)
      REFERENCES users(id)
      ON UPDATE CASCADE
      ON DELETE SET NULL;
  END IF;
END$$;

-- (Opsional) Pulihkan FK single-column ke class_sections (sebelum migrasi komposit)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid='announcements'::regclass
      AND conname='fk_ann_section_single'
  ) THEN
    ALTER TABLE announcements
      ADD CONSTRAINT fk_ann_section_single
      FOREIGN KEY (announcement_class_section_id)
      REFERENCES class_sections (class_sections_id)
      ON UPDATE CASCADE
      ON DELETE SET NULL;
  END IF;
END$$;

-- Hapus index yang dibuat saat Up
DROP INDEX IF EXISTS ix_announcements_title_trgm_live;
DROP INDEX IF EXISTS ix_announcements_search_gin_live;
DROP INDEX IF EXISTS ix_announcements_created_by_teacher_live;
DROP INDEX IF EXISTS ix_announcements_section_live;
DROP INDEX IF EXISTS ix_announcements_theme_live;
DROP INDEX IF EXISTS ix_announcements_tenant_date_live;
DROP INDEX IF EXISTS uq_announcements_id_tenant;

-- (Opsional) Kembalikan index lama berbasis user_id (kalau memang ingin restore)
CREATE INDEX IF NOT EXISTS ix_announcements_created_by_live
  ON announcements (announcement_created_by_user_id, announcement_created_at DESC)
  WHERE announcement_deleted_at IS NULL AND announcement_is_active = TRUE;


-- =========================================================
-- 3) ANNOUNCEMENT_THEMES — drop trigger, index/unique, table
-- =========================================================

-- Trigger & function
DROP TRIGGER IF EXISTS trg_announcement_themes_touch_updated_at ON announcement_themes;
DROP FUNCTION IF EXISTS fn_announcement_themes_touch_updated_at();

-- Hapus constraint unik komposit jika ada (karena Up bisa menambahkan sebagai CONSTRAINT)
ALTER TABLE IF EXISTS announcement_themes
  DROP CONSTRAINT IF EXISTS uq_announcement_themes_id_tenant;

-- Indexes (yang dibuat sebagai UNIQUE INDEX atau index biasa)
DROP INDEX IF EXISTS uq_announcement_themes_id_tenant;
DROP INDEX IF EXISTS ix_announcement_themes_name_trgm_live;
DROP INDEX IF EXISTS ix_announcement_themes_tenant_active_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_slug_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_name_live;

-- Table
DROP TABLE IF EXISTS announcement_themes;

COMMIT;
