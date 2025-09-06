-- =========================================================
-- ====================   D O W N   ========================
-- =========================================================

BEGIN;

-- Child tables dulu
DROP TABLE IF EXISTS user_follow_masjid;
DROP TABLE IF EXISTS masjids_profiles;

-- Parent
DROP TABLE IF EXISTS masjids;

COMMIT;

-- OPSIONAL (jalankan hanya jika yakin enum tidak dipakai objek lain):
-- DROP TYPE IF EXISTS verification_status_enum;

-- Extensions sengaja tidak di-drop (bisa dipakai objek lain).
