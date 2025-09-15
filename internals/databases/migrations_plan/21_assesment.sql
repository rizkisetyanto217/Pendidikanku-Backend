-- =========================================================
-- UP Migration — Assessment Types & Assessments (Final)
-- =========================================================
BEGIN;

-- -------------------------
-- Prasyarat
-- -------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram (GIN) untuk search

-- =========================================================
-- 1) MASTER: assessment_types
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_types (
  assessment_types_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessment_types_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  assessment_types_key  VARCHAR(32)  NOT NULL,  -- unik per masjid (uas, uts, tugas, ...)
  assessment_types_name VARCHAR(120) NOT NULL,

  assessment_types_weight_percent NUMERIC(5,2) NOT NULL DEFAULT 0
    CHECK (assessment_types_weight_percent >= 0 AND assessment_types_weight_percent <= 100),

  assessment_types_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  assessment_types_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_types_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_types_deleted_at TIMESTAMPTZ
);

-- unik per masjid + key
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_types_masjid_key
  ON assessment_types(assessment_types_masjid_id, assessment_types_key);

-- listing aktif (abaikan yang soft-deleted)
CREATE INDEX IF NOT EXISTS idx_assessment_types_masjid_active
  ON assessment_types(assessment_types_masjid_id, assessment_types_is_active)
  WHERE assessment_types_deleted_at IS NULL;

-- opsional: search nama type (trigram)
CREATE INDEX IF NOT EXISTS idx_gin_trgm_assessment_types_name
  ON assessment_types USING GIN (assessment_types_name gin_trgm_ops);

-- =========================================================
-- 2) GUARD: unique di CSST (tenant-safe)
--    Kombinasi id + masjid untuk validasi aplikasi/service layer
-- =========================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_id_masjid
  ON class_section_subject_teachers (
    class_section_subject_teachers_id,
    class_section_subject_teachers_masjid_id
  );

-- =========================================================
-- 3) ASSESSMENTS — CLEAN RELATION (ONLY TO CSST)
--    Termasuk improvement kolom & index performa
-- =========================================================
CREATE TABLE IF NOT EXISTS assessments (
  assessments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  assessments_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Relasi ke CSST (tenant-safe, validasi via unique di atas)
  assessments_class_section_subject_teacher_id UUID NULL,

  assessments_type_id UUID
    REFERENCES assessment_types(assessment_types_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  assessments_title       VARCHAR(180) NOT NULL,
  assessments_description TEXT,

  -- =============== IMPROVEMENTS (slug, lampiran, policy, dll) ===============
  assessments_slug VARCHAR(160),                    -- untuk URL
  assessments_attachment_url TEXT,                  -- lampiran soal / materi

  -- Jadwal
  assessments_start_at TIMESTAMPTZ,
  assessments_due_at   TIMESTAMPTZ,
  assessments_published_at TIMESTAMPTZ,             -- jadwal rilis
  assessments_closed_at   TIMESTAMPTZ,              -- jadwal tutup

  -- Durasi & attempts (ujian/quiz)
  assessments_duration_minutes INT,                 -- menit
  assessments_total_attempts_allowed INT NOT NULL DEFAULT 1,

  -- Nilai & bobot
  assessments_max_score NUMERIC(5,2) NOT NULL DEFAULT 100
    CHECK (assessments_max_score >= 0 AND assessments_max_score <= 100),
  assessments_min_pass_score NUMERIC(5,2),          -- KKM (opsional)
  assessments_weight_override_percent NUMERIC(5,2)  -- override 0..100 (opsional)
    CHECK (assessments_weight_override_percent IS NULL
           OR (assessments_weight_override_percent >= 0 AND assessments_weight_override_percent <= 100)),

  -- Publikasi & submission
  assessments_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  assessments_allow_submission BOOLEAN NOT NULL DEFAULT TRUE,

  -- Keterlambatan
  assessments_allow_late_submission BOOLEAN NOT NULL DEFAULT FALSE,
  assessments_late_penalty_percent NUMERIC(5,2)
    CHECK (assessments_late_penalty_percent IS NULL
           OR (assessments_late_penalty_percent >= 0 AND assessments_late_penalty_percent <= 100)),

  -- Visibility & grading policy
  assessments_visibility TEXT NOT NULL DEFAULT 'class'
    CHECK (assessments_visibility IN ('public','class','private')),
  assessments_grading_method TEXT NOT NULL DEFAULT 'latest'
    CHECK (assessments_grading_method IN ('highest','latest','average')),

  -- Lifecycle status
  assessments_status TEXT NOT NULL DEFAULT 'draft'
    CHECK (assessments_status IN ('draft','scheduled','open','closed','archived')),

  -- Audit
  assessments_created_by_teacher_id UUID,  -- ke masjid_teachers (opsional)

  assessments_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessments_deleted_at TIMESTAMPTZ
);

-- ===================== INDEX REKOMENDASI =====================

-- Unik slug per masjid (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessments_masjid_slug
  ON assessments(assessments_masjid_id, lower(assessments_slug))
  WHERE assessments_slug IS NOT NULL;

-- List by masjid & published (abaikan soft delete)
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_published
  ON assessments(assessments_masjid_id, assessments_is_published)
  WHERE assessments_deleted_at IS NULL;

-- Sorting by due_at (list tugas/ujian)
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_due_at
  ON assessments(assessments_masjid_id, assessments_due_at)
  WHERE assessments_deleted_at IS NULL;

-- Range by start_at (kalender/agenda)
CREATE INDEX IF NOT EXISTS idx_assessments_masjid_start_at
  ON assessments(assessments_masjid_id, assessments_start_at)
  WHERE assessments_deleted_at IS NULL;

-- Search judul (trigram)
CREATE INDEX IF NOT EXISTS idx_gin_trgm_assessments_title
  ON assessments USING GIN (assessments_title gin_trgm_ops);

-- Visibility + published (listing cepat)
CREATE INDEX IF NOT EXISTS idx_assessments_visibility_published
  ON assessments(assessments_masjid_id, assessments_visibility, assessments_is_published)
  WHERE assessments_deleted_at IS NULL;

-- Grouping by type (aggregate nilai)
CREATE INDEX IF NOT EXISTS idx_assessments_type
  ON assessments(assessments_masjid_id, assessments_type_id)
  WHERE assessments_deleted_at IS NULL;

-- Filter by CSST (mapel/kelas/guru)
CREATE INDEX IF NOT EXISTS idx_assessments_csst
  ON assessments(assessments_masjid_id, assessments_class_section_subject_teacher_id)
  WHERE assessments_deleted_at IS NULL;

-- Status (dashboard) + published/closed (jadwal)
CREATE INDEX IF NOT EXISTS idx_assessments_status
  ON assessments(assessments_masjid_id, assessments_status)
  WHERE assessments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessments_published_closed
  ON assessments(assessments_masjid_id, assessments_published_at, assessments_closed_at)
  WHERE assessments_deleted_at IS NULL;

COMMIT;