BEGIN;

-- Drop trigger & function
DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_stats ON masjid_stats;
DROP FUNCTION IF EXISTS set_updated_at_masjid_stats();

-- Drop indexes (yang kita buat manual)
DROP INDEX IF EXISTS idx_masjid_stats_updated_at_desc;
DROP INDEX IF EXISTS idx_masjid_stats_masjid_id_updated_at;

-- UNIQUE index di kolom masjid_stats_masjid_id akan ikut hilang saat tabel di-drop
DROP TABLE IF EXISTS masjid_stats;

COMMIT;
