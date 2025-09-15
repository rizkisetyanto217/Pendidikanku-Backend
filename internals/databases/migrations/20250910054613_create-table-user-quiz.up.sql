
-- =========================================
-- 3) USER QUIZ ATTEMPTS
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

-- UNIQUE untuk FK komposit (id, quiz_id)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_uqa_id_quiz') THEN
    ALTER TABLE user_quiz_attempts
      ADD CONSTRAINT uq_uqa_id_quiz UNIQUE (user_quiz_attempts_id, user_quiz_attempts_quiz_id);
  END IF;
END $$;

-- Indexes (user_quiz_attempts)
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
-- 4) USER QUIZ ATTEMPT ANSWERS (no selected_option_id)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quiz_attempt_answers (
  user_quiz_attempt_answers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Diisi otomatis dari attempt via trigger
  user_quiz_attempt_answers_quiz_id UUID,

  user_quiz_attempt_answers_attempt_id UUID NOT NULL,
  user_quiz_attempt_answers_question_id UUID NOT NULL,

  -- Jawaban user (SINGLE: label/teks opsi atau "A/B/C/D"; ESSAY: uraian)
  user_quiz_attempt_answers_text TEXT NOT NULL
    CHECK (length(btrim(user_quiz_attempt_answers_text)) > 0),

  -- Hasil penilaian
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
  UNIQUE (user_quiz_attempt_answers_attempt_id, user_quiz_attempt_answers_question_id)
);

-- Trigger: isi quiz_id otomatis dari attempt
CREATE OR REPLACE FUNCTION uqaa_fill_quiz_id()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.user_quiz_attempt_answers_quiz_id IS NULL THEN
    SELECT user_quiz_attempts_quiz_id
      INTO NEW.user_quiz_attempt_answers_quiz_id
    FROM user_quiz_attempts
    WHERE user_quiz_attempts_id = NEW.user_quiz_attempt_answers_attempt_id;
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_uqaa_fill_quiz_id ON user_quiz_attempt_answers;
CREATE TRIGGER trg_uqaa_fill_quiz_id
BEFORE INSERT ON user_quiz_attempt_answers
FOR EACH ROW EXECUTE FUNCTION uqaa_fill_quiz_id();

-- Wajibkan NOT NULL setelah trigger tersedia
ALTER TABLE user_quiz_attempt_answers
  ALTER COLUMN user_quiz_attempt_answers_quiz_id SET NOT NULL;

-- FK KOMPOSIT: attempt & question harus pada quiz yang sama
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_uqaa_attempt_quiz') THEN
    ALTER TABLE user_quiz_attempt_answers
      ADD CONSTRAINT fk_uqaa_attempt_quiz
      FOREIGN KEY (user_quiz_attempt_answers_attempt_id, user_quiz_attempt_answers_quiz_id)
      REFERENCES user_quiz_attempts(user_quiz_attempts_id, user_quiz_attempts_quiz_id)
      ON DELETE CASCADE;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_uqaa_question_quiz') THEN
    ALTER TABLE user_quiz_attempt_answers
      ADD CONSTRAINT fk_uqaa_question_quiz
      FOREIGN KEY (user_quiz_attempt_answers_question_id, user_quiz_attempt_answers_quiz_id)
      REFERENCES quiz_questions(quiz_questions_id, quiz_questions_quiz_id)
      ON DELETE CASCADE;
  END IF;
END $$;

-- Indexes (answers)
CREATE INDEX IF NOT EXISTS idx_uqaa_question
  ON user_quiz_attempt_answers (user_quiz_attempt_answers_question_id);

CREATE INDEX IF NOT EXISTS idx_uqaa_attempt
  ON user_quiz_attempt_answers (user_quiz_attempt_answers_attempt_id);

CREATE INDEX IF NOT EXISTS idx_uqaa_quiz
  ON user_quiz_attempt_answers (user_quiz_attempt_answers_quiz_id);

CREATE INDEX IF NOT EXISTS brin_uqaa_answered_at
  ON user_quiz_attempt_answers USING BRIN (user_quiz_attempt_answers_answered_at);

COMMIT;
