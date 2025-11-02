-- =========================================================
-- UP Migration — UI Themes with future custom support
-- (no JSON constraints, no triggers/functions)
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================ --
-- TABLE UI THEME PRESETS --
-- ============================ --
CREATE TABLE IF NOT EXISTS ui_theme_presets (
  ui_theme_preset_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ui_theme_preset_code VARCHAR(64)  NOT NULL UNIQUE,   -- ex: 'default','sunrise','midnight',...
  ui_theme_preset_name VARCHAR(128) NOT NULL,

  ui_theme_preset_light JSONB NOT NULL,  -- Palette light (bebas, validasi di app)
  ui_theme_preset_dark  JSONB NOT NULL,  -- Palette dark

  ui_theme_preset_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ui_theme_preset_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ui_theme_preset_deleted_at TIMESTAMPTZ
);


-- ============================ --
-- TABLE UI THEME COSTUM PRESETS --
-- ============================ --
-- ---------------------------------------------------------
--    - School bisa menyimpan beberapa preset kustom
--    - Boleh "turunan" dari preset sistem: base_preset_id (opsional)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS ui_theme_custom_presets (
  ui_theme_custom_preset_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  ui_theme_custom_preset_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  ui_theme_custom_preset_code VARCHAR(64)  NOT NULL,   -- unik per school (ex: 'brand-2025')
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
  CONSTRAINT ux_custom_preset_tenant_code UNIQUE (ui_theme_custom_preset_school_id, ui_theme_custom_preset_code)
);


-- ============================ --
-- TABLE UI THEME CHOICES --
-- ============================ --
-- ---------------------------------------------------------
-- 3) CHOICES per MASJID (bisa pilih banyak; 1 default)
--    - Satu baris mengaktifkan satu pilihan untuk school
--    - Pilihan bisa dari: preset sistem ATAU custom preset
--    - Tanpa trigger: pakai CHECK untuk "exactly one"
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS ui_theme_choices (
  ui_theme_choice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  ui_theme_choice_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

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

-- Cegah duplikat pilihan preset SISTEM per school
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_tenant_system_preset
  ON ui_theme_choices (ui_theme_choice_school_id, ui_theme_choice_preset_id)
  WHERE ui_theme_choice_preset_id IS NOT NULL;

-- Cegah duplikat pilihan preset CUSTOM per school
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_tenant_custom_preset
  ON ui_theme_choices (ui_theme_choice_school_id, ui_theme_choice_custom_preset_id)
  WHERE ui_theme_choice_custom_preset_id IS NOT NULL;

-- Hanya SATU default per school
CREATE UNIQUE INDEX IF NOT EXISTS ux_theme_choice_one_default_per_tenant
  ON ui_theme_choices (ui_theme_choice_school_id)
  WHERE ui_theme_choice_is_default = TRUE;

COMMIT;



-- =========================================================
-- SEED Data untuk UI Theme Presets
-- =========================================================
INSERT INTO ui_theme_presets (
  ui_theme_preset_code,
  ui_theme_preset_name,
  ui_theme_preset_light,
  ui_theme_preset_dark
) VALUES
-- DEFAULT
(
  'default',
  'Default Theme',
  '{
    "primary": "#007074",
    "primary2": "#0070741F",
    "secondary": "#769596",
    "tertiary": "#A3DADB",
    "quaternary": "#229CC8",
    "success1": "#57B236",
    "success2": "#E1FFF8",
    "white1": "#FFFFFF",
    "white2": "#FAFAFA",
    "white3": "#EEEEEE",
    "black1": "#222222",
    "black2": "#333333",
    "error1": "#D1403F",
    "error2": "#FFEDE7",
    "warning1": "#F59D09",
    "silver1": "#DDDDDD",
    "silver2": "#888888",
    "silver4": "#4B4B4B",
    "specialColor": "#FFCC00"
  }',
  '{
    "primary": "#007074",
    "primary2": "#00707433",
    "secondary": "#5A7070",
    "tertiary": "#75C4C4",
    "quaternary": "#1D7CA5",
    "success1": "#3D8A2A",
    "success2": "#143A37",
    "white1": "#1C1C1C",
    "white2": "#2A2A2A",
    "white3": "#3A3A3A",
    "black1": "#EAEAEA",
    "black2": "#CCCCCC",
    "error1": "#C53030",
    "error2": "#331111",
    "warning1": "#B86B00",
    "silver1": "#555555",
    "silver2": "#AAAAAA",
    "silver4": "#B0B0B0",
    "specialColor": "#FFD700"
  }'
),

-- SUNRISE
(
  'sunrise',
  'Sunrise Theme',
  '{
    "primary": "#F97316",
    "primary2": "#F9731633",
    "secondary": "#F59E0B",
    "tertiary": "#FDBA74",
    "quaternary": "#FBBF24",
    "success1": "#22C55E",
    "success2": "#DCFCE7",
    "white1": "#FFF7ED",
    "white2": "#FFEEDD",
    "white3": "#FDE68A",
    "black1": "#2A2A2A",
    "black2": "#3B3B3B",
    "error1": "#DC2626",
    "error2": "#FECACA",
    "warning1": "#F59E0B",
    "silver1": "#E8E2DA",
    "silver2": "#938C84",
    "silver4": "#6B635B",
    "specialColor": "#FFD27A"
  }',
  '{
    "primary": "#FDBA74",
    "primary2": "#FDBA7433",
    "secondary": "#F59E0B",
    "tertiary": "#F97316",
    "quaternary": "#FB923C",
    "success1": "#34D399",
    "success2": "#0B3A2A",
    "white1": "#161311",
    "white2": "#201A16",
    "white3": "#2B221B",
    "black1": "#F5F2EE",
    "black2": "#E2D9D0",
    "error1": "#F87171",
    "error2": "#3A1414",
    "warning1": "#F59E0B",
    "silver1": "#5A4F46",
    "silver2": "#B9A89A",
    "silver4": "#CAB6A4",
    "specialColor": "#FFD27A"
  }'
),

-- MIDNIGHT
(
  'midnight',
  'Midnight Theme',
  '{
    "primary": "#1E3A8A",
    "primary2": "#1E3A8A33",
    "secondary": "#3B82F6",
    "tertiary": "#60A5FA",
    "quaternary": "#7C3AED",
    "success1": "#10B981",
    "success2": "#E7FFF1",
    "white1": "#F7FAFC",
    "white2": "#EEF2F7",
    "white3": "#E6EAF0",
    "black1": "#1A2230",
    "black2": "#2A3446",
    "error1": "#DC2626",
    "error2": "#FFE8E8",
    "warning1": "#F59E0B",
    "silver1": "#D6DFEA",
    "silver2": "#91A0B6",
    "silver4": "#5C6B83",
    "specialColor": "#8AB4F8"
  }',
  '{
    "primary": "#0B1220",
    "primary2": "#0B122033",
    "secondary": "#1E40AF",
    "tertiary": "#312E81",
    "quaternary": "#6D28D9",
    "success1": "#10B981",
    "success2": "#06281E",
    "white1": "#0C1018",
    "white2": "#121926",
    "white3": "#172132",
    "black1": "#E6EDF7",
    "black2": "#C6D1E4",
    "error1": "#F87171",
    "error2": "#31141A",
    "warning1": "#D97706",
    "silver1": "#2B3447",
    "silver2": "#8A99B4",
    "silver4": "#A8B4CC",
    "specialColor": "#7AA2F7"
  }'
)
ON CONFLICT (ui_theme_preset_code) DO NOTHING;
