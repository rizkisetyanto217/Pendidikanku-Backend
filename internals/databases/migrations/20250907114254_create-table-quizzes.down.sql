-- +migrate Down

-- Urutan: child dulu, baru parent
DROP TABLE IF EXISTS quiz_questions CASCADE;
DROP TABLE IF EXISTS quizzes CASCADE;

-- Catatan:
-- - EXTENSION pgcrypto dan pg_trgm TIDAK di-drop,
--   karena kemungkinan besar dipakai juga oleh tabel/module lain.
--   Kalau mau benar-benar bersih dan yakin tidak dipakai apa-apa lagi,
--   bisa manual:
--   DROP EXTENSION IF EXISTS pg_trgm;
--   DROP EXTENSION IF EXISTS pgcrypto;
