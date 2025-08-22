-- +migrate Down
DROP INDEX IF EXISTS ix_announcements_title_trgm_live;
DROP INDEX IF EXISTS ix_announcements_search_gin_live;
DROP INDEX IF EXISTS ix_announcements_created_by_live;
DROP INDEX IF EXISTS ix_announcements_section_live;
DROP INDEX IF EXISTS ix_announcements_theme_live;
DROP INDEX IF EXISTS ix_announcements_tenant_date_live;
DROP INDEX IF EXISTS uq_announcements_id_tenant;
DROP TABLE IF EXISTS announcements;


-- +migrate Down
DROP INDEX IF EXISTS uq_announcement_themes_id_tenant;
DROP INDEX IF EXISTS ix_announcement_themes_name_trgm_live;
DROP INDEX IF EXISTS ix_announcement_themes_tenant_active_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_slug_live;
DROP INDEX IF EXISTS uq_announcement_themes_tenant_name_live;
DROP TABLE IF EXISTS announcement_themes;