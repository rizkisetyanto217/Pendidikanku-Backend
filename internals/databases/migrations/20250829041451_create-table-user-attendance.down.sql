-- =========================================
-- DOWN Migration (Refactor Final) — NO triggers / NO DO blocks
-- =========================================
BEGIN;


-- 2) Parent: attendance per siswa per sesi
DROP TABLE IF EXISTS user_attendance;

-- 3) Master: jenis attendance per masjid
DROP TABLE IF EXISTS user_attendance_type;

COMMIT;
