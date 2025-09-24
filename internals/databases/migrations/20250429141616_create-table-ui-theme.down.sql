-- =========================================================
-- DOWN Migration — UI Themes with custom support
-- =========================================================
BEGIN;

-- Drop indexes (choices)
DROP INDEX IF EXISTS ux_theme_choice_one_default_per_tenant;
DROP INDEX IF EXISTS ux_theme_choice_tenant_custom_preset;
DROP INDEX IF EXISTS ux_theme_choice_tenant_system_preset;

-- Drop tables (anak dulu)
DROP TABLE IF EXISTS ui_theme_choices;
DROP TABLE IF EXISTS ui_theme_custom_presets;
DROP TABLE IF EXISTS ui_theme_presets;

-- (opsional) DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
