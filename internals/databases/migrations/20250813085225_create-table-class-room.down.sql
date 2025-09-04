BEGIN;

-- Drop indeks eksplisit (opsional; tabel drop juga akan menghapus)
DROP INDEX IF EXISTS idx_class_rooms_location_trgm;
DROP INDEX IF EXISTS idx_class_rooms_name_trgm;
DROP INDEX IF EXISTS idx_class_rooms_features_gin;
DROP INDEX IF EXISTS idx_class_rooms_tenant_active;

-- Unique indexes
DROP INDEX IF EXISTS uq_class_rooms_tenant_code_ci;
DROP INDEX IF EXISTS uq_class_rooms_tenant_name_ci;

-- Drop table
DROP TABLE IF EXISTS class_rooms;

-- Catatan:
-- - Extensions (pgcrypto, pg_trgm) TIDAK di-drop agar tidak berdampak ke objek lain.

COMMIT;
