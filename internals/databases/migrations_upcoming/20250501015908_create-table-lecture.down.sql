
-- =========================
-- ========= DOWN ==========
-- =========================
BEGIN;

-- Hapus triggers dulu
DROP TRIGGER IF EXISTS trg_lecture_schedules_touch ON lecture_schedules;
DROP TRIGGER IF EXISTS trg_user_lectures_touch ON user_lectures;
DROP TRIGGER IF EXISTS trg_lectures_touch ON lectures;

-- Hapus index (aman bila belum ada)
-- lecture_schedules
DROP INDEX IF EXISTS idx_lecture_schedules_place_trgm;
DROP INDEX IF EXISTS idx_lecture_schedules_title_trgm;
DROP INDEX IF EXISTS idx_lecture_schedules_tsv_gin;
DROP INDEX IF EXISTS ux_lecture_schedules_unique_slot;
DROP INDEX IF EXISTS idx_lecture_schedules_day_time_live;
DROP INDEX IF EXISTS idx_lecture_schedules_lecture_id;

-- user_lectures
DROP INDEX IF EXISTS idx_user_lectures_paid_partial;
DROP INDEX IF EXISTS idx_user_lectures_user_lecture_unique;
DROP INDEX IF EXISTS idx_user_lectures_masjid_id;
DROP INDEX IF EXISTS idx_user_lectures_user_id;
DROP INDEX IF EXISTS idx_user_lectures_lecture_id;

-- lectures
DROP INDEX IF EXISTS idx_lectures_slug_trgm;
DROP INDEX IF EXISTS idx_lectures_title_trgm;
DROP INDEX IF EXISTS idx_lectures_tsv_gin;
DROP INDEX IF EXISTS idx_lectures_teachers_gin;
DROP INDEX IF EXISTS idx_lectures_masjid_recent_live;
DROP INDEX IF EXISTS idx_lectures_masjid_active_recent_live;
DROP INDEX IF EXISTS idx_lectures_created_at_desc;
DROP INDEX IF EXISTS idx_lectures_masjid_id;
DROP INDEX IF EXISTS ux_lectures_slug_ci;

-- Drop generated columns (jaga urutan sebelum DROP TABLE)
ALTER TABLE IF EXISTS lecture_schedules DROP COLUMN IF EXISTS lecture_schedules_search_tsv;
ALTER TABLE IF EXISTS lectures DROP COLUMN IF EXISTS lecture_search_tsv;

-- Drop tables (anak → induk)
DROP TABLE IF EXISTS lecture_schedules;
DROP TABLE IF EXISTS user_lectures;
DROP TABLE IF EXISTS lectures;

-- Drop functions terakhir
DROP FUNCTION IF EXISTS fn_touch_updated_at_lecture_schedules();
DROP FUNCTION IF EXISTS fn_touch_updated_at_user_lectures();
DROP FUNCTION IF EXISTS fn_touch_updated_at();

-- (Opsional) Drop ekstensi bila memang dibuat khusus untuk schema ini
-- HATI‑HATI: bisa berdampak ke objek lain.
-- DROP EXTENSION IF EXISTS btree_gin CASCADE;
-- DROP EXTENSION IF EXISTS pg_trgm CASCADE;
-- DROP EXTENSION IF EXISTS pgcrypto CASCADE;

COMMIT;