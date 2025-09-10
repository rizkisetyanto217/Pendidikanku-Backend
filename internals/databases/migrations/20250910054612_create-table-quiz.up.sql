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
  ON quizzes(quizzes_masjid_id, quizzes_is_published)
  WHERE quizzes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_quizzes_assessment
  ON quizzes(quizzes_assessment_id)
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
-- 2) QUIZ ITEMS (soal + opsi gabung)
--    SINGLE: 1 soal = banyak baris (1 baris = 1 opsi), tepat 1 opsi benar.
--    ESSAY : 1 soal = 1 baris; kolom opsi = NULL.
-- =========================================
CREATE TABLE IF NOT EXISTS quiz_items (
  quiz_items_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quiz_items_quiz_id UUID NOT NULL
    REFERENCES quizzes(quizzes_id) ON DELETE CASCADE,

  -- Info soal (shared oleh baris-barisnya)
  quiz_items_question_id   UUID NOT NULL,                       -- ID soal (dipakai lintas opsi)
  quiz_items_question_type VARCHAR(8)  NOT NULL                  -- 'single' | 'essay'
    CHECK (quiz_items_question_type IN ('single','essay')),
  quiz_items_question_text TEXT NOT NULL,
  quiz_items_points        NUMERIC(6,2) NOT NULL DEFAULT 1
    CHECK (quiz_items_points >= 0),

  -- Info opsi (hanya untuk SINGLE; NULL untuk ESSAY)
  quiz_items_option_id        UUID,      -- NULL jika ESSAY
  quiz_items_option_text      TEXT,      -- NULL jika ESSAY
  quiz_items_option_is_correct BOOLEAN   -- TRUE tepat satu; NULL jika ESSAY
);

-- Konsistensi bentuk data terhadap tipe soal
ALTER TABLE quiz_items
  ADD CONSTRAINT IF NOT EXISTS ck_quiz_items_shape
  CHECK (
    (quiz_items_question_type = 'single'
      AND quiz_items_option_id IS NOT NULL
      AND quiz_items_option_text IS NOT NULL
      AND quiz_items_option_is_correct IS NOT NULL)
    OR
    (quiz_items_question_type = 'essay'
      AND quiz_items_option_id IS NULL
      AND quiz_items_option_text IS NULL
      AND quiz_items_option_is_correct IS NULL)
  );

-- SINGLE: tepat satu opsi benar per soal
CREATE UNIQUE INDEX IF NOT EXISTS uq_single_correct_per_question
  ON quiz_items(quiz_items_question_id)
  WHERE quiz_items_question_type = 'single'
    AND quiz_items_option_is_correct = TRUE;

-- SINGLE: hindari duplikasi option_id dalam satu soal
CREATE UNIQUE INDEX IF NOT EXISTS uq_question_option_pair
  ON quiz_items(quiz_items_question_id, quiz_items_option_id)
  WHERE quiz_items_question_type = 'single';

-- ESSAY: pastikan hanya satu baris per question_id
CREATE UNIQUE INDEX IF NOT EXISTS uq_essay_single_row_per_question
  ON quiz_items(quiz_items_question_id)
  WHERE quiz_items_question_type = 'essay';

-- Indeks bantu
CREATE INDEX IF NOT EXISTS idx_quiz_items_quiz
  ON quiz_items(quiz_items_quiz_id);

CREATE INDEX IF NOT EXISTS idx_quiz_items_question
  ON quiz_items(quiz_items_question_id);

CREATE INDEX IF NOT EXISTS idx_quiz_items_type
  ON quiz_items(quiz_items_question_type);

-- Optimasi akses umum: ambil baris per kuis & group by soal
CREATE INDEX IF NOT EXISTS idx_quiz_items_quiz_question
  ON quiz_items (quiz_items_quiz_id, quiz_items_question_id);

-- Filter cepat hanya ESSAY per kuis
CREATE INDEX IF NOT EXISTS idx_quiz_items_quiz_essay
  ON quiz_items (quiz_items_quiz_id)
  WHERE quiz_items_question_type = 'essay';



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

-- Indeks bantu attempts (dasar)
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student
  ON user_quiz_attempts(user_quiz_attempts_quiz_id, user_quiz_attempts_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_status
  ON user_quiz_attempts(user_quiz_attempts_status);

CREATE INDEX IF NOT EXISTS brin_uqa_started_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempts_started_at);

-- Optimasi akses umum
-- latest attempt per (quiz, student)
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student_started_desc
  ON user_quiz_attempts (user_quiz_attempts_quiz_id, user_quiz_attempts_student_id, user_quiz_attempts_started_at DESC);

-- akses per tenant (masjid) → per kuis
CREATE INDEX IF NOT EXISTS idx_uqa_masjid_quiz
  ON user_quiz_attempts (user_quiz_attempts_masjid_id, user_quiz_attempts_quiz_id);

-- dashboard siswa/guru
CREATE INDEX IF NOT EXISTS idx_uqa_student
  ON user_quiz_attempts (user_quiz_attempts_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_student_status
  ON user_quiz_attempts (user_quiz_attempts_student_id, user_quiz_attempts_status);

-- attempts aktif saja
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_active
  ON user_quiz_attempts (user_quiz_attempts_quiz_id)
  WHERE user_quiz_attempts_status IN ('in_progress','submitted');

-- time-series tambahan
CREATE INDEX IF NOT EXISTS brin_uqa_created_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempts_created_at);



-- =========================================
-- 4) USER QUIZ ATTEMPT ANSWERS (jawaban per soal)
--    SINGLE → isi selected_option_id.
--    ESSAY  → isi text; dinilai manual oleh guru.
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempt_answers (
  user_quiz_attempt_answers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quiz_attempt_answers_attempt_id UUID NOT NULL
    REFERENCES user_quiz_attempts(user_quiz_attempts_id) ON DELETE CASCADE,

  user_quiz_attempt_answers_question_id UUID NOT NULL,            -- harus cocok dgn question_id di quiz_items

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

-- Indeks bantu answers
CREATE INDEX IF NOT EXISTS idx_user_answers_question
  ON user_quiz_attempt_answers(user_quiz_attempt_answers_question_id);

-- Time-series: answered_at
CREATE INDEX IF NOT EXISTS brin_user_answers_answered_at
  ON user_quiz_attempt_answers USING BRIN (user_quiz_attempt_answers_answered_at);

COMMIT;
