-- +migrate Down

/* =========================================
   4) USER_QUIZ_ATTEMPT_ANSWERS — DROP
   ========================================= */

-- Indexes (opsional; akan ikut hilang jika tabel di-drop. Disertakan agar eksplisit.)
DROP INDEX IF EXISTS idx_uqaa_need_grading;
DROP INDEX IF EXISTS brin_uqaa_answered_at;
DROP INDEX IF EXISTS idx_uqaa_quiz;
DROP INDEX IF EXISTS idx_uqaa_attempt;
DROP INDEX IF EXISTS idx_uqaa_question;

-- Table (anak; referensi ke user_quiz_attempts & quiz_questions)
DROP TABLE IF EXISTS user_quiz_attempt_answers;



/* =========================================
   3) USER_QUIZ_ATTEMPTS — DROP
   ========================================= */

-- Indexes (opsional; akan ikut hilang jika tabel di-drop. Disertakan agar eksplisit.)
DROP INDEX IF EXISTS brin_uqa_created_at;
DROP INDEX IF EXISTS brin_uqa_started_at;
DROP INDEX IF EXISTS idx_uqa_student_status;
DROP INDEX IF EXISTS idx_uqa_student;
DROP INDEX IF EXISTS idx_uqa_masjid_quiz;
DROP INDEX IF EXISTS idx_uqa_quiz_student_started_desc;
DROP INDEX IF EXISTS idx_uqa_status;
DROP INDEX IF EXISTS idx_uqa_quiz_student;
DROP INDEX IF EXISTS uq_uqa_id_tenant;

-- Table (induk untuk answers)
-- (Constraint uq_uqa_id_quiz adalah table-level constraint dan ikut terhapus)
DROP TABLE IF EXISTS user_quiz_attempts;
