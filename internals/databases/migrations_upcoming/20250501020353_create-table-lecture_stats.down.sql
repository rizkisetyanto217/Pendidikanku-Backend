
-- =========================
-- ========= DOWN ==========
-- =========================
BEGIN;

-- Lepas trigger dari user_lectures
DROP TRIGGER IF EXISTS trg_user_lectures_recalc_stats_aiud ON user_lectures;
DROP FUNCTION IF EXISTS trg_user_lectures_recalc_stats_fn();

-- Hapus trigger & function updated_at
DROP TRIGGER IF EXISTS trg_touch_lecture_stats_updated_at ON lecture_stats;
DROP FUNCTION IF EXISTS fn_touch_lecture_stats_updated_at();

-- Hapus function recalc
DROP FUNCTION IF EXISTS recalc_lecture_stats(UUID);

-- Hapus index
DROP INDEX IF EXISTS idx_lecture_stats_masjid_recent;
DROP INDEX IF EXISTS idx_lecture_stats_masjid_id;
DROP INDEX IF EXISTS idx_lecture_stats_lecture_id;

-- Hapus tabel
DROP TABLE IF EXISTS lecture_stats;

COMMIT;