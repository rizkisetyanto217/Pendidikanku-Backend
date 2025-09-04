BEGIN;

-- ========== Bersihkan trigger & function validasi ==========
DROP TRIGGER IF EXISTS trg_class_schedules_validate_links ON class_schedules;
DROP FUNCTION IF EXISTS fn_class_schedules_validate_links();

-- ========== Drop exclusion constraints (opsional; tabel drop juga akan menghapus) ==========
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_section_overlap;
ALTER TABLE IF EXISTS class_schedules DROP CONSTRAINT IF EXISTS excl_sched_room_overlap;

-- ========== Drop indeks eksplisit ==========
DROP INDEX IF EXISTS idx_class_schedules_active;
DROP INDEX IF EXISTS idx_class_schedules_class_subject;
DROP INDEX IF EXISTS idx_class_schedules_room_dow;
DROP INDEX IF EXISTS idx_class_schedules_section_dow_time;
DROP INDEX IF EXISTS idx_class_schedules_tenant_dow;

-- ========== Drop table ==========
DROP TABLE IF EXISTS class_schedules;

-- ========== (Opsional) Drop enum jika tidak lagi dipakai ==========
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    -- Hanya drop jika tidak ada kolom yang masih menggunakan tipe ini
    IF NOT EXISTS (
      SELECT 1
      FROM pg_attribute a
      JOIN pg_class c ON a.attrelid = c.oid
      WHERE a.atttypid = 'session_status_enum'::regtype
        AND c.relkind = 'r'  -- table
    ) THEN
      DROP TYPE session_status_enum;
    END IF;
  END IF;
END$$;

-- Catatan:
-- - Extensions (pgcrypto, btree_gist) TIDAK di-drop agar aman untuk objek lain.

COMMIT;
