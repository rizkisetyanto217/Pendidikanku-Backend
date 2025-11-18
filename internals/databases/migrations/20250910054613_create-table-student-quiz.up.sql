-- +migrate Up
/* =======================================================================
   TABLE: student_quiz_attempts (JSON version)
   1 row = 1 student × 1 quiz (per school)
   - history: semua attempt dalam JSONB
   - best_*: nilai tertinggi
   - last_*: nilai attempt terakhir
   - count  : total attempt
   ======================================================================= */

BEGIN;

CREATE TABLE IF NOT EXISTS student_quiz_attempts (
  -- PK teknis
  student_quiz_attempt_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & identitas
  student_quiz_attempt_school_id  uuid NOT NULL,
  student_quiz_attempt_quiz_id    uuid NOT NULL,
  student_quiz_attempt_student_id uuid NOT NULL,

  -- Semua riwayat attempt (termasuk jawaban) dalam JSONB
  student_quiz_attempt_history jsonb NOT NULL DEFAULT '[]'::jsonb,

  /* ======================= SUMMARY NILAI ======================= */

  -- Total attempt yang pernah dilakukan siswa untuk quiz ini
  student_quiz_attempt_count int NOT NULL DEFAULT 0,

  -- Nilai terbaik di antara semua attempt
  student_quiz_attempt_best_raw     numeric(7,3),
  student_quiz_attempt_best_percent numeric(6,3),

  -- Info attempt terbaik
  student_quiz_attempt_best_started_at  timestamptz,
  student_quiz_attempt_best_finished_at timestamptz,

  -- Nilai attempt terakhir
  student_quiz_attempt_last_raw     numeric(7,3),
  student_quiz_attempt_last_percent numeric(6,3),

  -- Info attempt terakhir (waktu)
  student_quiz_attempt_last_started_at  timestamptz,
  student_quiz_attempt_last_finished_at timestamptz,

  -- Timestamps
  student_quiz_attempt_created_at timestamptz NOT NULL DEFAULT now(),
  student_quiz_attempt_updated_at timestamptz NOT NULL DEFAULT now()
);

-- 1 siswa hanya punya 1 row per quiz di 1 sekolah
ALTER TABLE student_quiz_attempts
  ADD CONSTRAINT uq_sqa_school_quiz_student
  UNIQUE (
    student_quiz_attempt_school_id,
    student_quiz_attempt_quiz_id,
    student_quiz_attempt_student_id
  );

-- Index untuk ranking / daftar nilai per quiz (berdasarkan best_percent)
CREATE INDEX IF NOT EXISTS idx_sqa_best_percent
  ON student_quiz_attempts (
    student_quiz_attempt_school_id,
    student_quiz_attempt_quiz_id,
    student_quiz_attempt_best_percent DESC
  );

-- Optional: index untuk filter by quiz & student dengan cepat (kalau sering dipakai)
CREATE INDEX IF NOT EXISTS idx_sqa_quiz_student
  ON student_quiz_attempts (
    student_quiz_attempt_quiz_id,
    student_quiz_attempt_student_id
  );

-- BRIN index untuk query berdasarkan waktu
CREATE INDEX IF NOT EXISTS brin_sqa_created_at
  ON student_quiz_attempts
  USING BRIN (student_quiz_attempt_created_at);

CREATE INDEX IF NOT EXISTS brin_sqa_last_started_at
  ON student_quiz_attempts
  USING BRIN (student_quiz_attempt_last_started_at);

-- Opsional: index JSONB kalau nanti sering filter ke dalam history
-- CREATE INDEX IF NOT EXISTS gin_sqa_history
--   ON student_quiz_attempts
--   USING GIN (student_quiz_attempt_history);

COMMIT;

-- +migrate Down
-- DROP TABLE IF EXISTS student_quiz_attempts;
