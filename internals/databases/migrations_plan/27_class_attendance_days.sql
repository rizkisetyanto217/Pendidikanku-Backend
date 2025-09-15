BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================================
-- 1) TABEL SNAPSHOT HARIAN PER KELAS
--    (diisi dari agregasi user_attendance + class_attendance_sessions)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_day_stats (
  class_attendance_day_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & kunci hari+kelas
  class_attendance_day_stats_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_attendance_day_stats_date            DATE NOT NULL,
  class_attendance_day_stats_class_section_id UUID NOT NULL,

  -- ringkasan harian per kelas (aturan: hadir minimal di salah satu mapel = hadir)
  class_attendance_day_stats_total_students  INT NOT NULL DEFAULT 0 CHECK (class_attendance_day_stats_total_students >= 0),
  class_attendance_day_stats_present         INT NOT NULL DEFAULT 0 CHECK (class_attendance_day_stats_present >= 0),
  class_attendance_day_stats_sick            INT NOT NULL DEFAULT 0 CHECK (class_attendance_day_stats_sick >= 0),
  class_attendance_day_stats_permit          INT NOT NULL DEFAULT 0 CHECK (class_attendance_day_stats_permit >= 0),
  class_attendance_day_stats_unexcused       INT NOT NULL DEFAULT 0 CHECK (class_attendance_day_stats_unexcused >= 0),

  -- metadata
  class_attendance_day_stats_sessions_count  INT NOT NULL DEFAULT 0,  -- banyaknya sesi (mapel) yang tercatat hari itu
  class_attendance_day_stats_computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_day_stats_deleted_at      TIMESTAMPTZ
);

-- unik per (masjid, kelas, tanggal) yg masih hidup
CREATE UNIQUE INDEX IF NOT EXISTS uq_cads_alive
  ON class_attendance_day_stats(
    class_attendance_day_stats_masjid_id,
    class_attendance_day_stats_class_section_id,
    class_attendance_day_stats_date
  )
  WHERE class_attendance_day_stats_deleted_at IS NULL;

-- query cepat per tanggal / per tenant
CREATE INDEX IF NOT EXISTS idx_cads_masjid_date_alive
  ON class_attendance_day_stats(class_attendance_day_stats_masjid_id, class_attendance_day_stats_date)
  WHERE class_attendance_day_stats_deleted_at IS NULL;

-- time-scan arsip besar (opsional)
CREATE INDEX IF NOT EXISTS brin_cads_computed_at
  ON class_attendance_day_stats USING BRIN (class_attendance_day_stats_computed_at);

-- =========================================================
-- 2) VIEW: RINGKAS HARIAN (normalisasi kolom untuk konsumsi dashboard)
-- =========================================================
CREATE OR REPLACE VIEW v_class_attendance_day_stats AS
SELECT
  class_attendance_day_stats_masjid_id       AS masjid_id,
  class_attendance_day_stats_class_section_id AS class_section_id,
  class_attendance_day_stats_date            AS date,
  class_attendance_day_stats_total_students  AS total_students,
  class_attendance_day_stats_present         AS present,
  class_attendance_day_stats_sick            AS sick,
  class_attendance_day_stats_permit          AS permit,
  class_attendance_day_stats_unexcused       AS unexcused,
  class_attendance_day_stats_sessions_count  AS sessions_count,
  class_attendance_day_stats_computed_at     AS computed_at
FROM class_attendance_day_stats
WHERE class_attendance_day_stats_deleted_at IS NULL;

-- =========================================================
-- 3) VIEW: RINGKAS MINGGUAN (ISO week via date_trunc('week'))
-- =========================================================
CREATE OR REPLACE VIEW v_class_attendance_week_stats AS
SELECT
  masjid_id,
  class_section_id,
  date_trunc('week', date)::date AS period_week,
  SUM(total_students) AS total_students,
  SUM(present)        AS present,
  SUM(sick)           AS sick,
  SUM(permit)         AS permit,
  SUM(unexcused)      AS unexcused
FROM v_class_attendance_day_stats
GROUP BY masjid_id, class_section_id, date_trunc('week', date);

-- =========================================================
-- 4) VIEW: RINGKAS BULANAN
-- =========================================================
CREATE OR REPLACE VIEW v_class_attendance_month_stats AS
SELECT
  masjid_id,
  class_section_id,
  date_trunc('month', date)::date AS period_month,
  SUM(total_students) AS total_students,
  SUM(present)        AS present,
  SUM(sick)           AS sick,
  SUM(permit)         AS permit,
  SUM(unexcused)      AS unexcused
FROM v_class_attendance_day_stats
GROUP BY masjid_id, class_section_id, date_trunc('month', date);

-- =========================================================
-- 5) FUNCTION: RINGKAS SEMESTER / RENTANG CUSTOM
-- =========================================================
CREATE OR REPLACE FUNCTION get_semester_attendance_stats(
  p_masjid_id UUID,
  p_class_section_id UUID,
  p_start DATE,
  p_end   DATE
)
RETURNS TABLE (
  masjid_id UUID,
  class_section_id UUID,
  period_start DATE,
  period_end   DATE,
  total_students BIGINT,
  present BIGINT,
  sick BIGINT,
  permit BIGINT,
  unexcused BIGINT,
  present_pct NUMERIC
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    p_masjid_id,
    p_class_section_id,
    p_start,
    p_end,
    COALESCE(SUM(d.total_students),0) AS total_students,
    COALESCE(SUM(d.present),0)        AS present,
    COALESCE(SUM(d.sick),0)           AS sick,
    COALESCE(SUM(d.permit),0)         AS permit,
    COALESCE(SUM(d.unexcused),0)      AS unexcused,
    CASE
      WHEN COALESCE(SUM(d.total_students),0) = 0 THEN NULL
      ELSE ROUND(100.0 * SUM(d.present)::numeric / NULLIF(SUM(d.total_students),0), 2)
    END AS present_pct
  FROM v_class_attendance_day_stats d
  WHERE d.masjid_id = p_masjid_id
    AND d.class_section_id = p_class_section_id
    AND d.date BETWEEN p_start AND p_end
  GROUP BY p_masjid_id, p_class_section_id, p_start, p_end;
END;
$$ LANGUAGE plpgsql STABLE;

COMMIT;
