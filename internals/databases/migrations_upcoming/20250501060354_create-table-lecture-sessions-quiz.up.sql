-- =========================================================
-- MIGRATION UP: lecture_sessions_quiz & user_lecture_sessions_quiz (soft delete)
-- =========================================================

-- Ekstensi
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index


-- ---------------------------------------------------------
-- TABEL: lecture_sessions_quiz
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_sessions_quiz (
  lecture_sessions_quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  lecture_sessions_quiz_title       VARCHAR(255) NOT NULL,
  lecture_sessions_quiz_description TEXT,

  lecture_sessions_quiz_lecture_session_id UUID NOT NULL
    REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,

  lecture_sessions_quiz_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_sessions_quiz_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_quiz_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_sessions_quiz_deleted_at TIMESTAMPTZ NULL,

  -- Hindari judul duplikat dalam 1 lecture_session (case-insensitive, hanya baris hidup)
  CONSTRAINT ux_lsquiz_title_per_session_ci UNIQUE
    (lecture_sessions_quiz_lecture_session_id, lecture_sessions_quiz_title, lecture_sessions_quiz_deleted_at)
);

-- Unique index case-insensitive (hanya yang belum dihapus)
CREATE UNIQUE INDEX IF NOT EXISTS ux_lsquiz_per_session_title_ci
  ON lecture_sessions_quiz (
    lecture_sessions_quiz_lecture_session_id,
    LOWER(lecture_sessions_quiz_title)
  )
  WHERE lecture_sessions_quiz_deleted_at IS NULL;

-- Index komposit umum
CREATE INDEX IF NOT EXISTS idx_lsquiz_session_created_desc
  ON lecture_sessions_quiz (lecture_sessions_quiz_lecture_session_id, lecture_sessions_quiz_created_at DESC)
  WHERE lecture_sessions_quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lsquiz_masjid_created_desc
  ON lecture_sessions_quiz (lecture_sessions_quiz_masjid_id, lecture_sessions_quiz_created_at DESC)
  WHERE lecture_sessions_quiz_deleted_at IS NULL;

-- Full-Text Search (judul + deskripsi)
ALTER TABLE lecture_sessions_quiz
ADD COLUMN IF NOT EXISTS lecture_sessions_quiz_search_tsv tsvector
GENERATED ALWAYS AS (
  setweight(to_tsvector('simple', coalesce(lecture_sessions_quiz_title, '')), 'A') ||
  setweight(to_tsvector('simple', coalesce(lecture_sessions_quiz_description, '')), 'B')
) STORED;

CREATE INDEX IF NOT EXISTS idx_lsquiz_tsv_gin
  ON lecture_sessions_quiz USING GIN (lecture_sessions_quiz_search_tsv);

-- Trigram untuk ILIKE fuzzy
CREATE INDEX IF NOT EXISTS idx_lsquiz_title_trgm
  ON lecture_sessions_quiz USING GIN (LOWER(lecture_sessions_quiz_title) gin_trgm_ops)
  WHERE lecture_sessions_quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lsquiz_desc_trgm
  ON lecture_sessions_quiz USING GIN (LOWER(lecture_sessions_quiz_description) gin_trgm_ops)
  WHERE lecture_sessions_quiz_deleted_at IS NULL;

-- ---------------------------------------------------------
-- TABEL: user_lecture_sessions_quiz
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_lecture_sessions_quiz (
  user_lecture_sessions_quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_lecture_sessions_quiz_grade_result    FLOAT,  -- 0..100 (nullable)
  user_lecture_sessions_quiz_quiz_id         UUID NOT NULL
    REFERENCES lecture_sessions_quiz(lecture_sessions_quiz_id) ON DELETE CASCADE,
  user_lecture_sessions_quiz_user_id         UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_sessions_quiz_attempt_count   INT NOT NULL DEFAULT 1,
  user_lecture_sessions_quiz_lecture_session_id UUID NOT NULL
    REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  user_lecture_sessions_quiz_duration_seconds INT,

  user_lecture_sessions_quiz_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_lecture_sessions_quiz_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_sessions_quiz_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_sessions_quiz_deleted_at      TIMESTAMPTZ NULL,

  -- Data sehat
  CONSTRAINT ulsq_grade_range CHECK (
    user_lecture_sessions_quiz_grade_result IS NULL
    OR (user_lecture_sessions_quiz_grade_result >= 0 AND user_lecture_sessions_quiz_grade_result <= 100)
  ),
  CONSTRAINT ulsq_attempt_min CHECK (user_lecture_sessions_quiz_attempt_count >= 1),
  CONSTRAINT ulsq_duration_nonneg CHECK (user_lecture_sessions_quiz_duration_seconds IS NULL OR user_lecture_sessions_quiz_duration_seconds >= 0),

  -- Satu baris per (quiz, user, attempt)
  CONSTRAINT ux_ulsq_quser_attempt UNIQUE (
    user_lecture_sessions_quiz_quiz_id,
    user_lecture_sessions_quiz_user_id,
    user_lecture_sessions_quiz_attempt_count,
    user_lecture_sessions_quiz_deleted_at
  )
);

-- Index pola query umum
CREATE INDEX IF NOT EXISTS idx_ulsq_quser_created_desc
  ON user_lecture_sessions_quiz (user_lecture_sessions_quiz_quiz_id, user_lecture_sessions_quiz_user_id, user_lecture_sessions_quiz_created_at DESC)
  WHERE user_lecture_sessions_quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ulsq_session_user
  ON user_lecture_sessions_quiz (user_lecture_sessions_quiz_lecture_session_id, user_lecture_sessions_quiz_user_id)
  WHERE user_lecture_sessions_quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ulsq_user_created_desc
  ON user_lecture_sessions_quiz (user_lecture_sessions_quiz_user_id, user_lecture_sessions_quiz_created_at DESC)
  WHERE user_lecture_sessions_quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ulsq_masjid_created_desc
  ON user_lecture_sessions_quiz (user_lecture_sessions_quiz_masjid_id, user_lecture_sessions_quiz_created_at DESC)
  WHERE user_lecture_sessions_quiz_deleted_at IS NULL;
