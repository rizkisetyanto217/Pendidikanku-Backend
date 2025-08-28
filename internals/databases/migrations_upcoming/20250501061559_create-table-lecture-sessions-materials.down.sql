-- =========================================================
-- MIGRATION DOWN: lecture_sessions_materials & lecture_sessions_assets
-- =========================================================

-- Hapus triggers
DROP TRIGGER IF EXISTS trg_lsasset_touch ON lecture_sessions_assets;
DROP TRIGGER IF EXISTS trg_lsmat_touch ON lecture_sessions_materials;

-- Hapus index lecture_sessions_assets
DROP INDEX IF EXISTS idx_lsasset_title_trgm;
DROP INDEX IF EXISTS idx_lsasset_title_tsv_gin;
DROP INDEX IF EXISTS idx_lsasset_masjid_created_desc;
DROP INDEX IF EXISTS idx_lsasset_session_created_desc;
DROP INDEX IF EXISTS ux_lsasset_title_per_session_ci;
DROP INDEX IF EXISTS idx_lecture_sessions_assets_file_type;

-- Hapus index lecture_sessions_materials
DROP INDEX IF EXISTS idx_lsmat_summary_trgm;
DROP INDEX IF EXISTS idx_lsmat_tsv_gin;
DROP INDEX IF EXISTS idx_lsmat_masjid_created_desc;
DROP INDEX IF EXISTS idx_lsmat_session_created_desc;

-- Drop tables (anak/bergantung terakhir)
DROP TABLE IF EXISTS lecture_sessions_assets;
DROP TABLE IF EXISTS lecture_sessions_materials;

-- Hapus trigger functions
DROP FUNCTION IF EXISTS fn_touch_updated_at_lsassets();
DROP FUNCTION IF EXISTS fn_touch_updated_at_lsmaterials();

-- (Ekstensi pg_trgm/pgcrypto tidak di-drop karena mungkin dipakai objek lain)
