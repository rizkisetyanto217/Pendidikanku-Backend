-- +migrate Down
-- =========================================================
-- DOWN — UI Themes (presets, custom presets, choices)
-- =========================================================

-- 1) CHILD: ui_theme_choices
-- (index akan ikut terhapus bersama tabel; baris DROP INDEX opsional)
DROP INDEX IF EXISTS ux_theme_choice_one_default_per_tenant;
DROP INDEX IF EXISTS ux_theme_choice_tenant_custom_preset;
DROP INDEX IF EXISTS ux_theme_choice_tenant_system_preset;

DROP TABLE IF EXISTS ui_theme_choices;

-- 2) MIDDLE: ui_theme_custom_presets
-- (punya FK ke masjids dan ui_theme_presets)
-- (index unique ux_custom_preset_tenant_code ikut terhapus)
DROP TABLE IF EXISTS ui_theme_custom_presets;

-- 3) PARENT: ui_theme_presets
-- (punya data seed; cukup DROP TABLE)
DROP TABLE IF EXISTS ui_theme_presets;

-- Catatan:
-- - Tidak ada ENUM/trigger/function pada blok UP ini, jadi tidak ada yang perlu di-drop selain tabel & index.
-- - Extension pgcrypto dibiarkan (umum dipakai migrasi lain).
