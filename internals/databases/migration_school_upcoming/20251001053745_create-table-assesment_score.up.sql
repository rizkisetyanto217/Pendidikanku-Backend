-- +migrate Up
-- =====================================================================
-- UP — Assessment Snapshots (Unified, Partitioned by Scope, No ALTER)
-- Opsi B: 'assessment' di tiap scope, anchor wajib & unique alive ikut anchor
-- =====================================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================================================
-- ENUMs
-- =========================================================
DO $$ BEGIN
  CREATE TYPE assessment_snapshots_scope AS ENUM ('ucsst','section','school');
EXCEPTION WHEN duplicate_object THEN END $$;

DO $$ BEGIN
  CREATE TYPE assessment_snapshots_grain AS ENUM ('week','month','semester');
EXCEPTION WHEN duplicate_object THEN END $$;

-- =========================================================
-- PARENT (Partitioned by scope)
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_snapshots (
  assessment_snapshots_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant
  assessment_snapshots_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Partition key
  assessment_snapshots_scope assessment_snapshots_scope NOT NULL,

  -- Bentuk waktu:
  -- 'assessment' (satu assessment) | 'week' | 'month' | 'semester'
  assessment_snapshots_period_type VARCHAR(12) NOT NULL
    CHECK (assessment_snapshots_period_type IN ('assessment','week','month','semester')),

  -- Anchors (tergantung scope & period_type; divalidasi di partisi)
  assessment_snapshots_assessment_id UUID, -- wajib jika period_type='assessment'
  assessment_snapshots_ucsst_id      UUID, -- wajib utk scope=ucsst pada period (week|month|semester)
  assessment_snapshots_section_id    UUID, -- wajib utk scope=section pada period (week|month|semester)

  -- Period info (untuk week/month/semester)
  assessment_snapshots_period_start DATE,
  assessment_snapshots_period_end   DATE,
  assessment_snapshots_period_label VARCHAR(100), -- contoh '2025-S1' (wajib untuk semester; dicek di partisi)

  -- =========================
  -- Aggregates
  -- =========================
  -- Counters
  assessment_snapshots_total_enrolled    INT NOT NULL DEFAULT 0 CHECK (assessment_snapshots_total_enrolled    >= 0),
  assessment_snapshots_submissions_count INT NOT NULL DEFAULT 0 CHECK (assessment_snapshots_submissions_count >= 0),
  assessment_snapshots_graded_count      INT NOT NULL DEFAULT 0 CHECK (assessment_snapshots_graded_count      >= 0),

  -- Missing (generated)
  assessment_snapshots_missing_count INT GENERATED ALWAYS AS (
    GREATEST(assessment_snapshots_total_enrolled - assessment_snapshots_submissions_count, 0)
  ) STORED,

  -- Scores
  assessment_snapshots_score_total_sum   NUMERIC(18,3) NOT NULL DEFAULT 0,
  assessment_snapshots_score_percent_avg NUMERIC(8,3)
    CHECK (assessment_snapshots_score_percent_avg IS NULL OR assessment_snapshots_score_percent_avg BETWEEN 0 AND 100),
  assessment_snapshots_score_percent_min NUMERIC(6,3)
    CHECK (assessment_snapshots_score_percent_min IS NULL OR assessment_snapshots_score_percent_min BETWEEN 0 AND 100),
  assessment_snapshots_score_percent_max NUMERIC(6,3)
    CHECK (assessment_snapshots_score_percent_max IS NULL OR assessment_snapshots_score_percent_max BETWEEN 0 AND 100),

  -- Late
  assessment_snapshots_late_count      INT NOT NULL DEFAULT 0 CHECK (assessment_snapshots_late_count >= 0),
  assessment_snapshots_late_rate_pct   NUMERIC(6,3) GENERATED ALWAYS AS (
    CASE
      WHEN assessment_snapshots_submissions_count > 0
        THEN ROUND((assessment_snapshots_late_count::numeric * 100.0) / assessment_snapshots_submissions_count, 3)
      ELSE 0
    END
  ) STORED CHECK (assessment_snapshots_late_rate_pct BETWEEN 0 AND 100),
  assessment_snapshots_late_minute_avg NUMERIC(10,3)
    CHECK (assessment_snapshots_late_minute_avg IS NULL OR assessment_snapshots_late_minute_avg >= 0),
  assessment_snapshots_late_minute_p95 NUMERIC(10,3)
    CHECK (assessment_snapshots_late_minute_p95 IS NULL OR assessment_snapshots_late_minute_p95 >= 0),

  -- Remedial
  assessment_snapshots_remedial_count        INT NOT NULL DEFAULT 0 CHECK (assessment_snapshots_remedial_count >= 0),
  assessment_snapshots_remedial_rate_pct     NUMERIC(6,3) GENERATED ALWAYS AS (
    CASE
      WHEN assessment_snapshots_submissions_count > 0
        THEN ROUND((assessment_snapshots_remedial_count::numeric * 100.0) / assessment_snapshots_submissions_count, 3)
      ELSE 0
    END
  ) STORED CHECK (assessment_snapshots_remedial_rate_pct BETWEEN 0 AND 100),
  assessment_snapshots_remedial_attempts_avg NUMERIC(8,3)
    CHECK (assessment_snapshots_remedial_attempts_avg IS NULL OR assessment_snapshots_remedial_attempts_avg >= 0),
  assessment_snapshots_remedial_attempts_max INT
    CHECK (assessment_snapshots_remedial_attempts_max IS NULL OR assessment_snapshots_remedial_attempts_max >= 0),

  -- Improvement
  assessment_snapshots_score_improve_avg NUMERIC(8,3),
  assessment_snapshots_score_improve_max NUMERIC(8,3),

  -- JSONB
  assessment_snapshots_status_counts JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(assessment_snapshots_status_counts) = 'object'),
  assessment_snapshots_grade_hist JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(assessment_snapshots_grade_hist) = 'object'),
  assessment_snapshots_meta JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(assessment_snapshots_meta) = 'object'),

  -- Audit waktu submit
  assessment_snapshots_first_submitted_at TIMESTAMPTZ,
  assessment_snapshots_last_submitted_at  TIMESTAMPTZ,

  -- Finalization
  assessment_snapshots_is_final     BOOLEAN NOT NULL DEFAULT FALSE,
  assessment_snapshots_finalized_at TIMESTAMPTZ,

  -- Timestamps
  assessment_snapshots_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_snapshots_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assessment_snapshots_deleted_at TIMESTAMPTZ
)
PARTITION BY LIST (assessment_snapshots_scope);

-- =========================================================
-- PARTITION: UCSST
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_snapshots_ucsst
  PARTITION OF assessment_snapshots FOR VALUES IN ('ucsst')
(
  -- Shape rules:
  -- - 'assessment'  → wajib (assessment_id AND ucsst_id)
  -- - period (week|month|semester) → wajib (ucsst_id, period_start, period_end), label wajib utk semester
  CONSTRAINT ck_assessment_snapshots_ucsst_shape CHECK (
    (
      assessment_snapshots_period_type = 'assessment'
      AND assessment_snapshots_assessment_id IS NOT NULL
      AND assessment_snapshots_ucsst_id      IS NOT NULL
    )
    OR
    (
      assessment_snapshots_period_type IN ('week','month','semester')
      AND assessment_snapshots_ucsst_id      IS NOT NULL
      AND assessment_snapshots_period_start  IS NOT NULL
      AND assessment_snapshots_period_end    IS NOT NULL
      AND assessment_snapshots_period_start <= assessment_snapshots_period_end
      AND (
        assessment_snapshots_period_type <> 'semester'
        OR assessment_snapshots_period_label IS NOT NULL
      )
    )
  ),

  -- FKs
  CONSTRAINT fk_assessment_snapshots_ucsst_assessment_tenant
    FOREIGN KEY (assessment_snapshots_assessment_id, assessment_snapshots_school_id)
    REFERENCES assessments(assessment_id, assessment_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_assessment_snapshots_ucsst_ucsst
    FOREIGN KEY (assessment_snapshots_ucsst_id)
    REFERENCES user_class_section_subject_teachers(user_class_section_subject_teacher_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Exclusion: final snapshots tidak overlap per (tenant, ucsst, tipe, rentang)
  CONSTRAINT assessment_snapshots_ucsst_final_no_overlap EXCLUDE USING gist (
    assessment_snapshots_school_id WITH =,
    assessment_snapshots_ucsst_id  WITH =,
    assessment_snapshots_period_type WITH =,
    daterange(assessment_snapshots_period_start, assessment_snapshots_period_end, '[]') WITH &&
  ) WHERE (assessment_snapshots_is_final)
);

-- Unique (alive) — assessment per scope (ikut anchor)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_ucsst_alive_assessment
  ON assessment_snapshots_ucsst (assessment_snapshots_school_id, assessment_snapshots_assessment_id, assessment_snapshots_ucsst_id)
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type = 'assessment';

-- Unique (alive) — period per scope
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_ucsst_alive_period
  ON assessment_snapshots_ucsst (
    assessment_snapshots_school_id,
    assessment_snapshots_ucsst_id,
    assessment_snapshots_period_type,
    assessment_snapshots_period_start,
    assessment_snapshots_period_end
  )
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type IN ('week','month','semester');

-- =========================================================
-- PARTITION: SECTION
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_snapshots_section
  PARTITION OF assessment_snapshots FOR VALUES IN ('section')
(
  -- Shape rules:
  -- - 'assessment'  → wajib (assessment_id AND section_id)
  -- - period (week|month|semester) → wajib (section_id, period_start, period_end), label wajib utk semester
  CONSTRAINT ck_assessment_snapshots_section_shape CHECK (
    (
      assessment_snapshots_period_type = 'assessment'
      AND assessment_snapshots_assessment_id IS NOT NULL
      AND assessment_snapshots_section_id    IS NOT NULL
    )
    OR
    (
      assessment_snapshots_period_type IN ('week','month','semester')
      AND assessment_snapshots_section_id    IS NOT NULL
      AND assessment_snapshots_period_start  IS NOT NULL
      AND assessment_snapshots_period_end    IS NOT NULL
      AND assessment_snapshots_period_start <= assessment_snapshots_period_end
      AND (
        assessment_snapshots_period_type <> 'semester'
        OR assessment_snapshots_period_label IS NOT NULL
      )
    )
  ),

  -- FKs
  CONSTRAINT fk_assessment_snapshots_section_assessment_tenant
    FOREIGN KEY (assessment_snapshots_assessment_id, assessment_snapshots_school_id)
    REFERENCES assessments(assessment_id, assessment_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_assessment_snapshots_section_section
    FOREIGN KEY (assessment_snapshots_section_id)
    REFERENCES class_sections(class_section_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Exclusion: final snapshots tidak overlap per (tenant, section, tipe, rentang)
  CONSTRAINT assessment_snapshots_section_final_no_overlap EXCLUDE USING gist (
    assessment_snapshots_school_id WITH =,
    assessment_snapshots_section_id WITH =,
    assessment_snapshots_period_type WITH =,
    daterange(assessment_snapshots_period_start, assessment_snapshots_period_end, '[]') WITH &&
  ) WHERE (assessment_snapshots_is_final)
);

-- Unique (alive) — assessment per scope (ikut anchor)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_section_alive_assessment
  ON assessment_snapshots_section (assessment_snapshots_school_id, assessment_snapshots_assessment_id, assessment_snapshots_section_id)
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type = 'assessment';

-- Unique (alive) — period per scope
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_section_alive_period
  ON assessment_snapshots_section (
    assessment_snapshots_school_id,
    assessment_snapshots_section_id,
    assessment_snapshots_period_type,
    assessment_snapshots_period_start,
    assessment_snapshots_period_end
  )
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type IN ('week','month','semester');

-- =========================================================
-- PARTITION: MASJID
-- =========================================================
CREATE TABLE IF NOT EXISTS assessment_snapshots_school
  PARTITION OF assessment_snapshots FOR VALUES IN ('school')
(
  -- Shape rules:
  -- - 'assessment' → wajib (assessment_id)
  -- - period → wajib (period_start, period_end), label wajib utk semester
  CONSTRAINT ck_assessment_snapshots_school_shape CHECK (
    (
      assessment_snapshots_period_type = 'assessment'
      AND assessment_snapshots_assessment_id IS NOT NULL
    )
    OR
    (
      assessment_snapshots_period_type IN ('week','month','semester')
      AND assessment_snapshots_period_start IS NOT NULL
      AND assessment_snapshots_period_end   IS NOT NULL
      AND assessment_snapshots_period_start <= assessment_snapshots_period_end
      AND (
        assessment_snapshots_period_type <> 'semester'
        OR assessment_snapshots_period_label IS NOT NULL
      )
    )
  ),

  -- FK
  CONSTRAINT fk_assessment_snapshots_school_assessment_tenant
    FOREIGN KEY (assessment_snapshots_assessment_id, assessment_snapshots_school_id)
    REFERENCES assessments(assessment_id, assessment_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Exclusion: final snapshots tidak overlap per (tenant, tipe, rentang)
  CONSTRAINT assessment_snapshots_school_final_no_overlap EXCLUDE USING gist (
    assessment_snapshots_school_id WITH =,
    assessment_snapshots_period_type WITH =,
    daterange(assessment_snapshots_period_start, assessment_snapshots_period_end, '[]') WITH &&
  ) WHERE (assessment_snapshots_is_final)
);

-- Unique (alive) — assessment per scope (ikut anchor)
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_school_alive_assessment
  ON assessment_snapshots_school (assessment_snapshots_school_id, assessment_snapshots_assessment_id)
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type = 'assessment';

-- Unique (alive) — period per scope
CREATE UNIQUE INDEX IF NOT EXISTS uq_assessment_snapshots_school_alive_period
  ON assessment_snapshots_school (
    assessment_snapshots_school_id,
    assessment_snapshots_period_type,
    assessment_snapshots_period_start,
    assessment_snapshots_period_end
  )
  WHERE assessment_snapshots_deleted_at IS NULL
    AND assessment_snapshots_period_type IN ('week','month','semester');

-- =========================================================
-- GLOBAL INDEXES (parent) — JSONB & BRIN (diwariskan ke partisi)
-- =========================================================
CREATE INDEX IF NOT EXISTS gin_assessment_snapshots_status_counts
  ON assessment_snapshots USING GIN (assessment_snapshots_status_counts jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_assessment_snapshots_grade_hist
  ON assessment_snapshots USING GIN (assessment_snapshots_grade_hist jsonb_path_ops);

CREATE INDEX IF NOT EXISTS brin_assessment_snapshots_created
  ON assessment_snapshots USING BRIN (assessment_snapshots_created_at);

CREATE INDEX IF NOT EXISTS brin_assessment_snapshots_last_submitted
  ON assessment_snapshots USING BRIN (assessment_snapshots_last_submitted_at);

-- =========================================================
-- Helpful query indexes
-- =========================================================
CREATE INDEX IF NOT EXISTS idx_assessment_snapshots_tenant_periodtype
  ON assessment_snapshots (assessment_snapshots_school_id, assessment_snapshots_period_type)
  WHERE assessment_snapshots_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_assessment_snapshots_period_range
  ON assessment_snapshots (assessment_snapshots_period_start, assessment_snapshots_period_end)
  WHERE assessment_snapshots_deleted_at IS NULL;

COMMIT;
