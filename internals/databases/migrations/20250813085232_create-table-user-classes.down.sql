BEGIN;

-- Hapus tabel child dulu (user_class_sections), karena FK ke user_classes
DROP TABLE IF EXISTS user_class_sections CASCADE;

-- Lalu hapus tabel induk
DROP TABLE IF EXISTS user_classes CASCADE;

COMMIT;
