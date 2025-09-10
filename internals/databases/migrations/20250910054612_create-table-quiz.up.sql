-- =========================================
-- UP Migration — TABLES + INDEX OPTIMIZATION
-- =========================================
BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- 1) QUIZZES (inti kuis)
-- =========================================
CREATE TABLE IF NOT EXISTS quizzes (
  quizzes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quizzes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- opsional: link ke assessment agar nilai bisa disinkron via backend
  quizzes_assessment_id UUID
    REFERENCES assessments(assessments_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  quizzes_title VARCHAR(180) NOT NULL,
  quizzes_description TEXT,
  quizzes_is_published BOOLEAN NOT NULL DEFAULT FALSE,
  quizzes_time_limit_sec INT,

  quizzes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quizzes_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quizzes_deleted_at TIMESTAMPTZ
);

-- Indeks bantu (dasar)
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_published
  ON quizzes (quizzes_masjid_id, quizzes_is_published)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_assessment
  ON quizzes (quizzes_assessment_id)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_quizzes_created_at
  ON quizzes USING BRIN (quizzes_created_at);

-- Optimasi pencarian judul/deskripsi (ILIKE)
CREATE INDEX IF NOT EXISTS gin_quizzes_title_trgm
  ON quizzes USING GIN (quizzes_title gin_trgm_ops)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_quizzes_desc_trgm
  ON quizzes USING GIN (quizzes_description gin_trgm_ops)
  WHERE quizzes_deleted_at IS NULL;

-- Akses umum multi-tenant & sort terbaru
CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_assessment
  ON quizzes (quizzes_masjid_id, quizzes_assessment_id)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_masjid_created_desc
  ON quizzes (quizzes_masjid_id, quizzes_created_at DESC)
  WHERE quizzes_deleted_at IS NULL;



-- =========================================
-- 2) QUIZ QUESTIONS (soal + opsi via JSONB)
-- =========================================
CREATE TABLE IF NOT EXISTS quiz_questions (
  quiz_questions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  quiz_questions_quiz_id   UUID NOT NULL
    REFERENCES quizzes(quizzes_id) ON DELETE CASCADE,

  -- denormalisasi tenant (untuk filter cepat)
  quiz_questions_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- tipe soal
  quiz_questions_type VARCHAR(8) NOT NULL
    CHECK (quiz_questions_type IN ('single','essay')),

  quiz_questions_text   TEXT NOT NULL,
  quiz_questions_points NUMERIC(6,2) NOT NULL DEFAULT 1
    CHECK (quiz_questions_points >= 0),

  -- opsi/jawaban:
  --  - SINGLE: JSONB (object A..D ATAU array elemen {text,is_correct,...})
  --  - ESSAY : NULL
  quiz_questions_answers JSONB,
  quiz_questions_correct CHAR(1)
    CHECK (quiz_questions_correct IN ('A','B','C','D')),

  quiz_questions_explanation TEXT,

  quiz_questions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_questions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_questions_deleted_at TIMESTAMPTZ
);

-- ===============================
-- CHECK constraints (tanpa subquery)
-- ===============================
-- Bersihkan dulu biar idempotent
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_essay_shape;
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_single_answers_required;
ALTER TABLE quiz_questions DROP CONSTRAINT IF EXISTS ck_qq_single_answers_shape;

-- ESSAY: answers & correct harus NULL
ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_essay_shape
  CHECK (
    quiz_questions_type <> 'essay'
    OR (quiz_questions_answers IS NULL AND quiz_questions_correct IS NULL)
  );

-- SINGLE: answers wajib ada
ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_single_answers_required
  CHECK (
    quiz_questions_type <> 'single'
    OR quiz_questions_answers IS NOT NULL
  );

-- SINGLE: bentuk jawaban harus object ATAU array (tanpa subquery)
ALTER TABLE quiz_questions
  ADD CONSTRAINT ck_qq_single_answers_shape
  CHECK (
    quiz_questions_type <> 'single'
    OR jsonb_typeof(quiz_questions_answers) IN ('object','array')
  );

-- ===============================
-- Indexes (partial: hanya row hidup)
-- ===============================
-- Filter umum
CREATE INDEX IF NOT EXISTS idx_qq_quiz
  ON quiz_questions (quiz_questions_quiz_id)
  WHERE quiz_questions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qq_masjid
  ON quiz_questions (quiz_questions_masjid_id)
  WHERE quiz_questions_deleted_at IS NULL;

-- Sort terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_qq_masjid_created_desc
  ON quiz_questions (quiz_questions_masjid_id, quiz_questions_created_at DESC)
  WHERE quiz_questions_deleted_at IS NULL;

-- Time-series
CREATE INDEX IF NOT EXISTS brin_qq_created_at
  ON quiz_questions USING BRIN (quiz_questions_created_at);

-- GIN untuk JSONB answers
CREATE INDEX IF NOT EXISTS gin_qq_answers
  ON quiz_questions USING GIN (quiz_questions_answers jsonb_path_ops)
  WHERE quiz_questions_deleted_at IS NULL;

-- Trigram untuk pencarian teks (wajib double parentheses + operator class)
CREATE INDEX IF NOT EXISTS trgm_qq_text
  ON quiz_questions USING GIN ((LOWER(quiz_questions_text)) gin_trgm_ops)
  WHERE quiz_questions_deleted_at IS NULL;

-- Full-text search index (expression)
CREATE INDEX IF NOT EXISTS gin_qq_tsv
  ON quiz_questions USING GIN (
    (
      setweight(to_tsvector('simple', COALESCE(quiz_questions_text, '')), 'A') ||
      setweight(to_tsvector('simple', COALESCE(quiz_questions_explanation, '')), 'B')
    )
  )
  WHERE quiz_questions_deleted_at IS NULL;



-- =========================================
-- 3) USER QUIZ ATTEMPTS (sesi pengerjaan siswa)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempts (
  user_quiz_attempts_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quiz_attempts_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_quiz_attempts_quiz_id UUID NOT NULL
    REFERENCES quizzes(quizzes_id) ON DELETE CASCADE,

  user_quiz_attempts_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  user_quiz_attempts_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quiz_attempts_finished_at TIMESTAMPTZ,

  user_quiz_attempts_score_raw NUMERIC(7,3) DEFAULT 0,
  user_quiz_attempts_score_percent NUMERIC(6,3) DEFAULT 0,

  user_quiz_attempts_status VARCHAR(16) NOT NULL DEFAULT 'in_progress'
    CHECK (user_quiz_attempts_status IN ('in_progress','submitted','finished','abandoned')),

  user_quiz_attempts_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quiz_attempts_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index attempts
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student
  ON user_quiz_attempts (user_quiz_attempts_quiz_id, user_quiz_attempts_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_status
  ON user_quiz_attempts (user_quiz_attempts_status);

CREATE INDEX IF NOT EXISTS brin_uqa_started_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempts_started_at);

CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student_started_desc
  ON user_quiz_attempts (user_quiz_attempts_quiz_id, user_quiz_attempts_student_id, user_quiz_attempts_started_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqa_masjid_quiz
  ON user_quiz_attempts (user_quiz_attempts_masjid_id, user_quiz_attempts_quiz_id);

CREATE INDEX IF NOT EXISTS idx_uqa_student
  ON user_quiz_attempts (user_quiz_attempts_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_student_status
  ON user_quiz_attempts (user_quiz_attempts_student_id, user_quiz_attempts_status);

CREATE INDEX IF NOT EXISTS idx_uqa_quiz_active
  ON user_quiz_attempts (user_quiz_attempts_quiz_id)
  WHERE user_quiz_attempts_status IN ('in_progress','submitted');

CREATE INDEX IF NOT EXISTS brin_uqa_created_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempts_created_at);



-- =========================================
-- 4) USER QUIZ ATTEMPT ANSWERS (jawaban per soal)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempt_answers (
  user_quiz_attempt_answers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quiz_attempt_answers_attempt_id UUID NOT NULL
    REFERENCES user_quiz_attempts(user_quiz_attempts_id) ON DELETE CASCADE,

  -- harus cocok dgn quiz_questions_id
  user_quiz_attempt_answers_question_id UUID NOT NULL,

  -- SINGLE: isi option; ESSAY: NULL
  user_quiz_attempt_answers_selected_option_id UUID,

  -- ESSAY: isi text; SINGLE: NULL
  user_quiz_attempt_answers_text TEXT,

  -- Hasil penilaian (SINGLE auto via backend; ESSAY manual)
  user_quiz_attempt_answers_is_correct BOOLEAN,
  user_quiz_attempt_answers_earned_points NUMERIC(6,2) NOT NULL DEFAULT 0
    CHECK (user_quiz_attempt_answers_earned_points >= 0),

  -- Penilaian manual (ESSAY)
  user_quiz_attempt_answers_graded_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,
  user_quiz_attempt_answers_graded_at TIMESTAMPTZ,
  user_quiz_attempt_answers_feedback TEXT,

  user_quiz_attempt_answers_answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- 1 attempt hanya boleh 1 jawaban per soal
  UNIQUE (user_quiz_attempt_answers_attempt_id, user_quiz_attempt_answers_question_id),

  -- Pastikan tepat satu diisi: option ATAU text
  CONSTRAINT ck_user_answers_xor_content
    CHECK (
      (user_quiz_attempt_answers_selected_option_id IS NOT NULL AND user_quiz_attempt_answers_text IS NULL)
      OR
      (user_quiz_attempt_answers_selected_option_id IS NULL AND user_quiz_attempt_answers_text IS NOT NULL)
    )
);

-- Index answers
CREATE INDEX IF NOT EXISTS idx_user_answers_question
  ON user_quiz_attempt_answers (user_quiz_attempt_answers_question_id);

CREATE INDEX IF NOT EXISTS brin_user_answers_answered_at
  ON user_quiz_attempt_answers USING BRIN (user_quiz_attempt_answers_answered_at);

COMMIT;
