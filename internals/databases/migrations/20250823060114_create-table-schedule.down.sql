-- +migrate Down
-- =========================================================
-- DOWN — Schedules, Rules, Holidays (safe, no EXTENSION drops)
-- =========================================================

/* ---------------------------------------------------------
   1) CHILD: class_schedule_rules
   (Indexes akan terhapus otomatis bersama tabel)
--------------------------------------------------------- */
DROP TABLE IF EXISTS class_schedule_rules;

/* ---------------------------------------------------------
   2) PARENT: class_schedules
--------------------------------------------------------- */
-- (opsional) explicit drop indexes, tidak wajib:
-- DROP INDEX IF EXISTS brin_class_schedules_created_at;
-- DROP INDEX IF EXISTS idx_class_schedules_date_bounds_alive;
-- DROP INDEX IF EXISTS idx_class_schedules_active_alive;
-- DROP INDEX IF EXISTS idx_class_schedules_tenant_alive;
-- DROP INDEX IF EXISTS gin_class_schedules_slug_trgm_alive;
-- DROP INDEX IF EXISTS uq_class_schedules_slug_per_tenant_alive;

DROP TABLE IF EXISTS class_schedules;

/* ---------------------------------------------------------
   3) school_holidays (independent of others here)
--------------------------------------------------------- */
-- (opsional) explicit drop indexes:
-- DROP INDEX IF EXISTS brin_school_holidays_created_at;
-- DROP INDEX IF EXISTS gin_school_holidays_slug_trgm_alive;
-- DROP INDEX IF EXISTS idx_school_holidays_date_range_alive;
-- DROP INDEX IF EXISTS idx_school_holidays_tenant_alive;
-- DROP INDEX IF EXISTS uq_school_holidays_slug_per_tenant_alive;

DROP TABLE IF EXISTS school_holidays;

/* ---------------------------------------------------------
   4) national_holidays (independent of others here)
--------------------------------------------------------- */
-- (opsional) explicit drop indexes:
-- DROP INDEX IF EXISTS brin_national_holidays_created_at;
-- DROP INDEX IF EXISTS gin_national_holidays_slug_trgm_alive;
-- DROP INDEX IF EXISTS idx_national_holidays_date_range_alive;
-- DROP INDEX IF EXISTS uq_national_holidays_slug_alive;

DROP TABLE IF EXISTS national_holidays;

/* ---------------------------------------------------------
   5) ENUMS — drop only if no remaining dependencies
      (Safe guard: cek pg_depend, jangan pakai CASCADE)
--------------------------------------------------------- */
DO $$
BEGIN
  -- week_parity_enum
  IF EXISTS (
    SELECT 1
    FROM pg_type t
    WHERE t.typname = 'week_parity_enum'
      AND NOT EXISTS (
        SELECT 1
        FROM pg_depend d
        WHERE d.refobjid = t.oid
          AND d.deptype IN ('n','a','i','e','p') -- any dependency
      )
  ) THEN
    EXECUTE 'DROP TYPE week_parity_enum';
  END IF;

  -- session_status_enum
  IF EXISTS (
    SELECT 1
    FROM pg_type t
    WHERE t.typname = 'session_status_enum'
      AND NOT EXISTS (
        SELECT 1
        FROM pg_depend d
        WHERE d.refobjid = t.oid
          AND d.deptype IN ('n','a','i','e','p')
      )
  ) THEN
    EXECUTE 'DROP TYPE session_status_enum';
  END IF;
END$$;

-- Catatan:
-- - Tidak men-DROP EXTENSIONS (pgcrypto/pg_trgm/btree_gin) agar terhindar dari error dependensi.
-- - Jika ada objek lain di project yang masih memakai enum di atas, blok DO akan membiarkan enum tetap ada.
