-- =========================================================
-- DOWN: rollback masjids + relasinya
-- Catatan:
-- - Triggers ikut hilang saat tabelnya di-DROP, tapi functions perlu dihapus manual.
-- - Extensions (pgcrypto, pg_trgm, cube, earthdistance) TIDAK di-drop.
-- =========================================================

-- =========================
-- 1) Putus & hapus tabel relasi
-- =========================

-- USER_FOLLOW_MASJID
DROP TABLE IF EXISTS user_follow_masjid;

-- MASJIDS_PROFILES (+ function trigger-nya)
DROP TRIGGER IF EXISTS trg_set_updated_at_masjids_profiles ON masjids_profiles;
DROP FUNCTION IF EXISTS set_updated_at_masjids_profiles();

DROP TABLE IF EXISTS masjids_profiles;

-- =========================
-- 2) Hapus trigger & function pada MASJIDS
-- =========================
DROP TRIGGER IF EXISTS trg_handle_masjid_image_trash ON masjids;
DROP TRIGGER IF EXISTS trg_sync_verification ON masjids;
DROP TRIGGER IF EXISTS trg_set_updated_at_masjids ON masjids;

-- Functions
DROP FUNCTION IF EXISTS handle_masjid_image_trash();
DROP FUNCTION IF EXISTS sync_masjid_verification_flags();
DROP FUNCTION IF EXISTS set_updated_at_masjids();

-- =========================
-- 3) Hapus tabel utama
-- =========================
DROP TABLE IF EXISTS masjids;

-- =========================
-- 4) Drop ENUM (jika sudah tidak dipakai)
-- =========================
-- Jika tipe ini dipakai tabel lain, perintah ini akan gagal.
-- Pastikan hanya skema ini yang menggunakannya sebelum menjalankan down.
DROP TYPE IF EXISTS verification_status_enum;
