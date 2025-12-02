-- +migrate Up
-- =========================================
-- UP Migration — Assessments (3 tabel, final)
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ======================================================================
-- ENUM: assessment_score_aggregation_mode_enum
-- ======================================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_type WHERE typname = 'assessment_score_aggregation_mode_enum'
  ) THEN
    CREATE TYPE assessment_score_aggregation_mode_enum AS ENUM (
      'first',    -- pakai nilai attempt pertama
      'latest',   -- pakai nilai attempt terakhir
      'highest',  -- pakai attempt dengan nilai tertinggi
      'average'   -- rata-rata semua attempt
    );
  END IF;
END$$;

-- ======================================================================
-- 1) TABLE: assessment_types
--    - Master setting / preset
--    - Default policy untuk kuis/ujian/tugas
-- ======================================================================
CREATE TABLE IF NOT EXISTS assessment_types (
  assessment_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_type_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  assessment_type_key  VARCHAR(32)  NOT NULL,
  assessment_type_name VARCHAR(120) NOT NULL,

  -- Bobot nilai akhir (0–100)
  assessment_type_weight_percent NUMERIC(5,2)
    NOT NULL DEFAULT 0
    CHECK (
      assessment_type_weight_percent >= 0
      AND assessment_type_weight_percent <= 100
    ),

  -- ============================
  -- DEFAULT QUIZ SETTINGS
  -- ============================

  -- Acak urutan pertanyaan & opsi
  assessment_type_shuffle_questions BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_type_shuffle_options  BOOLEAN NOT NULL DEFAULT FALSE,

  -- Tampilkan jawaban benar / review setelah submit
  assessment_type_show_correct_after_submit BOOLEAN NOT NULL DEFAULT TRUE,

  -- ✅ Strict mode (gantikan one_question_per_page + prevent_back_navigation)
  -- Interpretasi di layer app:
  --   - bisa di-mapping jadi 1 soal per halaman,
  --   - tidak boleh back,
  --   - dll
  assessment_type_strict_mode BOOLEAN NOT NULL DEFAULT FALSE,

  -- Batas waktu (menit); NULL = tanpa batas
  assessment_type_time_limit_min INTEGER
    CHECK (assessment_type_time_limit_min IS NULL OR assessment_type_time_limit_min >= 0),

  -- Maksimal percobaan (attempts); minimal 1
  assessment_type_attempts_allowed INTEGER NOT NULL DEFAULT 1
    CHECK (assessment_type_attempts_allowed >= 1),

  -- Wajib login saat mengerjakan
  assessment_type_require_login BOOLEAN NOT NULL DEFAULT TRUE,

  -- Status aktif type ini
  assessment_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- Type ini menghasilkan nilai (graded) atau cuma hadir / survey
  assessment_type_is_graded BOOLEAN NOT NULL DEFAULT TRUE,

  -- ========== Default Late Policy ==========
  assessment_type_allow_late_submission BOOLEAN NOT NULL DEFAULT FALSE,

  assessment_type_late_penalty_percent NUMERIC(5,2)
    NOT NULL DEFAULT 0
    CHECK (
      assessment_type_late_penalty_percent >= 0
      AND assessment_type_late_penalty_percent <= 100
    ),

  -- Minimal nilai lulus
  assessment_type_passing_score_percent NUMERIC(5,2)
    NOT NULL DEFAULT 0
    CHECK (
      assessment_type_passing_score_percent >= 0
      AND assessment_type_passing_score_percent <= 100
    ),

  -- Cara agregasi nilai kalau ada banyak attempt
  -- (first / latest / highest / average)
  assessment_type_score_aggregation_mode assessment_score_aggregation_mode_enum
    NOT NULL DEFAULT 'latest',

  -- Behavior skor & review
  assessment_type_show_score_after_submit         BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_type_show_correct_after_closed       BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_type_allow_review_before_submit      BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_type_require_complete_attempt        BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_type_show_details_after_all_attempts BOOLEAN NOT NULL DEFAULT FALSE,

  assessment_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_type_deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_id_tenant
  ON assessment_types (assessment_type_id, assessment_type_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_key_per_school_alive
  ON assessment_types (assessment_type_school_id, LOWER(assessment_type_key))
  WHERE assessment_type_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessment_types_school_active
  ON assessment_types (assessment_type_school_id, assessment_type_is_active)
  WHERE assessment_type_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_assessment_types_created_at
  ON assessment_types USING BRIN (assessment_type_created_at);


-- =========================================================
-- 1) ENUM: assessment_kind_enum
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'assessment_kind_enum') THEN
    CREATE TYPE assessment_kind_enum AS ENUM (
      'quiz',
      'assignment_upload',
      'offline',
      'survey'
    );
  END IF;
END$$;

-- =========================================================
-- 2) TABLE: assessments (sinkron dengan AssessmentModel)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessments (
  assessment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Relasi ke CSST (single FK)
  assessment_class_section_subject_teacher_id UUID NULL,

  -- Tipe penilaian (kategori akademik, tenant-safe di-backend)
  assessment_type_id UUID,

  -- Identitas
  assessment_slug  VARCHAR(160),
  assessment_title VARCHAR(180) NOT NULL,
  assessment_description TEXT,

  -- Jadwal by date
  assessment_start_at     TIMESTAMPTZ,
  assessment_due_at       TIMESTAMPTZ,
  assessment_published_at TIMESTAMPTZ,
  assessment_closed_at    TIMESTAMPTZ,

  -- Pengaturan dasar assessment
  assessment_kind assessment_kind_enum NOT NULL DEFAULT 'quiz',
  assessment_duration_minutes       INT,
  assessment_total_attempts_allowed INT NOT NULL DEFAULT 1,
  assessment_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessment_max_score >= 0 AND assessment_max_score <= 100),

  -- total quiz/komponen quiz di assessment ini (global, sama untuk semua siswa)
  assessment_quiz_total SMALLINT NOT NULL DEFAULT 0,

  -- agregat submissions (diupdate dari service)
  assessment_submissions_total       INT NOT NULL DEFAULT 0,
  assessment_submissions_graded_total INT NOT NULL DEFAULT 0,

  assessment_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  -- Flag apakah assessment type ini menghasilkan nilai (graded) — snapshot
  assessment_type_is_graded_snapshot BOOLEAN NOT NULL DEFAULT FALSE,

  -- =========================
  -- Snapshot aturan dari AssessmentType (per assessment)
  -- =========================

  -- Quiz behaviour
  assessment_shuffle_questions_snapshot       BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_shuffle_options_snapshot         BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_show_correct_after_submit_snapshot BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_strict_mode_snapshot             BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_time_limit_min_snapshot          INT,
  assessment_attempts_allowed_snapshot        INT NOT NULL DEFAULT 1,
  assessment_require_login_snapshot           BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_score_aggregation_mode_snapshot  VARCHAR(20) NOT NULL DEFAULT 'latest',

  -- Late policy & visibility
  assessment_allow_late_submission_snapshot         BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_late_penalty_percent_snapshot          NUMERIC(5,2) NOT NULL DEFAULT 0,
  assessment_passing_score_percent_snapshot         NUMERIC(5,2) NOT NULL DEFAULT 0,
  assessment_show_score_after_submit_snapshot       BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_show_correct_after_closed_snapshot     BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_allow_review_before_submit_snapshot    BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_require_complete_attempt_snapshot      BOOLEAN NOT NULL DEFAULT TRUE,
  assessment_show_details_after_all_attempts_snapshot BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit pembuat (opsional)
  assessment_created_by_teacher_id UUID,

  -- Snapshots relasi (CSST & sesi kehadiran)
  assessment_csst_snapshot            JSONB NOT NULL DEFAULT '{}'::jsonb,
  assessment_announce_session_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  assessment_collect_session_snapshot  JSONB NOT NULL DEFAULT '{}'::jsonb,

  -- Mode pengumpulan (by date / by session)
  assessment_submission_mode TEXT NOT NULL DEFAULT 'date'
    CHECK (assessment_submission_mode IN ('date','session')),
  assessment_announce_session_id UUID,
  assessment_collect_session_id  UUID,

  -- Timestamps
  assessment_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- 3) FK KE CSST (single-col saja)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_csst') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_csst
      FOREIGN KEY (assessment_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teacher_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- 4) FK KE assessment_types (single-col saja)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_type') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_type
      FOREIGN KEY (assessment_type_id)
      REFERENCES assessment_types(assessment_type_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- index bantu untuk join cepat ke assessment_types
CREATE INDEX IF NOT EXISTS idx_assessments_type
  ON assessments (assessment_type_id)
  WHERE assessment_deleted_at IS NULL;

-- =========================================================
-- 5) FK OPSIONAL KE class_attendance_sessions
-- =========================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_name = 'class_attendance_sessions')
     AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_announce_session') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_announce_session
      FOREIGN KEY (assessment_announce_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_name = 'class_attendance_sessions')
     AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_assessment_collect_session') THEN
    ALTER TABLE assessments
      ADD CONSTRAINT fk_assessment_collect_session
      FOREIGN KEY (assessment_collect_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- 6) INDEXES assessments
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_id_tenant
  ON assessments (assessment_id, assessment_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_slug_per_tenant_alive
  ON assessments (assessment_school_id, LOWER(assessment_slug))
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_assessments_slug_trgm_alive
  ON assessments USING GIN (LOWER(assessment_slug) gin_trgm_ops)
  WHERE assessment_deleted_at IS NULL
    AND assessment_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_school_created_at
  ON assessments (assessment_school_id, assessment_created_at DESC)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments (assessment_class_section_subject_teacher_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_created_by_teacher
  ON assessments (assessment_created_by_teacher_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_submission_mode_alive
  ON assessments (assessment_school_id, assessment_submission_mode)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_announce_session_alive
  ON assessments (assessment_announce_session_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_collect_session_alive
  ON assessments (assessment_collect_session_id)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_kind_alive
  ON assessments (assessment_school_id, assessment_kind)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_assessments_created_at
  ON assessments USING BRIN (assessment_created_at);

-- Index tambahan untuk agregat submissions
CREATE INDEX IF NOT EXISTS idx_assessments_submissions_total_alive
  ON assessments (assessment_school_id, assessment_submissions_total)
  WHERE assessment_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_submissions_graded_total_alive
  ON assessments (assessment_school_id, assessment_submissions_graded_total)
  WHERE assessment_deleted_at IS NULL;

-- =========================================================
-- 7) ASSESSMENT_URLS (selaras dgn announcement_urls)
--    - tabel plural
--    - kolom singular (prefix assessment_url_)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_urls (
  assessment_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  assessment_url_school_id       UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,
  assessment_url_assessment_id   UUID NOT NULL
    REFERENCES assessments(assessment_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'image','video','attachment','link', dst.)
  assessment_url_kind            VARCHAR(24) NOT NULL,

  -- Lokasi file/link (skema dua-slot + retensi)
  assessment_url                    TEXT,
  assessment_url_object_key         TEXT,
  assessment_url_old                TEXT,
  assessment_url_object_key_old     TEXT,
  assessment_url_delete_pending_until TIMESTAMPTZ,

  -- Tampilan
  assessment_url_label           VARCHAR(160),
  assessment_url_order           INT NOT NULL DEFAULT 0,
  assessment_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  assessment_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_url_deleted_at      TIMESTAMPTZ
);

-- Pair unik id+tenant (tenant-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_id_tenant
  ON assessment_urls (assessment_url_id, assessment_url_school_id);

-- Lookup per assessment (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_owner_live
  ON assessment_urls (
    assessment_url_assessment_id,
    assessment_url_kind,
    assessment_url_is_primary DESC,
    assessment_url_order,
    assessment_url_created_at
  )
  WHERE assessment_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_by_school_live
  ON assessment_urls (assessment_url_school_id)
  WHERE assessment_url_deleted_at IS NULL;

-- Satu primary per (assessment, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_primary_per_kind_alive
  ON assessment_urls (assessment_url_assessment_id, assessment_url_kind)
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url_is_primary = TRUE;

-- Anti-duplikat URL per assessment (live only; case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_urls_assessment_url_alive
  ON assessment_urls (assessment_url_assessment_id, LOWER(assessment_url))
  WHERE assessment_url_deleted_at IS NULL
    AND assessment_url IS NOT NULL;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_assessment_urls_purge_due
  ON assessment_urls (assessment_url_delete_pending_until)
  WHERE assessment_url_delete_pending_until IS NOT NULL
    AND (
      (assessment_url_deleted_at IS NULL  AND assessment_url_object_key_old IS NOT NULL) OR
      (assessment_url_deleted_at IS NOT NULL AND assessment_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_assessment_urls_label_trgm_live
  ON assessment_urls USING GIN (assessment_url_label gin_trgm_ops)
  WHERE assessment_url_deleted_at IS NULL;

COMMIT;
