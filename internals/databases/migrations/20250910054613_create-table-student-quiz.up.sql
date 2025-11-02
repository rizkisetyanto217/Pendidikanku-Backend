-- =========================================
-- 3) STUDENT_QUIZ_ATTEMPTS
-- =========================================
CREATE TABLE IF NOT EXISTS student_quiz_attempts (
  student_quiz_attempt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  student_quiz_attempt_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  student_quiz_attempt_quiz_id UUID NOT NULL
    REFERENCES quizzes(quiz_id) ON DELETE CASCADE,

  student_quiz_attempt_student_id UUID NOT NULL
    REFERENCES school_students(school_student_id) ON DELETE CASCADE,

  student_quiz_attempt_started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_quiz_attempt_finished_at TIMESTAMPTZ,

  student_quiz_attempt_score_raw     NUMERIC(7,3) DEFAULT 0,
  student_quiz_attempt_score_percent NUMERIC(6,3) DEFAULT 0,

  student_quiz_attempt_status VARCHAR(16) NOT NULL DEFAULT 'in_progress'
    CHECK (student_quiz_attempt_status IN ('in_progress','submitted','finished','abandoned')),

  student_quiz_attempt_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_quiz_attempt_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Waktu selesai tidak boleh sebelum mulai (jika diisi)
  CONSTRAINT ck_sqa_time_order
    CHECK (
      student_quiz_attempt_finished_at IS NULL
      OR student_quiz_attempt_finished_at >= student_quiz_attempt_started_at
    ),

  -- Unik komposit (id, quiz_id) untuk FK “tenant-safe” di layer lain
  CONSTRAINT uq_sqa_id_quiz UNIQUE (student_quiz_attempt_id, student_quiz_attempt_quiz_id)
);

-- ========== INDEXES (student_quiz_attempts) ==========
-- Pair unik id+tenant (berguna utk join tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sqa_id_tenant
  ON student_quiz_attempts (student_quiz_attempt_id, student_quiz_attempt_school_id);

-- Jalur query umum
CREATE INDEX IF NOT EXISTS idx_sqa_quiz_student
  ON student_quiz_attempts (student_quiz_attempt_quiz_id, student_quiz_attempt_student_id);

CREATE INDEX IF NOT EXISTS idx_sqa_status
  ON student_quiz_attempts (student_quiz_attempt_status);

CREATE INDEX IF NOT EXISTS idx_sqa_quiz_student_started_desc
  ON student_quiz_attempts (student_quiz_attempt_quiz_id, student_quiz_attempt_student_id, student_quiz_attempt_started_at DESC);

CREATE INDEX IF NOT EXISTS idx_sqa_school_quiz
  ON student_quiz_attempts (student_quiz_attempt_school_id, student_quiz_attempt_quiz_id);

CREATE INDEX IF NOT EXISTS idx_sqa_student
  ON student_quiz_attempts (student_quiz_attempt_student_id);

CREATE INDEX IF NOT EXISTS idx_sqa_student_status
  ON student_quiz_attempts (student_quiz_attempt_student_id, student_quiz_attempt_status);

-- Attempt yang aktif (in_progress / submitted)
CREATE INDEX IF NOT EXISTS idx_sqa_quiz_active
  ON student_quiz_attempts (student_quiz_attempt_quiz_id)
  WHERE student_quiz_attempt_status IN ('in_progress','submitted');

-- Time-scan
CREATE INDEX IF NOT EXISTS brin_sqa_started_at
  ON student_quiz_attempts USING BRIN (student_quiz_attempt_started_at);

CREATE INDEX IF NOT EXISTS brin_sqa_created_at
  ON student_quiz_attempts USING BRIN (student_quiz_attempt_created_at);



-- =========================================
-- 4) STUDENT_QUIZ_ATTEMPT_ANSWERS (no selected_option_id)
--    (quiz_id diisi oleh backend, tidak pakai trigger)
-- =========================================
CREATE TABLE IF NOT EXISTS student_quiz_attempt_answers (
  student_quiz_attempt_answer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Diisi oleh backend (bukan trigger)
  student_quiz_attempt_answer_quiz_id UUID NOT NULL,

  student_quiz_attempt_answer_attempt_id  UUID NOT NULL,
  student_quiz_attempt_answer_question_id UUID NOT NULL,

  -- Jawaban student (SINGLE: "A/B/C/D" atau teks opsi; ESSAY: uraian)
  student_quiz_attempt_answer_text TEXT NOT NULL
    CHECK (length(btrim(student_quiz_attempt_answer_text)) > 0),

  -- Hasil penilaian
  student_quiz_attempt_answer_is_correct BOOLEAN,
  student_quiz_attempt_answer_earned_points NUMERIC(6,2) NOT NULL DEFAULT 0
    CHECK (student_quiz_attempt_answer_earned_points >= 0),

  -- Penilaian manual (ESSAY)
  student_quiz_attempt_answer_graded_by_teacher_id UUID
    REFERENCES school_teachers(school_teacher_id) ON DELETE SET NULL,
  student_quiz_attempt_answer_graded_at TIMESTAMPTZ,
  student_quiz_attempt_answer_feedback TEXT,

  student_quiz_attempt_answer_answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- 1 attempt hanya boleh 1 jawaban per soal
  CONSTRAINT uq_sqaa_attempt_question UNIQUE (student_quiz_attempt_answer_attempt_id, student_quiz_attempt_answer_question_id),

  -- FK KOMPOSIT: attempt & question harus pada quiz yang sama
  CONSTRAINT fk_sqaa_attempt_quiz
    FOREIGN KEY (student_quiz_attempt_answer_attempt_id, student_quiz_attempt_answer_quiz_id)
    REFERENCES student_quiz_attempts(student_quiz_attempt_id, student_quiz_attempt_quiz_id)
    ON DELETE CASCADE,

  CONSTRAINT fk_sqaa_question_quiz
    FOREIGN KEY (student_quiz_attempt_answer_question_id, student_quiz_attempt_answer_quiz_id)
    REFERENCES quiz_questions(quiz_question_id, quiz_question_quiz_id)
    ON DELETE CASCADE
);

-- ========== INDEXES (student_quiz_attempt_answers) ==========
CREATE INDEX IF NOT EXISTS idx_sqaa_question
  ON student_quiz_attempt_answers (student_quiz_attempt_answer_question_id);

CREATE INDEX IF NOT EXISTS idx_sqaa_attempt
  ON student_quiz_attempt_answers (student_quiz_attempt_answer_attempt_id);

CREATE INDEX IF NOT EXISTS idx_sqaa_quiz
  ON student_quiz_attempt_answers (student_quiz_attempt_answer_quiz_id);

CREATE INDEX IF NOT EXISTS brin_sqaa_answered_at
  ON student_quiz_attempt_answers USING BRIN (student_quiz_attempt_answer_answered_at);

-- (opsional) cepat untuk penilaian manual yang belum dinilai
CREATE INDEX IF NOT EXISTS idx_sqaa_need_grading
  ON student_quiz_attempt_answers (student_quiz_attempt_answer_graded_at)
  WHERE student_quiz_attempt_answer_graded_at IS NULL;
