-- +migrate Down

-- 1) Matikan trigger lalu hapus func
DROP TRIGGER IF EXISTS trg_cas_set_updated_at ON class_attendance_settings;
DROP FUNCTION IF EXISTS set_class_attendance_setting_updated_at();

-- 2) Hapus constraint CHECK & (opsional) FK audit ke users jika ada
ALTER TABLE class_attendance_settings
  DROP CONSTRAINT IF EXISTS ck_cas_require_implies_enable,
  DROP CONSTRAINT IF EXISTS fk_cas_created_by,
  DROP CONSTRAINT IF EXISTS fk_cas_updated_by;

-- 3) Hapus index (unik & non-unik)
DROP INDEX IF EXISTS uq_cas_masjid_unique;
DROP INDEX IF EXISTS idx_cas_masjid_id;
DROP INDEX IF EXISTS idx_cas_deleted_at;

-- 4) Terakhir: drop tabel
DROP TABLE IF EXISTS class_attendance_settings;
