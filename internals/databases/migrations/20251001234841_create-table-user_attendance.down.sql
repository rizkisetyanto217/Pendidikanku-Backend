-- +migrate Down
-- =========================================================
-- DOWN — USER ATTENDANCE SNAPSHOTS (UC-SST & UC-SEC)
-- Revert: drop indexes → partitions → parent → enums
-- =========================================================
BEGIN;

-- =========================
-- Drop INDEXES — UC-SST partition
-- =========================
DROP INDEX IF EXISTS idx_user_att_snap_ucsst_tenant_anchor;
DROP INDEX IF EXISTS brin_user_att_snap_ucsst_start;
DROP INDEX IF EXISTS idx_user_att_snap_ucsst_updated_at;

DROP INDEX IF EXISTS idx_user_att_snap_ucsst_week;
DROP INDEX IF EXISTS idx_user_att_snap_ucsst_month;
DROP INDEX IF EXISTS idx_user_att_snap_ucsst_semester;

DROP INDEX IF EXISTS gin_user_att_snap_ucsst_type_counts;
DROP INDEX IF EXISTS gin_user_att_snap_ucsst_method_counts;
DROP INDEX IF EXISTS gin_user_att_snap_ucsst_sessions_week_only;

-- =========================
-- Drop INDEXES — UC-SEC partition
-- =========================
DROP INDEX IF EXISTS idx_user_att_snap_ucsec_tenant_anchor;
DROP INDEX IF EXISTS brin_user_att_snap_ucsec_start;
DROP INDEX IF EXISTS idx_user_att_snap_ucsec_updated_at;

DROP INDEX IF EXISTS idx_user_att_snap_ucsec_week;
DROP INDEX IF EXISTS idx_user_att_snap_ucsec_month;
DROP INDEX IF EXISTS idx_user_att_snap_ucsec_semester;

DROP INDEX IF EXISTS gin_user_att_snap_ucsec_type_counts;
DROP INDEX IF EXISTS gin_user_att_snap_ucsec_method_counts;
DROP INDEX IF EXISTS gin_user_att_snap_ucsec_sessions_week_only;

-- =========================
-- Drop INDEXES — Parent
-- (akan otomatis terhapus saat table drop, tapi eksplisit aman)
-- =========================
DROP INDEX IF EXISTS idx_user_att_snap_tenant_grain_start;
DROP INDEX IF EXISTS brin_user_att_snap_created;

-- =========================
-- Drop PARTITIONS
-- =========================
DROP TABLE IF EXISTS user_attendance_snapshots_ucsst;
DROP TABLE IF EXISTS user_attendance_snapshots_ucsec;

-- =========================
-- Drop PARENT
-- =========================
DROP TABLE IF EXISTS user_attendance_snapshots;

-- =========================
-- Drop ENUM types (pastikan tidak dipakai objek lain)
-- =========================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_attendance_snapshots_scope') THEN
    DROP TYPE user_attendance_snapshots_scope;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_attendance_snapshots_grain') THEN
    DROP TYPE user_attendance_snapshots_grain;
  END IF;
END$$;

COMMIT;
