-- =========================================================
-- UP Migration — UI Themes with future custom support
-- (no JSON constraints, no triggers/functions)
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------
-- 1) MASTER PRESETS (Sistem)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS ui_theme_presets (
  ui_theme_preset_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ui_theme_preset_code VARCHAR(64)  NOT NULL UNIQUE,   -- ex: 'default','sunrise','midnight',...
  ui_theme_preset_name VARCHAR(128) NOT NULL,

  ui_theme_preset_light JSONB NOT NULL,  -- Palette light (bebas, validasi di app)
  ui_theme_preset_dark  JSONB NOT NULL,  -- Palette dark

  ui_theme_preset_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ui_theme_preset_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------
-- 2) CUSTOM PRESETS (per masjid)
--    - Masjid bisa menyimpan beberapa preset kustom
--    - Boleh "turunan" dari preset sistem: base_preset_id (opsional)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS ui_theme_custom_presets (
  ui_theme_custom_preset_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  ui_theme_custom_preset_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  ui_theme_custom_preset_code VARCHAR(64)  NOT NULL,   -- unik per masjid (ex: 'brand-2025')
  ui_theme_custom_preset_name VARCHAR(128) NOT NULL,

  ui_theme_custom_preset_light JSONB NOT NULL,
  ui_theme_custom_preset_dark  JSONB NOT NULL,

  -- opsional: jejak asal kustomisasi dari preset sistem
  ui_theme_custom_base_preset_id UUID
    REFERENCES ui_theme_presets(ui_theme_preset_id) ON DELETE SET NULL,

  ui_theme_custom_preset_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  ui_theme_custom_preset_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ui_theme_custom_preset_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- unik per tenant
  CONSTRAINT ux_custom_preset_tenant_code UNIQUE (ui_theme_custom_preset_masjid_id, ui_theme_custom_preset_code)
);

-- ---------------------------------------------------------
-- 3) CHOICES per MASJID (bisa pilih banyak; 1 default)
--    - Satu baris mengaktifkan satu pilihan untuk masjid
--    - Pilihan bisa dari: preset sistem ATAU custom preset
--    - Tanpa trigger: pakai CHECK untuk "exactly one"
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS ui_theme_choices (
  ui_theme_choice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  ui_theme_choice_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- salah satu harus diisi:
  ui_theme_choice_preset_id UUID
    REFERENCES ui_theme_presets(ui_theme_preset_id) ON DELETE CASCADE,

  ui_theme_choice_custom_preset_id UUID
    REFERENCES ui_theme_custom_presets(ui_theme_custom_preset_id) ON DELETE CASCADE,

  -- exactly one of (preset_id, custom_preset_id) must be non-null
  CONSTRAINT chk_theme_choice_one_of CHECK (
    (ui_theme_choice_preset_id IS NOT NULL AND ui_theme_choice_custom_preset_id IS NULL)
    OR
    (ui_theme_choice_preset_id IS NULL AND ui_theme_choice_custom_preset_id IS NOT NULL)
  ),

  ui_theme_choice_is_default BOOLEAN NOT NULL DEFAULT FALSE,
  ui_theme_choice_is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

  ui_theme_choice_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ui_theme_choice_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------
-- 4) Indexes & Constraints
-- ---------------------------------------------------------

-- Cegah duplikat pilihan preset SISTEM per masjid
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_tenant_system_preset
  ON ui_theme_choices (ui_theme_choice_masjid_id, ui_theme_choice_preset_id)
  WHERE ui_theme_choice_preset_id IS NOT NULL;

-- Cegah duplikat pilihan preset CUSTOM per masjid
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_tenant_custom_preset
  ON ui_theme_choices (ui_theme_choice_masjid_id, ui_theme_choice_custom_preset_id)
  WHERE ui_theme_choice_custom_preset_id IS NOT NULL;

-- Hanya SATU default per masjid
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_one_default_per_tenant
  ON ui_theme_choices (ui_theme_choice_masjid_id)
  WHERE ui_theme_choice_is_default = TRUE;

COMMIT;
