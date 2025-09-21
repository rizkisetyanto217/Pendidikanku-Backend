-- +migrate Down

-- 1) Child dulu
DROP TABLE IF EXISTS user_attendance_urls;

-- 2) Lalu parent
DROP TABLE IF EXISTS user_attendance;

-- 3) Terakhir master
DROP TABLE IF EXISTS user_attendance_type;
