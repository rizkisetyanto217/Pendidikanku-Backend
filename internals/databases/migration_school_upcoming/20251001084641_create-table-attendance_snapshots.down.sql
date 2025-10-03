-- +migrate Down
-- =========================================
-- DOWN Migration — Revert User Class Session Attendance (incl. Snapshots)
-- =========================================
BEGIN;

-- 1) Hapus tabel snapshot rekap (mingguan/bulanan/semester)
DROP TABLE IF EXISTS user_class_session_attendance_snapshots;


COMMIT;
