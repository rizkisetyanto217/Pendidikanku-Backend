BEGIN;

-- =========================================================
-- QUIZ_SETTINGS — kontrol visibilitas & review hasil kuis
--  - Tanpa trigger/backfill; strong FKs; idempotent indexes
--  - Mengatur: apa yang ditampilkan saat/ setelah kuis
-- =========================================================

-- ===== Enums (idempotent) =====
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'quiz_result_visibility_enum') THEN
    CREATE TYPE quiz_result_visibility_enum AS ENUM ('immediate','after_close','never');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'quiz_question_order_enum') THEN
    CREATE TYPE quiz_question_order_enum AS ENUM ('as_created','random');
  END IF;
END$$;

-- ===== Table =====
CREATE TABLE IF NOT EXISTS quiz_settings (
  quiz_settings_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  quiz_settings_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  quiz_settings_quiz_id    UUID NOT NULL
    REFERENCES quizzes(quiz_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Waktu ketersediaan (opsional, untuk “after_close” & window review)
  quiz_settings_open_at    TIMESTAMPTZ,
  quiz_settings_close_at   TIMESTAMPTZ,

  -- Pengendalian pengerjaan
  quiz_settings_time_limit_sec     INT,                 -- override per kuis (jika ingin)
  quiz_settings_shuffle_questions  BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_settings_shuffle_answers    BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_settings_question_order     quiz_question_order_enum NOT NULL DEFAULT 'as_created',

  -- Tampilan saat pengerjaan
  quiz_settings_show_timer         BOOLEAN NOT NULL DEFAULT TRUE,
  quiz_settings_show_progress      BOOLEAN NOT NULL DEFAULT TRUE,
  quiz_settings_lock_question_nav  BOOLEAN NOT NULL DEFAULT FALSE,  -- batasi lompat soal

  -- Visibilitas hasil (kapan skor ditampilkan)
  quiz_settings_result_visibility  quiz_result_visibility_enum NOT NULL DEFAULT 'immediate',
  quiz_settings_show_score         BOOLEAN NOT NULL DEFAULT TRUE,    -- tampilkan skor akhir ke peserta

  -- Perilaku review (apakah peserta boleh melihat rincian setelah submit)
  quiz_settings_allow_review                   BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_settings_review_open_at                 TIMESTAMPTZ,  -- jika ingin window khusus review
  quiz_settings_review_close_at                TIMESTAMPTZ,

  -- Detail apa yang boleh dilihat saat review
  quiz_settings_review_include_questions       BOOLEAN NOT NULL DEFAULT TRUE,   -- teks soal
  quiz_settings_review_include_user_answers    BOOLEAN NOT NULL DEFAULT TRUE,   -- jawaban peserta
  quiz_settings_review_include_correct         BOOLEAN NOT NULL DEFAULT FALSE,  -- jawaban benar
  quiz_settings_review_include_explanations    BOOLEAN NOT NULL DEFAULT FALSE,  -- pembahasan/penjelasan
  quiz_settings_review_include_points          BOOLEAN NOT NULL DEFAULT TRUE,   -- bobot/point per soal
  quiz_settings_review_include_breakdown       BOOLEAN NOT NULL DEFAULT TRUE,   -- rincian benar/salah per soal

  -- Retake/percobaan ulang
  quiz_settings_allow_retake                   BOOLEAN NOT NULL DEFAULT FALSE,
  quiz_settings_retake_limit                   SMALLINT CHECK (quiz_settings_retake_limit IS NULL OR quiz_settings_retake_limit >= 0),
  quiz_settings_min_minutes_between_retake     SMALLINT CHECK (quiz_settings_min_minutes_between_retake IS NULL OR quiz_settings_min_minutes_between_retake >= 0),

  -- Passing threshold opsional (bila tak di assessment)
  quiz_settings_pass_threshold                 NUMERIC(5,2) CHECK (quiz_settings_pass_threshold IS NULL OR (quiz_settings_pass_threshold >= 0 AND quiz_settings_pass_threshold <= 100)),

  -- Audit
  quiz_settings_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_settings_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  quiz_settings_deleted_at TIMESTAMPTZ
);

-- ===== Konsistensi ringan (idempotent) =====
-- Satu baris setting aktif per quiz (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_quiz_settings_quiz_alive
  ON quiz_settings(quiz_settings_quiz_id)
  WHERE quiz_settings_deleted_at IS NULL;

-- Jika result_visibility = 'after_close', close_at sebaiknya diisi (opsional: soft rule via CHECK)
ALTER TABLE quiz_settings DROP CONSTRAINT IF EXISTS ck_qs_after_close_requires_close_at;
ALTER TABLE quiz_settings
  ADD CONSTRAINT ck_qs_after_close_requires_close_at
  CHECK (
    quiz_settings_result_visibility <> 'after_close'
    OR quiz_settings_close_at IS NOT NULL
  );

-- Jika allow_review = true dan tanpa window khusus, tetap valid (window opsional)
-- Jika window dipakai, open <= close
ALTER TABLE quiz_settings DROP CONSTRAINT IF EXISTS ck_qs_review_window_order;
ALTER TABLE quiz_settings
  ADD CONSTRAINT ck_qs_review_window_order
  CHECK (
    quiz_settings_review_open_at IS NULL
    OR quiz_settings_review_close_at IS NULL
    OR quiz_settings_review_open_at <= quiz_settings_review_close_at
  );

-- ===== Indexes =====
CREATE INDEX IF NOT EXISTS idx_qs_masjid_alive
  ON quiz_settings(quiz_settings_masjid_id)
  WHERE quiz_settings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_qs_quiz_alive
  ON quiz_settings(quiz_settings_quiz_id)
  WHERE quiz_settings_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_qs_created_at
  ON quiz_settings USING BRIN (quiz_settings_created_at);

-- Query dukungan: filter review window & after_close
CREATE INDEX IF NOT EXISTS idx_qs_review_window
  ON quiz_settings(quiz_settings_allow_review, quiz_settings_review_open_at, quiz_settings_review_close_at)
  WHERE quiz_settings_deleted_at IS NULL;

COMMIT;
