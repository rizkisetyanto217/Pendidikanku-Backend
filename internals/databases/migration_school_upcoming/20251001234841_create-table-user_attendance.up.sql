-- +migrate Up
-- =========================================================
-- UP — USER ATTENDANCE SNAPSHOTS (UC-SST & UC-SEC, week|month|semester)
-- Unified + Partitioned, No ALTER
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =========================
-- ENUMs
-- =========================
DO $$ BEGIN
  CREATE TYPE user_attendance_snapshots_scope AS ENUM ('ucsst','ucsec');
EXCEPTION WHEN duplicate_object THEN END $$;

DO $$ BEGIN
  CREATE TYPE user_attendance_snapshots_grain AS ENUM ('week','month','semester');
EXCEPTION WHEN duplicate_object THEN END $$;

-- =========================
-- PARENT (partitioned by scope)
-- =========================
CREATE TABLE IF NOT EXISTS user_attendance_snapshots (
  user_attendance_snapshots_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_attendance_snapshots_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- partition key
  user_attendance_snapshots_scope user_attendance_snapshots_scope NOT NULL,

  -- anchors (diisi sesuai scope di partisi)
  user_attendance_snapshots_ucsst_id UUID, -- per siswa × subject-teacher
  user_attendance_snapshots_ucsec_id UUID, -- per siswa × section

  -- grain & rentang (lokal di layer app)
  user_attendance_snapshots_grain user_attendance_snapshots_grain NOT NULL, -- 'week'|'month'|'semester'
  user_attendance_snapshots_start DATE NOT NULL,
  user_attendance_snapshots_end   DATE NOT NULL,

  -- label opsional (mis. '2025-S1'); bisa diwajibkan utk semester di partisi
  user_attendance_snapshots_period_label VARCHAR(100),

  -- ===== Agregat dasar =====
  user_attendance_snapshots_weeks INT, -- NULL utk week; >=1 utk month & semester
  user_attendance_snapshots_sessions_expected INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_sessions_expected >= 0),
  user_attendance_snapshots_sessions_marked   INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_sessions_marked   >= 0),
  user_attendance_snapshots_sessions_unmarked INT GENERATED ALWAYS AS (
    GREATEST(user_attendance_snapshots_sessions_expected - user_attendance_snapshots_sessions_marked, 0)
  ) STORED,

  -- ===== Hitung status =====
  user_attendance_snapshots_present_count INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_present_count >= 0),
  user_attendance_snapshots_absent_count  INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_absent_count  >= 0),
  user_attendance_snapshots_excused_count INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_excused_count >= 0),
  user_attendance_snapshots_late_count    INT NOT NULL DEFAULT 0 CHECK (user_attendance_snapshots_late_count    >= 0),

  -- ===== Persentase =====
  user_attendance_snapshots_present_rate_pct NUMERIC(6,3),
  user_attendance_snapshots_absent_rate_pct  NUMERIC(6,3),
  user_attendance_snapshots_excused_rate_pct NUMERIC(6,3),
  user_attendance_snapshots_late_rate_pct    NUMERIC(6,3),
  user_attendance_snapshots_marked_rate_pct  NUMERIC(6,3),

  -- ===== Keterlambatan (detik/menit) =====
  user_attendance_snapshots_late_seconds_total BIGINT,
  user_attendance_snapshots_late_seconds_avg   NUMERIC(12,3),
  user_attendance_snapshots_late_seconds_p95   NUMERIC(12,3),

  -- ===== Histogram breakdown =====
  user_attendance_snapshots_type_counts   JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_att_snap_type_counts_obj
    CHECK (jsonb_typeof(user_attendance_snapshots_type_counts) = 'object'),

  user_attendance_snapshots_method_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_att_snap_method_counts_obj
    CHECK (jsonb_typeof(user_attendance_snapshots_method_counts) = 'object'),

  -- ===== Detail sesi (hanya weekly) =====
  -- contoh elemen: {"session_id":"...","status":"present","late_sec":120,"marked_at":"..."}
  user_attendance_snapshots_sessions JSONB NOT NULL DEFAULT '[]'::jsonb,
  CONSTRAINT ck_user_att_snap_sessions_arr
    CHECK (jsonb_typeof(user_attendance_snapshots_sessions) = 'array'),

  -- Meta opsional
  user_attendance_snapshots_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_user_att_snap_meta_obj
    CHECK (jsonb_typeof(user_attendance_snapshots_meta) = 'object'),

  -- Finalisasi batch
  user_attendance_snapshots_is_final     BOOLEAN NOT NULL DEFAULT FALSE,
  user_attendance_snapshots_finalized_at TIMESTAMPTZ,
  user_attendance_snapshots_version      INT NOT NULL DEFAULT 1,

  -- Audit
  user_attendance_snapshots_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_snapshots_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
PARTITION BY LIST (user_attendance_snapshots_scope);

-- =========================================================
-- PARTITION: UC-SST (per siswa × subject-teacher)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_attendance_snapshots_ucsst
  PARTITION OF user_attendance_snapshots FOR VALUES IN ('ucsst')
(
  -- Shape rules
  CONSTRAINT ck_user_att_snap_ucsst_shape CHECK (
    user_attendance_snapshots_ucsst_id IS NOT NULL
    AND user_attendance_snapshots_start IS NOT NULL
    AND user_attendance_snapshots_end   IS NOT NULL
    AND user_attendance_snapshots_start <= user_attendance_snapshots_end
    AND (
      -- weeks: NULL utk week; >=1 utk month/semester
      (user_attendance_snapshots_grain = 'week'  AND user_attendance_snapshots_weeks IS NULL)
      OR
      (user_attendance_snapshots_grain IN ('month','semester')
        AND user_attendance_snapshots_weeks IS NOT NULL AND user_attendance_snapshots_weeks >= 1)
    )
    AND (
      -- sessions array hanya untuk weekly
      (user_attendance_snapshots_grain = 'week')
      OR
      (user_attendance_snapshots_grain IN ('month','semester')
        AND jsonb_array_length(user_attendance_snapshots_sessions) = 0)
    )
  ),

  -- Wajibkan label utk semester (opsional tapi direkomendasikan)
  CONSTRAINT ck_user_att_snap_ucsst_semester_label CHECK (
    user_attendance_snapshots_grain <> 'semester'
    OR user_attendance_snapshots_period_label IS NOT NULL
  ),

  -- FK anchor
  CONSTRAINT fk_user_att_snap_ucsst_anchor
    FOREIGN KEY (user_attendance_snapshots_ucsst_id)
    REFERENCES user_class_section_subject_teachers(user_class_section_subject_teacher_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Unik per (tenant × ucsst × grain × start)
  CONSTRAINT uq_user_att_snap_ucsst_unique
    UNIQUE (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsst_id, user_attendance_snapshots_grain, user_attendance_snapshots_start)
);

-- Indexes UC-SST
CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsst_tenant_anchor
  ON user_attendance_snapshots_ucsst (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsst_id);

CREATE INDEX IF NOT EXISTS brin_user_att_snap_ucsst_start
  ON user_attendance_snapshots_ucsst USING BRIN (user_attendance_snapshots_start);

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsst_updated_at
  ON user_attendance_snapshots_ucsst (user_attendance_snapshots_updated_at);

-- Partial per grain
CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsst_week
  ON user_attendance_snapshots_ucsst (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsst_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'week';

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsst_month
  ON user_attendance_snapshots_ucsst (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsst_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'month';

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsst_semester
  ON user_attendance_snapshots_ucsst (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsst_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'semester';

-- JSONB
CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsst_type_counts
  ON user_attendance_snapshots_ucsst USING GIN (user_attendance_snapshots_type_counts);

CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsst_method_counts
  ON user_attendance_snapshots_ucsst USING GIN (user_attendance_snapshots_method_counts);

CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsst_sessions_week_only
  ON user_attendance_snapshots_ucsst USING GIN (user_attendance_snapshots_sessions)
  WHERE user_attendance_snapshots_grain = 'week';

-- =========================================================
-- PARTITION: UC-SEC (per siswa × section)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_attendance_snapshots_ucsec
  PARTITION OF user_attendance_snapshots FOR VALUES IN ('ucsec')
(
  CONSTRAINT ck_user_att_snap_ucsec_shape CHECK (
    user_attendance_snapshots_ucsec_id IS NOT NULL
    AND user_attendance_snapshots_start IS NOT NULL
    AND user_attendance_snapshots_end   IS NOT NULL
    AND user_attendance_snapshots_start <= user_attendance_snapshots_end
    AND (
      (user_attendance_snapshots_grain = 'week'  AND user_attendance_snapshots_weeks IS NULL)
      OR
      (user_attendance_snapshots_grain IN ('month','semester')
        AND user_attendance_snapshots_weeks IS NOT NULL AND user_attendance_snapshots_weeks >= 1)
    )
    AND (
      (user_attendance_snapshots_grain = 'week')
      OR
      (user_attendance_snapshots_grain IN ('month','semester')
        AND jsonb_array_length(user_attendance_snapshots_sessions) = 0)
    )
  ),

  CONSTRAINT ck_user_att_snap_ucsec_semester_label CHECK (
    user_attendance_snapshots_grain <> 'semester'
    OR user_attendance_snapshots_period_label IS NOT NULL
  ),

  -- FK anchor
  CONSTRAINT fk_user_att_snap_ucsec_anchor
    FOREIGN KEY (user_attendance_snapshots_ucsec_id)
    REFERENCES user_class_sections(user_class_section_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Unik per (tenant × ucsec × grain × start)
  CONSTRAINT uq_user_att_snap_ucsec_unique
    UNIQUE (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsec_id, user_attendance_snapshots_grain, user_attendance_snapshots_start)
);

-- Indexes UC-SEC
CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsec_tenant_anchor
  ON user_attendance_snapshots_ucsec (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsec_id);

CREATE INDEX IF NOT EXISTS brin_user_att_snap_ucsec_start
  ON user_attendance_snapshots_ucsec USING BRIN (user_attendance_snapshots_start);

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsec_updated_at
  ON user_attendance_snapshots_ucsec (user_attendance_snapshots_updated_at);

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsec_week
  ON user_attendance_snapshots_ucsec (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsec_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'week';

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsec_month
  ON user_attendance_snapshots_ucsec (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsec_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'month';

CREATE INDEX IF NOT EXISTS idx_user_att_snap_ucsec_semester
  ON user_attendance_snapshots_ucsec (user_attendance_snapshots_masjid_id, user_attendance_snapshots_ucsec_id, user_attendance_snapshots_start)
  WHERE user_attendance_snapshots_grain = 'semester';

-- JSONB
CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsec_type_counts
  ON user_attendance_snapshots_ucsec USING GIN (user_attendance_snapshots_type_counts);

CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsec_method_counts
  ON user_attendance_snapshots_ucsec USING GIN (user_attendance_snapshots_method_counts);

CREATE INDEX IF NOT EXISTS gin_user_att_snap_ucsec_sessions_week_only
  ON user_attendance_snapshots_ucsec USING GIN (user_attendance_snapshots_sessions)
  WHERE user_attendance_snapshots_grain = 'week';

-- =========================
-- Helpful Parent Indexes
-- =========================
CREATE INDEX IF NOT EXISTS idx_user_att_snap_tenant_grain_start
  ON user_attendance_snapshots (user_attendance_snapshots_masjid_id, user_attendance_snapshots_grain, user_attendance_snapshots_start);

CREATE INDEX IF NOT EXISTS brin_user_att_snap_created
  ON user_attendance_snapshots USING BRIN (user_attendance_snapshots_created_at);

COMMIT;
