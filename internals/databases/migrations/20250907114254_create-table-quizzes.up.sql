-- =========================================
-- UP Migration — TABLES + STRONG FKs (no selected_option_id)
-- Fresh create (tanpa ALTER / DO blocks)
-- =========================================

-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- 1) QUIZZES (FINAL)
-- =========================================
CREATE TABLE IF NOT EXISTS quizzes (
  quiz_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  quiz_assessment_id UUID
    REFERENCES assessments(assessment_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- SLUG (opsional; unik per tenant saat alive)
  quiz_slug VARCHAR(160),

  quiz_title          VARCHAR(180) NOT NULL,
  quiz_description    TEXT,
  quiz_is_published   BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_time_limit_sec INT,

  -- ==============================
  -- Snapshot quiz behaviour & scoring
  -- (dipindah dari AssessmentType/assessments)
  -- ==============================

  -- tampilan & UX pengerjaan
  quiz_shuffle_questions_snapshot               BOOLEAN      NOT NULL DEFAULT FALSE,
  quiz_shuffle_options_snapshot                 BOOLEAN      NOT NULL DEFAULT FALSE,
  quiz_show_correct_after_submit_snapshot       BOOLEAN      NOT NULL DEFAULT TRUE,
  quiz_strict_mode_snapshot                     BOOLEAN      NOT NULL DEFAULT FALSE,
  quiz_time_limit_min_snapshot                  INT,
  quiz_require_login_snapshot                   BOOLEAN      NOT NULL DEFAULT TRUE,
  quiz_show_score_after_submit_snapshot         BOOLEAN      NOT NULL DEFAULT TRUE,
  quiz_show_correct_after_closed_snapshot       BOOLEAN      NOT NULL DEFAULT FALSE,
  quiz_allow_review_before_submit_snapshot      BOOLEAN      NOT NULL DEFAULT TRUE,
  quiz_require_complete_attempt_snapshot        BOOLEAN      NOT NULL DEFAULT TRUE,
  quiz_show_details_after_all_attempts_snapshot BOOLEAN      NOT NULL DEFAULT FALSE,

  -- attempts & agregasi nilai (final score dari attempts quiz)
  quiz_attempts_allowed_snapshot                INT          NOT NULL DEFAULT 1,
  quiz_score_aggregation_mode_snapshot          VARCHAR(20)  NOT NULL DEFAULT 'latest',

  quiz_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_deleted_at TIMESTAMPTZ
);

-- Indexes / Optimizations (quizzes)

-- SLUG unik per tenant (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_quizzes_slug_per_tenant_alive
  ON quizzes (quiz_school_id, LOWER(quiz_slug))
  WHERE quiz_deleted_at IS NULL
    AND quiz_slug IS NOT NULL;

-- (opsional) pencarian slug cepat (trigram, alive only)
CREATE INDEX IF NOT EXISTS gin_quizzes_slug_trgm_alive
  ON quizzes USING GIN (LOWER(quiz_slug) gin_trgm_ops)
  WHERE quiz_deleted_at IS NULL
    AND quiz_slug IS NOT NULL;

-- pair unik id+tenant (tenant-safe FK di masa depan)
CREATE UNIQUE INDEX IF NOT EXISTS uq_quizzes_id_tenant
  ON quizzes (quiz_id, quiz_school_id);

-- Publikasi per tenant (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_school_published
  ON quizzes (quiz_school_id, quiz_is_published)
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
CREATE INDEX IF NOT EXISTS idx_quizzes_school_assessment
  ON quizzes (quiz_school_id, quiz_assessment_id)
  WHERE quiz_deleted_at IS NULL;

-- Listing terbaru per tenant (alive only)
CREATE INDEX IF NOT EXISTS idx_quizzes_school_created_desc
  ON quizzes (quiz_school_id, quiz_created_at DESC)
  WHERE quiz_deleted_at IS NULL;



-- =========================================
-- 2) QUIZ_QUESTIONS
-- =========================================
CREATE TABLE IF NOT EXISTS quiz_questions (
  quiz_question_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  quiz_question_quiz_id   UUID NOT NULL
    REFERENCES quizzes(quiz_id) ON DELETE CASCADE,

  quiz_question_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  quiz_question_type VARCHAR(8) NOT NULL
    CHECK (quiz_question_type IN ('single','essay')),

  quiz_question_text   TEXT NOT NULL,
  quiz_question_points NUMERIC(6,2) NOT NULL DEFAULT 1
    CHECK (quiz_question_points >= 0),

  -- JSON berisi opsi jawaban (A, B, C, D, E, ...)
  -- Disarankan bentuk object: { "A": "...", "B": "...", ... }
  quiz_question_answers JSONB,

  -- Huruf / key jawaban benar (harus salah satu key di quiz_question_answers)
  quiz_question_correct TEXT,

  -- Pembahasan / penjelasan
  quiz_question_explanation TEXT,

  /* =============================
     VERSIONING + HISTORY
     ============================= */

  -- Versi aktif saat ini (mulai dari 1, tiap edit +1 di service)
  quiz_question_version INT NOT NULL DEFAULT 1,

  -- Riwayat versi sebelumnya (array of snapshot)
  -- Contoh isi:
  -- [
  --   {
  --     "version": 1,
  --     "saved_at": "2025-11-30T09:00:00Z",
  --     "text": "...",
  --     "answers": {...},
  --     "correct": "C",
  --     "explanation": "...",
  --     "points": 1
  --   }
  -- ]
  quiz_question_history JSONB NOT NULL DEFAULT '[]'::jsonb,

  /* =============================
     CONSTRAINT SHAPE & LOGIC
     ============================= */

  -- ESSAY: tidak boleh punya answers & correct
  CONSTRAINT ck_quiz_question_essay_shape
    CHECK (
      quiz_question_type <> 'essay'
      OR (quiz_question_answers IS NULL AND quiz_question_correct IS NULL)
    ),

  -- SINGLE: answers wajib ada
  CONSTRAINT ck_quiz_question_single_answers_required
    CHECK (
      quiz_question_type <> 'single'
      OR quiz_question_answers IS NOT NULL
    ),

  -- SINGLE: answers harus berupa object (biar key "A","B","C", dst jelas)
  CONSTRAINT ck_quiz_question_single_answers_shape
    CHECK (
      quiz_question_type <> 'single'
      OR jsonb_typeof(quiz_question_answers) = 'object'
    ),

  -- SINGLE: kunci jawaban wajib salah satu key di answers
  CONSTRAINT ck_quiz_question_single_correct_in_answers
    CHECK (
      quiz_question_type <> 'single'
      OR (
        quiz_question_correct IS NOT NULL
        AND quiz_question_answers ? quiz_question_correct
      )
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

CREATE INDEX IF NOT EXISTS idx_qq_school_alive
  ON quiz_questions (quiz_question_school_id)
  WHERE quiz_question_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_school_created_desc_alive
  ON quiz_questions (quiz_question_school_id, quiz_question_created_at DESC)
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
