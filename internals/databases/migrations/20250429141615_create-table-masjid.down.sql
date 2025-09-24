BEGIN;

-- Drop tabel yang bergantung dulu
DROP TABLE IF EXISTS masjid_profiles;

-- Lalu tabel induknya
DROP TABLE IF EXISTS masjids;

-- (Opsional aman) Hapus enum tenant_profile_enum jika sudah tidak dipakai
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tenant_profile_enum') THEN
    -- Pastikan tidak ada dependency lagi pada enum tsb
    IF NOT EXISTS (
      SELECT 1
      FROM pg_depend d
      JOIN pg_type t ON d.refobjid = t.oid
      WHERE t.typname = 'tenant_profile_enum'
    ) THEN
      EXECUTE 'DROP TYPE tenant_profile_enum';
    END IF;
  END IF;
END$$;

COMMIT;

-- Catatan:
-- - Index yang dibuat pada tabel akan otomatis ter-drop bersama tabelnya.
-- - Extensions dibiarkan tetap ada agar tidak mengganggu migrasi lain.
-- - verification_status_enum tidak dihapus karena bisa dipakai objek lain.
