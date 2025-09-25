-- +migrate Down

-- =========================
-- 1) DROP TRIGGERS (posts)
-- =========================
DROP TRIGGER IF EXISTS tg_post_theme_kind_match   ON posts;
DROP TRIGGER IF EXISTS tg_post_theme_tenant_guard ON posts;

-- =========================
-- 2) DROP FUNCTIONS
-- (trigger functions & helper funcs)
-- =========================
-- Trigger functions
DROP FUNCTION IF EXISTS trg_post_theme_kind_match()      CASCADE;
DROP FUNCTION IF EXISTS trg_post_theme_tenant_guard()    CASCADE;

-- Helper functions
DROP FUNCTION IF EXISTS resolve_teacher_sections_by_csst(UUID, UUID, UUID) CASCADE;
DROP FUNCTION IF EXISTS upsert_post_meta_subjects(UUID, UUID[])            CASCADE;
DROP FUNCTION IF EXISTS publish_post_with_sections(UUID, UUID[], BOOLEAN)  CASCADE;
DROP FUNCTION IF EXISTS publish_post_for_all_sections(UUID, UUID, BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS publish_post_via_csst(UUID, UUID, UUID, UUID, BOOLEAN) CASCADE;
DROP FUNCTION IF EXISTS publish_post_by_csst_ids(UUID, UUID[], BOOLEAN)    CASCADE;

-- =================================
-- 3) DROP TABLES (anak → induk)
-- =================================
-- post_urls (anak dari posts)
DROP TABLE IF EXISTS post_urls;

-- posts (anak dari post_themes)
DROP TABLE IF EXISTS posts;

-- post_themes (induk)
DROP TABLE IF EXISTS post_themes;

-- ============================================================
-- 4) OPTIONAL: DROP helper indexes di tabel eksternal
--    (Biasanya JANGAN dihapus karena mungkin dipakai fitur lain)
-- ============================================================
-- DROP INDEX IF EXISTS uq_class_section_id_tenant;
-- DROP INDEX IF EXISTS ix_class_sections_tenant_alive;
-- DROP INDEX IF EXISTS ix_class_sections_tenant_active;

-- DROP INDEX IF EXISTS uq_masjid_teachers_id_tenant;
-- DROP INDEX IF EXISTS ix_masjid_teachers_alive;

-- Catatan:
-- - Extensions (pgcrypto, pg_trgm) tidak perlu di-DROP di Down.
-- - Index yang berada di dalam tabel yang di-DROP, ikut terhapus otomatis.
