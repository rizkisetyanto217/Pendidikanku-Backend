-- =========================================================
-- MIGRATION UP: lecture_exams & user_lecture_exams
-- =========================================================

-- Ekstensi
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index

-- ---------------------------------------------------------
-- Trigger functions: updated_at (TIMESTAMP)
-- ---------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_touch_updated_at_lexams()
RETURNS TRIGGER AS $$
BEGIN
  NEW.lecture_exam_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_updated_at_user_lexams()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_lecture_exam_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- ============================
-- Tabel: lecture_exams
-- ============================
CREATE TABLE IF NOT EXISTS lecture_exams (
  lecture_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_exam_title VARCHAR(255) NOT NULL,
  lecture_exam_description TEXT,
  lecture_exam_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_exam_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_exam_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  lecture_exam_updated_at TIMESTAMP
);

-- Unique judul per lecture (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_lecture_exams_title_per_lecture_ci
  ON lecture_exams (lecture_exam_lecture_id, LOWER(lecture_exam_title));

-- Index komposit untuk listing cepat
CREATE INDEX IF NOT EXISTS idx_lexams_lecture_created_desc
  ON lecture_exams (lecture_exam_lecture_id, lecture_exam_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_lexams_masjid_created_desc
  ON lecture_exams (lecture_exam_masjid_id, lecture_exam_created_at DESC);

-- Full-Text Search (judul + deskripsi)
ALTER TABLE lecture_exams
ADD COLUMN IF NOT EXISTS lecture_exam_search_tsv tsvector
GENERATED ALWAYS AS (
  setweight(to_tsvector('simple', coalesce(lecture_exam_title, '')), 'A') ||
  setweight(to_tsvector('simple', coalesce(lecture_exam_description, '')), 'B')
) STORED;

CREATE INDEX IF NOT EXISTS idx_lexams_tsv_gin
  ON lecture_exams USING GIN (lecture_exam_search_tsv);

-- Trigram untuk ILIKE (fuzzy search)
CREATE INDEX IF NOT EXISTS idx_lexams_title_trgm
  ON lecture_exams USING GIN (LOWER(lecture_exam_title) gin_trgm_ops);

-- (Opsional bila sering cari di deskripsi)
CREATE INDEX IF NOT EXISTS idx_lexams_desc_trgm
  ON lecture_exams USING GIN (LOWER(lecture_exam_description) gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_lexams_touch ON lecture_exams;
CREATE TRIGGER trg_lexams_touch
BEFORE UPDATE ON lecture_exams
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_lexams();

-- ================================
-- Tabel: user_lecture_exams
-- ================================
CREATE TABLE IF NOT EXISTS user_lecture_exams (
  user_lecture_exam_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_lecture_exam_grade_result FLOAT,  -- 0..100 (nullable)

  user_lecture_exam_exam_id UUID REFERENCES lecture_exams(lecture_exam_id) ON DELETE CASCADE,
  user_lecture_exam_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_exam_user_name VARCHAR(100),
  user_lecture_exam_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_lecture_exam_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_lecture_exam_updated_at TIMESTAMP,

  -- Data sehat
  CONSTRAINT ulexams_grade_range CHECK (
    user_lecture_exam_grade_result IS NULL
    OR (user_lecture_exam_grade_result >= 0 AND user_lecture_exam_grade_result <= 100)
  ),

  -- Satu hasil per (exam, user) — jika ingin multi-attempt, hapus constraint ini
  CONSTRAINT ux_ulexams_exam_user UNIQUE (user_lecture_exam_exam_id, user_lecture_exam_user_id)
);

-- Index pola query umum
-- Ambil hasil (exam, user) cepat + terbaru
CREATE INDEX IF NOT EXISTS idx_ulexams_exam_user_created_desc
  ON user_lecture_exams (user_lecture_exam_exam_id, user_lecture_exam_user_id, user_lecture_exam_created_at DESC);

-- Semua hasil per user (terbaru)
CREATE INDEX IF NOT EXISTS idx_ulexams_user_created_desc
  ON user_lecture_exams (user_lecture_exam_user_id, user_lecture_exam_created_at DESC);

-- Rekap per masjid (terbaru)
CREATE INDEX IF NOT EXISTS idx_ulexams_masjid_created_desc
  ON user_lecture_exams (user_lecture_exam_masjid_id, user_lecture_exam_created_at DESC);

-- Pencarian nama user (fuzzy)
CREATE INDEX IF NOT EXISTS idx_ulexams_username_trgm
  ON user_lecture_exams USING GIN (LOWER(user_lecture_exam_user_name) gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_user_lexams_touch ON user_lecture_exams;
CREATE TRIGGER trg_user_lexams_touch
BEFORE UPDATE ON user_lecture_exams
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_user_lexams();
