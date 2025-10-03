-- +migrate Up
-- =========================================================
-- UP — USER SCORE SNAPSHOTS (UC-SST & UC-SEC, week|month|semester)
-- Unified + Partitioned, No ALTER
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- =========================
-- ENUMs
-- =========================
DO $$ BEGIN
  -- scope partisi
  CREATE TYPE user_score_snapshots_scope AS ENUM ('ucsst','ucsec');
EXCEPTION WHEN duplicate_object THEN END $$;

DO $$ BEGIN
  -- grain periode
  CREATE TYPE user_score_snapshots_grain AS ENUM ('week','month','semester');
EXCEPTION WHEN duplicate_object THEN END $$;

-- =========================
-- PARENT (partitioned by scope)
-- =========================
CREATE TABLE IF NOT EXISTS user_score_snapshots (
  user_score_snapshots_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_score_snapshots_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- partition key
  user_score_snapshots_scope user_score_snapshots_scope NOT NULL,

  -- anchors (diisi sesuai scope di partisi)
  user_score_snapshots_ucsst_id UUID, -- per siswa × subject-teacher
  user_score_snapshots_ucsec_id UUID, -- per siswa × section

  -- grain & rentang lokal (Asia/Jakarta di layer app)
  user_score_snapshots_grain user_score_snapshots_grain NOT NULL, -- 'week' | 'month' | 'semester'
  user_score_snapshots_start DATE NOT NULL,
  user_score_snapshots_end   DATE NOT NULL,

  -- label opsional (mis. '2025-S1'); tidak diwajibkan di parent
  user_score_snapshots_period_label VARCHAR(100),

  -- ===== Agregat dasar =====
  user_score_snapshots_weeks INT, -- NULL utk week; >=1 utk month & semester
  user_score_snapshots_submissions_count INT NOT NULL DEFAULT 0,
  user_score_snapshots_score_total       NUMERIC(10,2),
  user_score_snapshots_score_max_total   NUMERIC(10,2),
  user_score_snapshots_score_percent_avg NUMERIC(6,3),
  user_score_snapshots_score_percent_min NUMERIC(6,3),
  user_score_snapshots_score_percent_max NUMERIC(6,3),

  -- ===== Late metrics =====
  user_score_snapshots_late_count      INT NOT NULL DEFAULT 0,
  user_score_snapshots_late_rate_pct   NUMERIC(6,3),
  user_score_snapshots_late_minute_avg NUMERIC(8,2),
  user_score_snapshots_late_minute_p95 NUMERIC(8,2),

  -- ===== Remedial metrics =====
  user_score_snapshots_remedial_count        INT NOT NULL DEFAULT 0,
  user_score_snapshots_remedial_rate_pct     NUMERIC(6,3),
  user_score_snapshots_remedial_attempts_avg NUMERIC(6,3),
  user_score_snapshots_remedial_attempts_max INT,
  user_score_snapshots_score_improve_avg     NUMERIC(6,3),
  user_score_snapshots_score_improve_max     NUMERIC(6,3),

  -- ===== Breakdown & histogram =====
  user_score_snapshots_status_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_score_snapshots_status_counts_obj
    CHECK (jsonb_typeof(user_score_snapshots_status_counts) = 'object'),

  user_score_snapshots_grade_hist JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_score_snapshots_grade_hist_obj
    CHECK (jsonb_typeof(user_score_snapshots_grade_hist) = 'object'),

  -- Ringkasan per-mapel (lebih berguna di ucsec tapi diseragamkan)
  user_score_snapshots_subject_briefs JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_user_score_snapshots_subject_briefs_arr
    CHECK (jsonb_typeof(user_score_snapshots_subject_briefs) = 'array'),

  -- Detail submissions (hanya weekly)
  user_score_snapshots_submissions JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_user_score_snapshots_submissions_arr
    CHECK (jsonb_typeof(user_score_snapshots_submissions) = 'array'),

  -- Meta opsional
  user_score_snapshots_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_score_snapshots_meta_obj
    CHECK (jsonb_typeof(user_score_snapshots_meta) = 'object'),

  -- Finalisasi batch
  user_score_snapshots_is_final     BOOLEAN NOT NULL DEFAULT FALSE,
  user_score_snapshots_finalized_at TIMESTAMPTZ,
  user_score_snapshots_version      INT NOT NULL DEFAULT 1,

  -- Audit
  user_score_snapshots_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_score_snapshots_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
PARTITION BY LIST (user_score_snapshots_scope);

-- =========================================================
-- PARTITION: UC-SST (per siswa × subject-teacher)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_score_snapshots_ucsst
  PARTITION OF user_score_snapshots FOR VALUES IN ('ucsst')
(
  -- Shape & anchor rules
  CONSTRAINT ck_user_score_snapshots_ucsst_shape CHECK (
    user_score_snapshots_ucsst_id IS NOT NULL
    AND user_score_snapshots_start IS NOT NULL
    AND user_score_snapshots_end   IS NOT NULL
    AND user_score_snapshots_start <= user_score_snapshots_end
    AND (
      -- weeks: NULL utk week; >=1 utk month/semester
      (user_score_snapshots_grain = 'week'  AND user_score_snapshots_weeks IS NULL)
      OR
      (user_score_snapshots_grain IN ('month','semester')
        AND user_score_snapshots_weeks IS NOT NULL AND user_score_snapshots_weeks >= 1)
    )
    AND (
      -- submissions array hanya untuk weekly
      (user_score_snapshots_grain = 'week')
      OR
      (user_score_snapshots_grain IN ('month','semester')
        AND jsonb_array_length(user_score_snapshots_submissions) = 0)
    )
  ),

  -- OPTION: Wajibkan label untuk semester (lebih rapi untuk BI)
  CONSTRAINT ck_user_score_snapshots_ucsst_semester_label CHECK (
    user_score_snapshots_grain <> 'semester'
    OR user_score_snapshots_period_label IS NOT NULL
  ),

  -- FK anchor
  CONSTRAINT fk_user_score_snapshots_ucsst_anchor
    FOREIGN KEY (user_score_snapshots_ucsst_id)
    REFERENCES user_class_section_subject_teachers(user_class_section_subject_teacher_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Unik per (tenant × ucsst × grain × start)
  CONSTRAINT uq_user_score_snapshots_ucsst_unique
    UNIQUE (user_score_snapshots_masjid_id, user_score_snapshots_ucsst_id, user_score_snapshots_grain, user_score_snapshots_start)
);

-- Indexes UC-SST
CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsst_tenant_anchor
  ON user_score_snapshots_ucsst (user_score_snapshots_masjid_id, user_score_snapshots_ucsst_id);

CREATE INDEX IF NOT EXISTS brin_user_score_snapshots_ucsst_start
  ON user_score_snapshots_ucsst USING BRIN (user_score_snapshots_start);

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsst_updated_at
  ON user_score_snapshots_ucsst (user_score_snapshots_updated_at);

-- Partial index per grain
CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsst_week
  ON user_score_snapshots_ucsst (user_score_snapshots_masjid_id, user_score_snapshots_ucsst_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'week';

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsst_month
  ON user_score_snapshots_ucsst (user_score_snapshots_masjid_id, user_score_snapshots_ucsst_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'month';

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsst_semester
  ON user_score_snapshots_ucsst (user_score_snapshots_masjid_id, user_score_snapshots_ucsst_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'semester';

-- JSONB indexes
CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsst_status_counts
  ON user_score_snapshots_ucsst USING GIN (user_score_snapshots_status_counts jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsst_grade_hist
  ON user_score_snapshots_ucsst USING GIN (user_score_snapshots_grade_hist jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsst_submissions_week_only
  ON user_score_snapshots_ucsst USING GIN (user_score_snapshots_submissions jsonb_path_ops)
  WHERE user_score_snapshots_grain = 'week';

-- =========================================================
-- PARTITION: UC-SEC (per siswa × section)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_score_snapshots_ucsec
  PARTITION OF user_score_snapshots FOR VALUES IN ('ucsec')
(
  CONSTRAINT ck_user_score_snapshots_ucsec_shape CHECK (
    user_score_snapshots_ucsec_id IS NOT NULL
    AND user_score_snapshots_start IS NOT NULL
    AND user_score_snapshots_end   IS NOT NULL
    AND user_score_snapshots_start <= user_score_snapshots_end
    AND (
      (user_score_snapshots_grain = 'week'  AND user_score_snapshots_weeks IS NULL)
      OR
      (user_score_snapshots_grain IN ('month','semester')
        AND user_score_snapshots_weeks IS NOT NULL AND user_score_snapshots_weeks >= 1)
    )
    AND (
      (user_score_snapshots_grain = 'week')
      OR
      (user_score_snapshots_grain IN ('month','semester')
        AND jsonb_array_length(user_score_snapshots_submissions) = 0)
    )
  ),

  CONSTRAINT ck_user_score_snapshots_ucsec_semester_label CHECK (
    user_score_snapshots_grain <> 'semester'
    OR user_score_snapshots_period_label IS NOT NULL
  ),

  -- FK anchor
  CONSTRAINT fk_user_score_snapshots_ucsec_anchor
    FOREIGN KEY (user_score_snapshots_ucsec_id)
    REFERENCES user_class_sections(user_class_section_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Unik per (tenant × ucsec × grain × start)
  CONSTRAINT uq_user_score_snapshots_ucsec_unique
    UNIQUE (user_score_snapshots_masjid_id, user_score_snapshots_ucsec_id, user_score_snapshots_grain, user_score_snapshots_start)
);

-- Indexes UC-SEC
CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsec_tenant_anchor
  ON user_score_snapshots_ucsec (user_score_snapshots_masjid_id, user_score_snapshots_ucsec_id);

CREATE INDEX IF NOT EXISTS brin_user_score_snapshots_ucsec_start
  ON user_score_snapshots_ucsec USING BRIN (user_score_snapshots_start);

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsec_updated_at
  ON user_score_snapshots_ucsec (user_score_snapshots_updated_at);

-- Partial index per grain
CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsec_week
  ON user_score_snapshots_ucsec (user_score_snapshots_masjid_id, user_score_snapshots_ucsec_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'week';

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsec_month
  ON user_score_snapshots_ucsec (user_score_snapshots_masjid_id, user_score_snapshots_ucsec_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'month';

CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_ucsec_semester
  ON user_score_snapshots_ucsec (user_score_snapshots_masjid_id, user_score_snapshots_ucsec_id, user_score_snapshots_start)
  WHERE user_score_snapshots_grain = 'semester';

-- JSONB indexes
CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsec_status_counts
  ON user_score_snapshots_ucsec USING GIN (user_score_snapshots_status_counts jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsec_grade_hist
  ON user_score_snapshots_ucsec USING GIN (user_score_snapshots_grade_hist jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsec_subject_briefs
  ON user_score_snapshots_ucsec USING GIN (user_score_snapshots_subject_briefs jsonb_path_ops);

CREATE INDEX IF NOT EXISTS gin_user_score_snapshots_ucsec_submissions_week_only
  ON user_score_snapshots_ucsec USING GIN (user_score_snapshots_submissions jsonb_path_ops)
  WHERE user_score_snapshots_grain = 'week';

-- =========================
-- Helpful Parent Indexes
-- =========================
CREATE INDEX IF NOT EXISTS idx_user_score_snapshots_tenant_grain_start
  ON user_score_snapshots (user_score_snapshots_masjid_id, user_score_snapshots_grain, user_score_snapshots_start);

CREATE INDEX IF NOT EXISTS brin_user_score_snapshots_created
  ON user_score_snapshots USING BRIN (user_score_snapshots_created_at);

COMMIT;
