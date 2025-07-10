CREATE TABLE IF NOT EXISTS lecture_sessions_materials (
  lecture_sessions_material_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_material_title VARCHAR(255) NOT NULL,
  lecture_sessions_material_summary TEXT,
  lecture_sessions_material_transcript_full TEXT,
  lecture_sessions_material_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  lecture_sessions_material_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing (optional tapi disarankan untuk performa)
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_materials_lecture_session_id ON lecture_sessions_materials(lecture_sessions_material_lecture_session_id);



CREATE TABLE IF NOT EXISTS lecture_sessions_assets (
  lecture_sessions_asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_sessions_asset_title VARCHAR(255) NOT NULL,
  lecture_sessions_asset_file_url TEXT NOT NULL,
  lecture_sessions_asset_file_type INT NOT NULL, -- 1 = YouTube, 2 = PDF, 3 = DOCX, etc
  lecture_sessions_asset_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  lecture_sessions_asset_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_assets_lecture_session_id ON lecture_sessions_assets(lecture_sessions_asset_lecture_session_id);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_assets_file_type ON lecture_sessions_assets(lecture_sessions_asset_file_type);
CREATE INDEX IF NOT EXISTS idx_lecture_sessions_assets_created_at ON lecture_sessions_assets(lecture_sessions_asset_created_at);