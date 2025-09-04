-- =========================================================
-- MIGRATION UP: lecture_sessions_materials & lecture_sessions_assets (pakai soft delete)
-- =========================================================

-- Ekstensi
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index


-- ---------------------------------------------------------
-- TABEL: lecture_sessions_materials
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_sessions_materials (
  lecture_sessions_material_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  lecture_sessions_material_summary          TEXT,
  lecture_sessions_material_transcript_full  TEXT,

  lecture_sessions_material_lecture_session_id UUID NOT NULL
    REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,

  lecture_sessions_material_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_material_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_material_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_material_deleted_at TIMESTAMPTZ NULL
);

-- Index komposit umum (listing cepat)
CREATE INDEX IF NOT EXISTS idx_lsmat_session_created_desc
  ON lecture_sessions_materials (lecture_sessions_material_lecture_session_id, lecture_sessions_material_created_at DESC)
  WHERE lecture_sessions_material_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lsmat_masjid_created_desc
  ON lecture_sessions_materials (lecture_sessions_material_masjid_id, lecture_sessions_material_created_at DESC)
  WHERE lecture_sessions_material_deleted_at IS NULL;

-- Full-Text Search: summary + transcript
ALTER TABLE lecture_sessions_materials
ADD COLUMN IF NOT EXISTS lecture_sessions_material_search_tsv tsvector
GENERATED ALWAYS AS (
  setweight(to_tsvector('simple', coalesce(lecture_sessions_material_summary, '')), 'A') ||
  setweight(to_tsvector('simple', coalesce(lecture_sessions_material_transcript_full, '')), 'B')
) STORED;

CREATE INDEX IF NOT EXISTS idx_lsmat_tsv_gin
  ON lecture_sessions_materials USING GIN (lecture_sessions_material_search_tsv);

-- Trigram (fuzzy) untuk summary
CREATE INDEX IF NOT EXISTS idx_lsmat_summary_trgm
  ON lecture_sessions_materials USING GIN (LOWER(lecture_sessions_material_summary) gin_trgm_ops)
  WHERE lecture_sessions_material_deleted_at IS NULL;


 ---------------------------------------------------------
-- TABEL: lecture_sessions_assets
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_sessions_assets (
  lecture_sessions_asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  lecture_sessions_asset_title     VARCHAR(255) NOT NULL,
  lecture_sessions_asset_file_url  TEXT NOT NULL,
  lecture_sessions_asset_file_type INT  NOT NULL,  -- 1 = YouTube, 2 = PDF, 3 = DOCX, ...
  lecture_sessions_asset_lecture_session_id UUID NOT NULL
    REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  lecture_sessions_asset_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_asset_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_asset_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_asset_deleted_at TIMESTAMPTZ NULL,

  -- Data sehat
  CONSTRAINT lsasset_file_type_pos CHECK (lecture_sessions_asset_file_type >= 1),
  CONSTRAINT lsasset_file_url_nonempty CHECK (length(btrim(coalesce(lecture_sessions_asset_file_url, ''))) > 0)
);

-- Unique: judul asset tidak duplikat dalam 1 lecture_session (case-insensitive, hanya baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS ux_lsasset_title_per_session_ci
  ON lecture_sessions_assets (
    lecture_sessions_asset_lecture_session_id,
    LOWER(lecture_sessions_asset_title)
  )
  WHERE lecture_sessions_asset_deleted_at IS NULL;

-- Index komposit umum
CREATE INDEX IF NOT EXISTS idx_lsasset_session_created_desc
  ON lecture_sessions_assets (lecture_sessions_asset_lecture_session_id, lecture_sessions_asset_created_at DESC)
  WHERE lecture_sessions_asset_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lsasset_masjid_created_desc
  ON lecture_sessions_assets (lecture_sessions_asset_masjid_id, lecture_sessions_asset_created_at DESC)
  WHERE lecture_sessions_asset_deleted_at IS NULL;

-- Filter by type
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_assets_file_type 
  ON lecture_sessions_assets (lecture_sessions_asset_file_type)
  WHERE lecture_sessions_asset_deleted_at IS NULL;

-- Full-Text Search untuk title
ALTER TABLE lecture_sessions_assets
ADD COLUMN IF NOT EXISTS lecture_sessions_asset_title_tsv tsvector
GENERATED ALWAYS AS (
  setweight(to_tsvector('simple', coalesce(lecture_sessions_asset_title, '')), 'A')
) STORED;

CREATE INDEX IF NOT EXISTS idx_lsasset_title_tsv_gin
  ON lecture_sessions_assets USING GIN (lecture_sessions_asset_title_tsv);

-- Trigram (fuzzy) untuk title
CREATE INDEX IF NOT EXISTS idx_lsasset_title_trgm
  ON lecture_sessions_assets USING GIN (LOWER(lecture_sessions_asset_title) gin_trgm_ops)
  WHERE lecture_sessions_asset_deleted_at IS NULL;