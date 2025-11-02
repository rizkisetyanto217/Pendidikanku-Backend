-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi index (opsional)

-- =========================================================
-- ENUM
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'week_parity_enum') THEN
    CREATE TYPE week_parity_enum AS ENUM ('all','odd','even');
  END IF;
END$$;

-- =========================================================
-- FUNC: validasi dan anti-duplikat untuk rules JSONB
-- =========================================================
CREATE OR REPLACE FUNCTION class_schedule_rules_has_dupes(rules JSONB)
RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$
  SELECT EXISTS (
    SELECT 1
    FROM (
      SELECT
        (e->>'day_of_week')::int AS dow,
        (e->>'start_time')       AS st,
        (e->>'end_time')         AS en
      FROM jsonb_array_elements(COALESCE(rules,'[]'::jsonb)) e
      WHERE jsonb_typeof(e) = 'object'
    ) x
    GROUP BY dow, st, en
    HAVING COUNT(*) > 1
  );
$$;

CREATE OR REPLACE FUNCTION class_schedule_rules_is_valid(rules JSONB)
RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$
  SELECT
    jsonb_typeof(COALESCE(rules,'[]'::jsonb)) = 'array'
    AND NOT EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(rules,'[]'::jsonb)) e
      WHERE NOT (
        jsonb_typeof(e) = 'object'
        AND (e ? 'day_of_week') AND (e->>'day_of_week') ~ '^[1-7]$'
        AND (e ? 'start_time')  AND (e->>'start_time')  ~ '^[0-2][0-9]:[0-5][0-9](:[0-5][0-9])?$'
        AND (e ? 'end_time')    AND (e->>'end_time')    ~ '^[0-2][0-9]:[0-5][0-9](:[0-5][0-9])?$'
        AND (e->>'end_time') > (e->>'start_time')
        AND COALESCE((e->>'interval_weeks')::int, 1) >= 1
        AND COALESCE((e->>'start_offset_weeks')::int, 0) >= 0
        AND COALESCE((e->>'week_parity'),'all') IN ('all','odd','even')
      )
    );
$$;

-- =========================================================
-- TABLE: class_schedules (rules langsung di JSONB)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedules (
  class_schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant scope
  class_schedule_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- slug (unik per tenant selama alive)
  class_schedule_slug VARCHAR(160),

  -- masa berlaku (header)
  class_schedule_start_date DATE NOT NULL,
  class_schedule_end_date   DATE NOT NULL
    CHECK (class_schedule_end_date >= class_schedule_start_date),

  -- status & flags
  class_schedule_status    session_status_enum NOT NULL DEFAULT 'scheduled',
  class_schedule_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- rules JSONB (array objek rule)
  class_schedule_rules JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- audit
  class_schedule_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_schedule_deleted_at TIMESTAMPTZ,

  -- tenant-safe composite key (target FK komposit)
  UNIQUE (class_schedule_school_id, class_schedule_id),

  -- CHECK: validasi & anti-duplikat
  CONSTRAINT chk_class_schedule_rules_valid
    CHECK (
      class_schedule_rules_is_valid(class_schedule_rules)
      AND NOT class_schedule_rules_has_dupes(class_schedule_rules)
    )
);

-- Indexes (class_schedules)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_schedules_slug_per_tenant_alive
  ON class_schedules (class_schedule_school_id, LOWER(class_schedule_slug))
  WHERE class_schedule_deleted_at IS NULL
    AND class_schedule_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS gin_class_schedules_slug_trgm_alive
  ON class_schedules USING GIN (LOWER(class_schedule_slug) gin_trgm_ops)
  WHERE class_schedule_deleted_at IS NULL
    AND class_schedule_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_tenant_alive
  ON class_schedules (class_schedule_school_id)
  WHERE class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_active_alive
  ON class_schedules (class_schedule_is_active)
  WHERE class_schedule_is_active = TRUE
    AND class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_schedules_date_bounds_alive
  ON class_schedules (class_schedule_start_date, class_schedule_end_date)
  WHERE class_schedule_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_class_schedules_created_at
  ON class_schedules USING BRIN (class_schedule_created_at);

-- GIN untuk lookup rules JSONB (JSONPath)
CREATE INDEX IF NOT EXISTS gin_class_schedule_rules
  ON class_schedules USING GIN (class_schedule_rules jsonb_path_ops);

-- =========================================================
-- VIEW: class_schedule_rules_v (hamparkan JSONB → baris)
-- =========================================================
CREATE OR REPLACE VIEW class_schedule_rules_v AS
SELECT
  cs.class_schedule_school_id,
  cs.class_schedule_id,
  (r->>'day_of_week')::int                         AS day_of_week,
  (r->>'start_time')::time                         AS start_time,
  (r->>'end_time')::time                           AS end_time,
  COALESCE((r->>'interval_weeks')::int, 1)         AS interval_weeks,
  COALESCE((r->>'start_offset_weeks')::int, 0)     AS start_offset_weeks,
  COALESCE((r->>'week_parity'),'all')::week_parity_enum AS week_parity,
  COALESCE((
    SELECT array_agg(value::int)
    FROM jsonb_array_elements_text(r->'weeks_of_month')
  ), '{}')::int[]                                  AS weeks_of_month,
  COALESCE((r->>'last_week_of_month')::boolean, false) AS last_week_of_month,
  row_number() OVER (
    PARTITION BY cs.class_schedule_school_id, cs.class_schedule_id
    ORDER BY (r->>'day_of_week'), (r->>'start_time'), (r->>'end_time')
  )::int                                           AS rule_idx,
  r                                                AS rule_json
FROM class_schedules cs
CROSS JOIN LATERAL jsonb_array_elements(cs.class_schedule_rules) r
WHERE cs.class_schedule_deleted_at IS NULL;

-- =========================================================
-- (OPSIONAL) TABLE: exceptions per tanggal
-- =========================================================
CREATE TABLE IF NOT EXISTS class_schedule_exceptions (
  cse_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cse_school_id UUID NOT NULL,
  cse_schedule_id UUID NOT NULL,
  cse_date DATE NOT NULL,
  cse_is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
  cse_new_start TIME,
  cse_new_end   TIME,
  cse_new_teacher_id UUID,
  cse_new_class_room_id UUID,
  cse_note TEXT,
  cse_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cse_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (cse_school_id, cse_schedule_id, cse_date)
);

-- =========================================================
-- FUNC: expand_occurrences_jsonb (rules → rentang tanggal)
-- =========================================================
CREATE OR REPLACE FUNCTION expand_occurrences_jsonb(
  p_school_id   UUID,
  p_schedule_id UUID,
  p_from        DATE,
  p_to          DATE
) RETURNS TABLE(
  school_id   UUID,
  schedule_id UUID,
  dt          DATE,
  start_time  TIME,
  end_time    TIME,
  rule_idx    INT,
  rule_json   JSONB
) LANGUAGE sql STABLE AS $$
  WITH params AS (
    SELECT p_from AS d_from, p_to AS d_to
  ),
  days AS (
    SELECT d::date AS dt FROM params, generate_series(p_from, p_to, interval '1 day') g(d)
  ),
  sched AS (
    SELECT *
    FROM class_schedule_rules_v r
    WHERE r.class_schedule_school_id = p_school_id
      AND r.class_schedule_id = p_schedule_id
  ),
  match_dow AS (
    SELECT s.*, d.dt
    FROM sched s
    JOIN days d
      ON EXTRACT(ISODOW FROM d.dt)::int = s.day_of_week
  ),
  mark_weeks AS (
    SELECT
      m.*,
      -- minggu sejak start_date (kasar tapi cukup untuk interval & parity)
      (EXTRACT(WEEK FROM m.dt)::int - EXTRACT(WEEK FROM (SELECT class_schedule_start_date FROM class_schedules
                                                         WHERE class_schedule_id = p_schedule_id
                                                           AND class_schedule_school_id = p_school_id))::int
         + 52 * (EXTRACT(YEAR FROM m.dt)::int - EXTRACT(YEAR FROM (SELECT class_schedule_start_date FROM class_schedules
                                                                   WHERE class_schedule_id = p_schedule_id
                                                                     AND class_schedule_school_id = p_school_id))::int)
      ) AS weeks_since_start,
      (EXTRACT(WEEK FROM m.dt)::int % 2) AS week_parity_mod,
      ((EXTRACT(DAY FROM m.dt)::int - 1)/7 + 1) AS week_of_month,
      (m.dt + (7 * INTERVAL '1 day') > (date_trunc('month', m.dt) + INTERVAL '1 month')) AS is_last_week_of_month
    FROM match_dow m
  ),
  filtered AS (
    SELECT *
    FROM mark_weeks
    WHERE
      -- within header date bounds
      dt BETWEEN (SELECT class_schedule_start_date FROM class_schedules
                  WHERE class_schedule_id = p_schedule_id AND class_schedule_school_id = p_school_id)
              AND (SELECT class_schedule_end_date   FROM class_schedules
                  WHERE class_schedule_id = p_schedule_id AND class_schedule_school_id = p_school_id)
      -- interval & offset
      AND (weeks_since_start - start_offset_weeks) % GREATEST(interval_weeks,1) = 0
      -- parity
      AND (
        week_parity = 'all'
        OR (week_parity = 'odd'  AND week_parity_mod = 1)
        OR (week_parity = 'even' AND week_parity_mod = 0)
      )
      -- weeks_of_month
      AND (
        COALESCE(array_length(weeks_of_month,1),0) = 0
        OR week_of_month = ANY(weeks_of_month)
      )
      -- last week of month
      AND (
        last_week_of_month = FALSE
        OR is_last_week_of_month = TRUE
      )
  )
  SELECT
    p_school_id   AS school_id,
    p_schedule_id AS schedule_id,
    f.dt,
    f.start_time,
    f.end_time,
    f.rule_idx,
    f.rule_json
  FROM filtered f
  ORDER BY f.dt, f.start_time;
$$;

-- =========================================================
-- TABLE: class_attendance_sessions (sessions-first)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard + relasi header (FK komposit tenant-safe → schedules)
  class_attendance_session_school_id   UUID NOT NULL,
  class_attendance_session_schedule_id UUID NOT NULL,
  CONSTRAINT fk_cas_schedule_tenant
    FOREIGN KEY (class_attendance_session_school_id, class_attendance_session_schedule_id)
    REFERENCES class_schedules (class_schedule_school_id, class_schedule_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- jejak rule asal (snapshot JSONB + idx)
  class_attendance_session_rule_idx  INT,
  class_attendance_session_rule_json JSONB,
  class_attendance_session_rule_hash TEXT,

  -- occurrence
  class_attendance_session_date      DATE NOT NULL,
  class_attendance_session_starts_at TIMESTAMPTZ,
  class_attendance_session_ends_at   TIMESTAMPTZ,

  -- lifecycle
  class_attendance_session_status            session_status_enum NOT NULL DEFAULT 'scheduled',
  class_attendance_session_attendance_status TEXT NOT NULL DEFAULT 'open',  -- open|closed
  class_attendance_session_locked            BOOLEAN NOT NULL DEFAULT FALSE,

  -- override flags & original
  class_attendance_session_is_override BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_session_is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_session_original_start_at TIMESTAMPTZ,
  class_attendance_session_original_end_at   TIMESTAMPTZ,
  class_attendance_session_kind TEXT,
  class_attendance_session_override_reason TEXT,

  -- override resource (opsional)
  class_attendance_session_teacher_id    UUID REFERENCES school_teachers(school_teacher_id) ON DELETE SET NULL,
  class_attendance_session_class_room_id UUID REFERENCES class_rooms(class_room_id)         ON DELETE SET NULL,
  class_attendance_session_csst_id       UUID REFERENCES class_section_subject_teachers(class_section_subject_teacher_id) ON DELETE SET NULL,

  -- info & rekap
  class_attendance_session_title         TEXT,
  class_attendance_session_general_info  TEXT NOT NULL DEFAULT '',
  class_attendance_session_note          TEXT,
  class_attendance_session_present_count INT,
  class_attendance_session_absent_count  INT,
  class_attendance_session_late_count    INT,
  class_attendance_session_excused_count INT,
  class_attendance_session_sick_count    INT,
  class_attendance_session_leave_count   INT,

  -- audit
  class_attendance_session_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_deleted_at TIMESTAMPTZ,

  -- checks
  CONSTRAINT chk_cas_time_order
    CHECK (
      class_attendance_session_starts_at IS NULL
      OR class_attendance_session_ends_at IS NULL
      OR class_attendance_session_ends_at >= class_attendance_session_starts_at
    ),
  CONSTRAINT chk_cas_time_order_original
    CHECK (
      class_attendance_session_original_start_at IS NULL
      OR class_attendance_session_original_end_at IS NULL
      OR class_attendance_session_original_end_at >= class_attendance_session_original_start_at
    ),
  CONSTRAINT chk_cas_attendance_status
    CHECK (class_attendance_session_attendance_status IN ('open','closed'))
);

-- Indexes (sessions)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_id_tenant
  ON class_attendance_sessions (class_attendance_session_id, class_attendance_session_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_school_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_schedule_id,
    class_attendance_session_date
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_sched_start
  ON class_attendance_sessions (
    class_attendance_session_schedule_id,
    class_attendance_session_starts_at
  );

CREATE INDEX IF NOT EXISTS idx_cas_school_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_session_schedule_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_teacher_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_canceled_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_is_canceled,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_override_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_is_override,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_override_event_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_csst_id
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_rule_alive
  ON class_attendance_sessions (class_attendance_session_rule_idx)
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_cas_created_at
  ON class_attendance_sessions USING BRIN (class_attendance_session_created_at);
