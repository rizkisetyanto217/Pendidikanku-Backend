-- =========================================
-- DOWN: masjid_urls + enum + objek terkait
-- =========================================
BEGIN;

-- 1) View
DROP VIEW IF EXISTS v_masjid_primary_urls;

-- 2) Triggers (harus drop sebelum table di-drop)
DROP TRIGGER IF EXISTS trg_masjid_urls_single_primary_upd ON masjid_urls;
DROP TRIGGER IF EXISTS trg_masjid_urls_single_primary_ins ON masjid_urls;

-- 3) Functions
DROP FUNCTION IF EXISTS ensure_single_primary_masjid_url();

-- 4) Indexes (opsional; akan ikut hilang saat tabel di-drop, tapi eksplisit lebih rapi)
DROP INDEX IF EXISTS idx_masjid_urls_delete_due;
DROP INDEX IF EXISTS uq_masjid_urls_singleton_types;
DROP INDEX IF EXISTS uq_masjid_urls_primary_per_type;
DROP INDEX IF EXISTS uq_masjid_urls_file_ci;
DROP INDEX IF EXISTS idx_masjid_urls_masjid;

-- 5) Table
DROP TABLE IF EXISTS masjid_urls;

-- 6) Enum type (akan sukses jika tidak dipakai objek lain)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'masjid_url_type_enum') THEN
    -- Pastikan tidak ada kolom lain yang masih memakai enum ini
    IF NOT EXISTS (
      SELECT 1
      FROM pg_attribute a
      JOIN pg_class c ON c.oid = a.attrelid AND c.relkind IN ('r','p','v','m','f')
      WHERE a.atttypid = (SELECT oid FROM pg_type WHERE typname = 'masjid_url_type_enum')
        AND a.attnum > 0
        AND NOT a.attisdropped
    ) THEN
      EXECUTE 'DROP TYPE masjid_url_type_enum';
    END IF;
  END IF;
END$$;

COMMIT;
