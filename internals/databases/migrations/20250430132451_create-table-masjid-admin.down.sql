-- masjid_teachers: drop trigger & function, index, table
DROP TRIGGER  IF EXISTS trg_set_updated_at_masjid_teachers ON masjid_teachers;
DROP FUNCTION IF EXISTS set_updated_at_masjid_teachers();

DROP INDEX IF EXISTS idx_masjid_teachers_user_id_alive;
DROP INDEX IF EXISTS ux_masjid_teachers_masjid_user_alive;
-- DROP INDEX IF EXISTS idx_masjid_teachers_masjid_id_alive; -- jika diaktifkan

DROP TABLE IF EXISTS masjid_teachers;

-- masjid_admins: drop trigger & function, index, table
DROP TRIGGER  IF EXISTS trg_set_updated_at_masjid_admins ON masjid_admins;
DROP FUNCTION IF EXISTS set_updated_at_masjid_admins();

DROP INDEX IF EXISTS idx_masjid_admins_masjid_active_alive;
DROP INDEX IF EXISTS idx_masjid_admins_user_active_alive;
DROP INDEX IF EXISTS ux_masjid_admins_masjid_user_alive;

DROP TABLE IF EXISTS masjid_admins;
