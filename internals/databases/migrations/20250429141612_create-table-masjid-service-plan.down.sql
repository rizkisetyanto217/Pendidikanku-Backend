-- Hapus trigger & function
DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_service_plans ON masjid_service_plans;
DROP FUNCTION IF EXISTS set_updated_at_masjid_service_plans;

-- Hapus tabel utama
DROP TABLE IF EXISTS masjid_service_plans;
