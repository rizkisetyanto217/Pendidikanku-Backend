BEGIN;

-- Hapus tabel yang bergantung dulu
DROP TABLE IF EXISTS class_room_virtual_links;

-- Lalu tabel induk
DROP TABLE IF EXISTS class_rooms;

COMMIT;

-- (opsional) jika memang ingin ikut melepas extension:
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;
