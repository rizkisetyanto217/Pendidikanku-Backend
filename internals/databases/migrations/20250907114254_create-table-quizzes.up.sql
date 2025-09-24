-- =========================================
-- UP Migration — TABLES + STRONG FKs (no selected_option_id)
-- Fresh create (tanpa ALTER / DO blocks)
-- =========================================

-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- 1) QUIZZES
-- =========================================
CREATE TABLE IF NOT EXISTS quizzes (
  quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  quiz_assessment_id UUID
    REFERENCES assessments(assessment_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- >>> SLUG (opsional; unik per tenant saat alive)
  quiz_slug VARCHAR(160),

  quiz_title         VARCHAR(180) NOT NULL,
  quiz_description   TEXT,
  quiz_is_published  BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_time_limit_sec INT,

  quiz_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_deleted_at TIMESTAMPTZ
);

-- Indexes / Optimizations (quizzes)

-- SLUG unik per tenant (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_quizzes_slug_per_tenant_alive
  ON quizzes (quiz_masjid_id, LOWER(quiz_slug))
  WHERE quiz_deleted_at IS NULL
    AND quiz_slug IS NOT NULL;

-- (opsional) pencarian slug cepat (trigram, alive only)
CREATE INDEX IF NOT EXISTS gin_quizzes_slug_trgm_alive
  ON quizzes USING GIN (LOWER(quiz_slug) gin_trgm_ops)
  WHERE quiz_deleted_at IS NULL
    AND quiz_slug IS NOT NULL;

-- pair unik id+tenant (tenant-safe FK di masa depan)
CREATE UNIQUE INDEX IF NOT EXISTS uq_quizzes_id_tenant
  ON quizzes (quiz_id, quiz_masjid_id);

-- Publikasi per tenant (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_published
  ON quizzes (quiz_masjid_id, quiz_is_published)
  WHERE quiz_deleted_at IS NULL;

-- Relasi assessment (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_assessment
  ON quizzes (quiz_assessment_id)
  WHERE quiz_deleted_at IS NULL;

-- Time-range besar (BRIN)
CREATE INDEX IF NOT EXISTS brin_quizzes_created_at
  ON quizzes USING BRIN (quiz_created_at);

-- Pencarian judul & deskripsi (trigram, alive only)
CREATE INDEX IF NOT EXISTS gin_quizzes_title_trgm
  ON quizzes USING GIN (quiz_title gin_trgm_ops)
  WHERE quiz_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_quizzes_desc_trgm
  ON quizzes USING GIN (quiz_description gin_trgm_ops)
  WHERE quiz_deleted_at IS NULL;

-- Kombinasi tenant + assessment (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_assessment
  ON quizzes (quiz_masjid_id, quiz_assessment_id)
  WHERE quiz_deleted_at IS NULL;

-- Listing terbaru per tenant (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_created_desc
  ON quizzes (quiz_masjid_id, quiz_created_at DESC)
  WHERE quiz_deleted_at IS NULL;



-- =========================================
-- 2) QUIZ_QUESTIONS
-- =========================================
CREATE TABLE IF NOT EXISTS quiz_questions (
  quiz_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  quiz_question_quiz_id   UUID NOT NULL
    REFERENCES quizzes(quiz_id) ON DELETE CASCADE,

  quiz_question_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  quiz_question_type VARCHAR(8) NOT NULL
    CHECK (quiz_question_type IN ('single','essay')),

  quiz_question_text   TEXT NOT NULL,
  quiz_question_points NUMERIC(6,2) NOT NULL DEFAULT 1
    CHECK (quiz_question_points >= 0),

  quiz_question_answers JSONB,
  quiz_question_correct CHAR(1)
    CHECK (quiz_question_correct IN ('A','B','C','D')),

  quiz_question_explanation TEXT,

  -- Constraint bentuk data langsung di tabel (tanpa ALTER)
  -- ESSAY: tidak boleh ada kunci pilihan/benar
  -- SINGLE: answers wajib ada dan berbentuk object/array
  CONSTRAINT ck_quiz_question_essay_shape
    CHECK (
      quiz_question_type <> 'essay'
      OR (quiz_question_answers IS NULL AND quiz_question_correct IS NULL)
    ),
  CONSTRAINT ck_quiz_question_single_answers_required
    CHECK (
      quiz_question_type <> 'single'
      OR quiz_question_answers IS NOT NULL
    ),
  CONSTRAINT ck_quiz_question_single_answers_shape
    CHECK (
      quiz_question_type <> 'single'
      OR jsonb_typeof(quiz_question_answers) IN ('object','array')
    ),

  -- UNIQUE untuk FK komposit (id, quiz_id) — tenant-safe join
  CONSTRAINT uq_quiz_question_id_quiz UNIQUE (quiz_question_id, quiz_question_quiz_id),

  quiz_question_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_question_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_question_deleted_at TIMESTAMPTZ
);

-- Indexes (quiz_questions)

CREATE INDEX IF NOT EXISTS idx_qq_quiz_alive
  ON quiz_questions (quiz_question_quiz_id)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_masjid_alive
  ON quiz_questions (quiz_question_masjid_id)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_masjid_created_desc_alive
  ON quiz_questions (quiz_question_masjid_id, quiz_question_created_at DESC)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_qq_created_at
  ON quiz_questions USING BRIN (quiz_question_created_at);

CREATE INDEX IF NOT EXISTS gin_qq_answers_alive
  ON quiz_questions USING GIN (quiz_question_answers jsonb_path_ops)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS trgm_qq_text_alive
  ON quiz_questions USING GIN ((LOWER(quiz_question_text)) gin_trgm_ops)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_qq_tsv_alive
  ON quiz_questions USING GIN (
    (
      setweight(to_tsvector('simple', COALESCE(quiz_question_text, '')), 'A') ||
      setweight(to_tsvector('simple', COALESCE(quiz_question_explanation, '')), 'B')
    )
  )
  WHERE quiz_question_deleted_at IS NULL;