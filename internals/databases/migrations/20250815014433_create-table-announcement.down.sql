-- +migrate Down
BEGIN;

-- =========================================================
-- 1) ANNOUNCEMENTS — drop trigger, fk, index, table
-- =========================================================

-- Trigger & function
DROP TRIGGER IF EXISTS trg_announcements_touch_updated_at ON announcements;
DROP FUNCTION IF EXISTS fn_announcements_touch_updated_at();

-- Hapus FK yang dibuat saat Up
ALTER TABLE IF EXISTS announcements
  DROP CONSTRAINT IF EXISTS fk_ann_section_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_theme_same_tenant,
  DROP CONSTRAINT IF EXISTS fk_ann_created_by_teacher_same_tenant;

-- Hapus FK lama ke users kalau ada
ALTER TABLE IF EXISTS announcements
  DROP CONSTRAINT IF EXISTS fk_ann_created_by_user;

-- Indexes
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
-- 2) ANNOUNCEMENT_THEMES — drop trigger, index/unique, table
-- =========================================================

-- Trigger & function
DROP TRIGGER IF EXISTS trg_announcement_themes_touch_updated_at ON announcement_themes;
DROP FUNCTION IF EXISTS fn_announcement_themes_touch_updated_at();

-- Constraints
ALTER TABLE IF EXISTS announcement_themes
  DROP CONSTRAINT IF EXISTS uq_announcement_themes_id_tenant;

-- Indexes
DROP INDEX IF EXISTS uq_announcement_themes_id_tenant;
DROP INDEX IF EXISTS ix_announcement_themes_name_trgm_live;
DROP INDEX IF EXISTS ix_announcement_themes_tenant_active_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_slug_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_name_live;

-- Table
DROP TABLE IF EXISTS announcement_themes;

COMMIT;
