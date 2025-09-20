-- +migrate Down
BEGIN;

-- =========================================================
-- 0) ANNOUNCEMENT_URLS — drop indexes, table (child of announcements/masjids)
-- =========================================================
DROP INDEX IF EXISTS gin_ann_urls_label_trgm_live;
DROP INDEX IF EXISTS ix_ann_urls_purge_due;
DROP INDEX IF EXISTS uq_ann_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS ix_ann_urls_by_masjid_live;
DROP INDEX IF EXISTS ix_ann_urls_by_owner_live;

DROP TABLE IF EXISTS announcement_urls;

-- =========================================================
-- 1) ANNOUNCEMENTS — drop FKs, indexes, table
-- =========================================================

-- (Tidak usah DROP TRIGGER ON TABEL; aman di-skip)
DROP FUNCTION IF EXISTS fn_announcements_touch_updated_at();

-- FKs yang ditambahkan saat UP
ALTER TABLE IF EXISTS announcements
  DROP CONSTRAINT IF EXISTS fk_ann_section_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_theme_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_created_by_teacher_same_tenant;

-- (opsional) FK lama jika masih ada
ALTER TABLE IF EXISTS announcements
  DROP CONSTRAINT IF EXISTS fk_ann_created_by_user;

-- Indexes (sinkron dengan UP)
DROP INDEX IF EXISTS ix_announcements_title_trgm_live;
DROP INDEX IF EXISTS ix_announcements_search_gin_live;
DROP INDEX IF EXISTS ix_announcements_created_by_teacher_live;
DROP INDEX IF EXISTS ix_announcements_section_live;
DROP INDEX IF EXISTS ix_announcements_theme_live;
DROP INDEX IF EXISTS ix_announcements_tenant_date_live;
DROP INDEX IF EXISTS uq_announcements_id_tenant;
DROP INDEX IF EXISTS ix_announcements_created_by_live;

-- Table
DROP TABLE IF EXISTS announcements CASCADE;

-- =========================================================
-- 2) ANNOUNCEMENT_THEMES — drop indexes, table
--    (nama index disamakan persis dgn UP)
-- =========================================================

-- (Tidak ada trigger/function di UP; lewati atau drop function jika pernah dibuat)
DROP FUNCTION IF EXISTS fn_announcement_themes_touch_updated_at();

-- Indexes sesuai UP
DROP INDEX IF EXISTS uq_announcement_themes_id_tenant;
DROP INDEX IF EXISTS ix_announcement_themes_icon_purge_due;
DROP INDEX IF EXISTS gin_announcement_themes_name_trgm_alive;
DROP INDEX IF EXISTS ix_announcement_themes_tenant_active_alive;
DROP INDEX IF EXISTS uq_announcement_themes_slug_per_tenant_alive;
DROP INDEX IF EXISTS uq_announcement_themes_name_per_tenant_alive;

-- Table
DROP TABLE IF EXISTS announcement_themes;

-- =========================================================
-- 3) INDEX KOMPOSIT YANG DITAMBAHKAN DI TABEL LAIN — hapus
-- =========================================================
DROP INDEX IF EXISTS uq_class_sections_id_tenant;
DROP INDEX IF EXISTS uq_masjid_teachers_id_tenant;

COMMIT;
