-- +migrate Down
BEGIN;

-- =========================================================
-- 1) USER CSST SNAPSHOTS — drop indexes dulu
-- =========================================================
DROP INDEX IF EXISTS gin_ucsst_snap_submissions_week_only;
DROP INDEX IF EXISTS gin_ucsst_snap_grade_hist;
DROP INDEX IF EXISTS gin_ucsst_snap_status_counts;
DROP INDEX IF EXISTS idx_ucsst_snap_updated_at;
DROP INDEX IF EXISTS idx_ucsst_snap_month;
DROP INDEX IF EXISTS idx_ucsst_snap_week;
DROP INDEX IF EXISTS brin_ucsst_snap_start;
DROP INDEX IF EXISTS idx_ucsst_snap_tenant_ucsst;

-- Lalu drop table
DROP TABLE IF EXISTS user_csst_score_snapshots;

-- =========================================================
-- 2) UCSEC SNAPSHOTS — drop indexes dulu
-- =========================================================
DROP INDEX IF EXISTS gin_ucsec_snap_submissions_week_only;
DROP INDEX IF EXISTS gin_ucsec_snap_subject_briefs;
DROP INDEX IF EXISTS gin_ucsec_snap_grade_hist;
DROP INDEX IF EXISTS gin_ucsec_snap_status_counts;
DROP INDEX IF EXISTS idx_ucsec_snap_month;
DROP INDEX IF EXISTS idx_ucsec_snap_week;
DROP INDEX IF EXISTS idx_ucsec_snap_updated_at;
DROP INDEX IF EXISTS brin_ucsec_snap_start;
DROP INDEX IF EXISTS idx_ucsec_snap_tenant_ucsec;

-- Lalu drop table
DROP TABLE IF EXISTS ucsec_score_snapshots;

-- =========================================================
-- 3) ENUM type (dipakai keduanya) — aman di-drop setelah tables
-- =========================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'snapshot_period_grain') THEN
    -- pastikan tidak ada objek lain yang masih refer ke type ini
    DROP TYPE snapshot_period_grain;
  END IF;
END$$;

COMMIT;
