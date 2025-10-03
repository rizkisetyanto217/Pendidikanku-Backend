-- +migrate Up
-- =========================================================
-- UP Migration — Attendance Snapshots (Unified, Partitioned, No ALTER)
-- =========================================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =========================================================
-- ENUM scope
-- =========================================================
DO $$ BEGIN
  CREATE TYPE attendance_snapshots_scope AS ENUM ('ucsst','section','masjid');
EXCEPTION WHEN duplicate_object THEN END $$;

-- =========================================================
-- PARENT TABLE (partitioned by scope)
-- =========================================================
CREATE TABLE IF NOT EXISTS attendance_snapshots (
  attendance_snapshots_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  attendance_snapshots_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  attendance_snapshots_scope attendance_snapshots_scope NOT NULL, -- partition key
  attendance_snapshots_period_type VARCHAR(12) NOT NULL
    CHECK (attendance_snapshots_period_type IN ('session','week','month','semester')),

  -- anchors (opsional; di-validate per partisi)
  attendance_snapshots_session_id UUID,
  attendance_snapshots_ucsst_id   UUID,
  attendance_snapshots_section_id UUID,

  -- period info
  attendance_snapshots_period_start DATE,
  attendance_snapshots_period_end   DATE,
  attendance_snapshots_period_code  VARCHAR(24),

  -- agregat populasi & tanda
  attendance_snapshots_total_enrolled     INT,
  attendance_snapshots_expected_attendees INT,
  attendance_snapshots_marked_count       INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_marked_count >= 0),
  attendance_snapshots_unmarked_count     INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_unmarked_count >= 0),

  -- hitung status
  attendance_snapshots_present_count INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_present_count >= 0),
  attendance_snapshots_absent_count  INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_absent_count  >= 0),
  attendance_snapshots_excused_count INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_excused_count >= 0),
  attendance_snapshots_late_count    INT NOT NULL DEFAULT 0 CHECK (attendance_snapshots_late_count    >= 0),

  -- persentase
  attendance_snapshots_present_rate_pct NUMERIC(6,3),
  attendance_snapshots_absent_rate_pct  NUMERIC(6,3),
  attendance_snapshots_excused_rate_pct NUMERIC(6,3),
  attendance_snapshots_late_rate_pct    NUMERIC(6,3),
  attendance_snapshots_marked_rate_pct  NUMERIC(6,3),

  -- keterlambatan
  attendance_snapshots_late_seconds_total BIGINT,
  attendance_snapshots_late_seconds_avg   NUMERIC(12,3),
  attendance_snapshots_late_seconds_p95   NUMERIC(12,3),

  -- skor (opsional)
  attendance_snapshots_score_count INT,
  attendance_snapshots_score_avg   NUMERIC(8,3),
  attendance_snapshots_score_min   NUMERIC(8,3),
  attendance_snapshots_score_max   NUMERIC(8,3),

  -- kelulusan (opsional)
  attendance_snapshots_passed_count    INT,
  attendance_snapshots_passed_rate_pct NUMERIC(6,3),

  -- histogram JSONB
  attendance_snapshots_type_counts   JSONB NOT NULL DEFAULT '{}'::jsonb,
  attendance_snapshots_method_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_attendance_snapshots_type_counts_obj
    CHECK (jsonb_typeof(attendance_snapshots_type_counts) = 'object'),
  CONSTRAINT ck_attendance_snapshots_method_counts_obj
    CHECK (jsonb_typeof(attendance_snapshots_method_counts) = 'object'),

  -- audit waktu penandaan
  attendance_snapshots_first_marked_at TIMESTAMPTZ,
  attendance_snapshots_last_marked_at  TIMESTAMPTZ,

  -- audit & soft delete
  attendance_snapshots_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  attendance_snapshots_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  attendance_snapshots_deleted_at TIMESTAMPTZ
)
PARTITION BY LIST (attendance_snapshots_scope);

-- =========================================================
-- PARTITION: UCSST
-- =========================================================
CREATE TABLE IF NOT EXISTS attendance_snapshots_ucsst
  PARTITION OF attendance_snapshots FOR VALUES IN ('ucsst')
  (
    -- Validasi bentuk (inline constraint)
    CONSTRAINT ck_attendance_snapshots_ucsst_shape CHECK (
      (attendance_snapshots_period_type = 'session'
        AND attendance_snapshots_session_id IS NOT NULL)
      OR
      (attendance_snapshots_period_type IN ('week','month','semester')
        AND attendance_snapshots_ucsst_id IS NOT NULL
        AND attendance_snapshots_period_start IS NOT NULL
        AND attendance_snapshots_period_end   IS NOT NULL
        AND attendance_snapshots_period_start <= attendance_snapshots_period_end)
    ),

    -- FK relevan (inline)
    CONSTRAINT fk_attendance_snapshots_ucsst_session
      FOREIGN KEY (attendance_snapshots_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE CASCADE,

    CONSTRAINT fk_attendance_snapshots_ucsst_ucsst
      FOREIGN KEY (attendance_snapshots_ucsst_id)
      REFERENCES user_class_section_subject_teachers(user_class_section_subject_teacher_id)
      ON UPDATE CASCADE ON DELETE CASCADE
  );

-- =========================================================
-- PARTITION: SECTION
-- =========================================================
CREATE TABLE IF NOT EXISTS attendance_snapshots_section
  PARTITION OF attendance_snapshots FOR VALUES IN ('section')
  (
    CONSTRAINT ck_attendance_snapshots_section_shape CHECK (
      (attendance_snapshots_period_type = 'session'
        AND attendance_snapshots_session_id IS NOT NULL)
      OR
      (attendance_snapshots_period_type IN ('week','month','semester')
        AND attendance_snapshots_section_id IS NOT NULL
        AND attendance_snapshots_period_start IS NOT NULL
        AND attendance_snapshots_period_end   IS NOT NULL
        AND attendance_snapshots_period_start <= attendance_snapshots_period_end)
    ),

    CONSTRAINT fk_attendance_snapshots_section_session
      FOREIGN KEY (attendance_snapshots_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE CASCADE,

    CONSTRAINT fk_attendance_snapshots_section_section
      FOREIGN KEY (attendance_snapshots_section_id)
      REFERENCES class_sections(class_section_id)
      ON UPDATE CASCADE ON DELETE CASCADE
  );

-- =========================================================
-- PARTITION: MASJID
-- =========================================================
CREATE TABLE IF NOT EXISTS attendance_snapshots_masjid
  PARTITION OF attendance_snapshots FOR VALUES IN ('masjid')
  (
    CONSTRAINT ck_attendance_snapshots_masjid_shape CHECK (
      (attendance_snapshots_period_type = 'session'
        AND attendance_snapshots_session_id IS NOT NULL)
      OR
      (attendance_snapshots_period_type IN ('week','month','semester')
        AND attendance_snapshots_period_start IS NOT NULL
        AND attendance_snapshots_period_end   IS NOT NULL
        AND attendance_snapshots_period_start <= attendance_snapshots_period_end)
    ),

    CONSTRAINT fk_attendance_snapshots_masjid_session
      FOREIGN KEY (attendance_snapshots_session_id)
      REFERENCES class_attendance_sessions(class_attendance_session_id)
      ON UPDATE CASCADE ON DELETE CASCADE
  );

-- =========================================================
-- UNIQUE "ALIVE" (per partition; soft delete aware)
-- =========================================================

-- UCSST: unik per session
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_ucsst_alive_session
  ON attendance_snapshots_ucsst (attendance_snapshots_masjid_id, attendance_snapshots_session_id)
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type = 'session'
    AND attendance_snapshots_session_id IS NOT NULL;

-- UCSST: unik per period
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_ucsst_alive_period
  ON attendance_snapshots_ucsst (
    attendance_snapshots_masjid_id,
    attendance_snapshots_ucsst_id,
    attendance_snapshots_period_type,
    attendance_snapshots_period_start,
    attendance_snapshots_period_end
  )
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type IN ('week','month','semester')
    AND attendance_snapshots_ucsst_id IS NOT NULL;

-- SECTION: unik per session
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_section_alive_session
  ON attendance_snapshots_section (attendance_snapshots_masjid_id, attendance_snapshots_session_id)
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type = 'session'
    AND attendance_snapshots_session_id IS NOT NULL;

-- SECTION: unik per period
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_section_alive_period
  ON attendance_snapshots_section (
    attendance_snapshots_masjid_id,
    attendance_snapshots_section_id,
    attendance_snapshots_period_type,
    attendance_snapshots_period_start,
    attendance_snapshots_period_end
  )
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type IN ('week','month','semester')
    AND attendance_snapshots_section_id IS NOT NULL;

-- MASJID: unik per session
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_masjid_alive_session
  ON attendance_snapshots_masjid (attendance_snapshots_masjid_id, attendance_snapshots_session_id)
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type = 'session'
    AND attendance_snapshots_session_id IS NOT NULL;

-- MASJID: unik per period
CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_snapshots_masjid_alive_period
  ON attendance_snapshots_masjid (
    attendance_snapshots_masjid_id,
    attendance_snapshots_period_type,
    attendance_snapshots_period_start,
    attendance_snapshots_period_end
  )
  WHERE attendance_snapshots_deleted_at IS NULL
    AND attendance_snapshots_period_type IN ('week','month','semester');

-- =========================================================
-- JSONB GIN & BRIN (per parent untuk simplicity; boleh di-partition juga)
-- =========================================================
CREATE INDEX IF NOT EXISTS gin_attendance_snapshots_type_counts
  ON attendance_snapshots USING GIN (attendance_snapshots_type_counts);

CREATE INDEX IF NOT EXISTS gin_attendance_snapshots_method_counts
  ON attendance_snapshots USING GIN (attendance_snapshots_method_counts);

CREATE INDEX IF NOT EXISTS brin_attendance_snapshots_created_at
  ON attendance_snapshots USING BRIN (attendance_snapshots_created_at);

CREATE INDEX IF NOT EXISTS brin_attendance_snapshots_last_marked_at
  ON attendance_snapshots USING BRIN (attendance_snapshots_last_marked_at);

COMMIT;
