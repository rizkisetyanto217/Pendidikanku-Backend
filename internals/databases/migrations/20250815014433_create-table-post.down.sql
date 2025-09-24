-- +migrate Down

-- =====================================================================
-- DROP FUNCTIONS (publish, resolver, trigger helpers)
-- =====================================================================
DROP FUNCTION IF EXISTS publish_post_by_csst_ids(UUID, UUID[], BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS publish_post_via_csst(UUID, UUID, UUID, UUID, BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS publish_post_for_all_sections(UUID, UUID, BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS publish_post_with_sections(UUID, UUID[], BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS upsert_post_meta_subjects(UUID, UUID[]) CASCADE;
DROP FUNCTION IF EXISTS resolve_teacher_sections_by_csst(UUID, UUID, UUID) CASCADE;

DROP FUNCTION IF EXISTS trg_post_theme_kind_match() CASCADE;
DROP FUNCTION IF EXISTS trg_post_theme_tenant_guard() CASCADE;

-- =====================================================================
-- DROP TRIGGERS
-- =====================================================================
DROP TRIGGER IF EXISTS tg_post_theme_kind_match ON post;
DROP TRIGGER IF EXISTS tg_post_theme_tenant_guard ON post;

-- =====================================================================
-- DROP TABLES (urutan dari child ke parent)
-- =====================================================================
DROP TABLE IF EXISTS post_urls CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS post_themes CASCADE;
