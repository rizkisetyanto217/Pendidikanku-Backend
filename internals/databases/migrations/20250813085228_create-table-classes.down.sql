BEGIN;

-- Urutan: drop child dulu (punya FK ke parent)
DROP TABLE IF EXISTS classes;

-- Lalu parent
DROP TABLE IF EXISTS class_parents;

-- Hapus enum mode (hanya dipakai oleh classes di migration ini)
DROP TYPE IF EXISTS class_delivery_mode_enum;

-- Catatan: billing_cycle_enum sengaja tidak di-drop
-- karena bisa dipakai tabel lain di sistem.

COMMIT;
