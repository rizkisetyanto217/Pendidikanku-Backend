-- =========================================
-- 3) USER_QUIZ_ATTEMPTS
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempts (
  user_quiz_attempt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quiz_attempt_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_quiz_attempt_quiz_id UUID NOT NULL
    REFERENCES quizzes(quiz_id) ON DELETE CASCADE,

  user_quiz_attempt_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  user_quiz_attempt_started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quiz_attempt_finished_at TIMESTAMPTZ,

  user_quiz_attempt_score_raw     NUMERIC(7,3) DEFAULT 0,
  user_quiz_attempt_score_percent NUMERIC(6,3) DEFAULT 0,

  user_quiz_attempt_status VARCHAR(16) NOT NULL DEFAULT 'in_progress'
    CHECK (user_quiz_attempt_status IN ('in_progress','submitted','finished','abandoned')),

  user_quiz_attempt_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quiz_attempt_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Waktu selesai tidak boleh sebelum mulai (jika diisi)
  CONSTRAINT ck_uqa_time_order
    CHECK (
      user_quiz_attempt_finished_at IS NULL
      OR user_quiz_attempt_finished_at >= user_quiz_attempt_started_at
    ),

  -- Unik komposit (id, quiz_id) untuk FK “tenant-safe” di layer lain
  CONSTRAINT uq_uqa_id_quiz UNIQUE (user_quiz_attempt_id, user_quiz_attempt_quiz_id)
);

-- ========== INDEXES (user_quiz_attempts) ==========
-- Pair unik id+tenant (berguna utk join tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uqa_id_tenant
  ON user_quiz_attempts (user_quiz_attempt_id, user_quiz_attempt_masjid_id);

-- Jalur query umum
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student
  ON user_quiz_attempts (user_quiz_attempt_quiz_id, user_quiz_attempt_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_status
  ON user_quiz_attempts (user_quiz_attempt_status);

CREATE INDEX IF NOT EXISTS idx_uqa_quiz_student_started_desc
  ON user_quiz_attempts (user_quiz_attempt_quiz_id, user_quiz_attempt_student_id, user_quiz_attempt_started_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqa_masjid_quiz
  ON user_quiz_attempts (user_quiz_attempt_masjid_id, user_quiz_attempt_quiz_id);

CREATE INDEX IF NOT EXISTS idx_uqa_student
  ON user_quiz_attempts (user_quiz_attempt_student_id);

CREATE INDEX IF NOT EXISTS idx_uqa_student_status
  ON user_quiz_attempts (user_quiz_attempt_student_id, user_quiz_attempt_status);

-- Attempt yang aktif (in_progress / submitted)
CREATE INDEX IF NOT EXISTS idx_uqa_quiz_active
  ON user_quiz_attempts (user_quiz_attempt_quiz_id)
  WHERE user_quiz_attempt_status IN ('in_progress','submitted');

-- Time-scan
CREATE INDEX IF NOT EXISTS brin_uqa_started_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempt_started_at);

CREATE INDEX IF NOT EXISTS brin_uqa_created_at
  ON user_quiz_attempts USING BRIN (user_quiz_attempt_created_at);



-- =========================================
-- 4) USER_QUIZ_ATTEMPT_ANSWERS (no selected_option_id)
--    (quiz_id diisi oleh backend, tidak pakai trigger)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempt_answers (
  user_quiz_attempt_answer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Diisi oleh backend (bukan trigger)
  user_quiz_attempt_answer_quiz_id UUID NOT NULL,

  user_quiz_attempt_answer_attempt_id  UUID NOT NULL,
  user_quiz_attempt_answer_question_id UUID NOT NULL,

  -- Jawaban user (SINGLE: "A/B/C/D" atau teks opsi; ESSAY: uraian)
  user_quiz_attempt_answer_text TEXT NOT NULL
    CHECK (length(btrim(user_quiz_attempt_answer_text)) > 0),

  -- Hasil penilaian
  user_quiz_attempt_answer_is_correct BOOLEAN,
  user_quiz_attempt_answer_earned_points NUMERIC(6,2) NOT NULL DEFAULT 0
    CHECK (user_quiz_attempt_answer_earned_points >= 0),

  -- Penilaian manual (ESSAY)
  user_quiz_attempt_answer_graded_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,
  user_quiz_attempt_answer_graded_at TIMESTAMPTZ,
  user_quiz_attempt_answer_feedback TEXT,

  user_quiz_attempt_answer_answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- 1 attempt hanya boleh 1 jawaban per soal
  CONSTRAINT uq_uqaa_attempt_question UNIQUE (user_quiz_attempt_answer_attempt_id, user_quiz_attempt_answer_question_id),

  -- FK KOMPOSIT: attempt & question harus pada quiz yang sama
  CONSTRAINT fk_uqaa_attempt_quiz
    FOREIGN KEY (user_quiz_attempt_answer_attempt_id, user_quiz_attempt_answer_quiz_id)
    REFERENCES user_quiz_attempts(user_quiz_attempt_id, user_quiz_attempt_quiz_id)
    ON DELETE CASCADE,

  CONSTRAINT fk_uqaa_question_quiz
    FOREIGN KEY (user_quiz_attempt_answer_question_id, user_quiz_attempt_answer_quiz_id)
    REFERENCES quiz_questions(quiz_question_id, quiz_question_quiz_id)
    ON DELETE CASCADE
);

-- ========== INDEXES (user_quiz_attempt_answers) ==========
CREATE INDEX IF NOT EXISTS idx_uqaa_question
  ON user_quiz_attempt_answers (user_quiz_attempt_answer_question_id);

CREATE INDEX IF NOT EXISTS idx_uqaa_attempt
  ON user_quiz_attempt_answers (user_quiz_attempt_answer_attempt_id);

CREATE INDEX IF NOT EXISTS idx_uqaa_quiz
  ON user_quiz_attempt_answers (user_quiz_attempt_answer_quiz_id);

CREATE INDEX IF NOT EXISTS brin_uqaa_answered_at
  ON user_quiz_attempt_answers USING BRIN (user_quiz_attempt_answer_answered_at);

-- (opsional) cepat untuk penilaian manual yang belum dinilai
CREATE INDEX IF NOT EXISTS idx_uqaa_need_grading
  ON user_quiz_attempt_answers (user_quiz_attempt_answer_graded_at)
  WHERE user_quiz_attempt_answer_graded_at IS NULL;