-- ============================
-- NOTIFICATION USERS (turunkan dulu child)
-- ============================
DROP INDEX IF EXISTS idx_nu_notification_id;
DROP INDEX IF EXISTS idx_nu_user_masjid_sent;
DROP INDEX IF EXISTS idx_nu_user_unread;

DROP TRIGGER IF EXISTS trg_notification_users_set_read_at ON notification_users;
DROP FUNCTION IF EXISTS set_notification_users_read_at();

DROP TABLE IF EXISTS notification_users;

-- ============================
-- NOTIFICATIONS
-- ============================
DROP INDEX IF EXISTS idx_notifications_tags_gin;
DROP INDEX IF EXISTS idx_notifications_global_created;
DROP INDEX IF EXISTS idx_notifications_masjid_type_created;
DROP INDEX IF EXISTS idx_notifications_masjid_created;

DROP TRIGGER IF EXISTS trg_notifications_set_updated_at ON notifications;
DROP FUNCTION IF EXISTS set_notifications_updated_at();

DROP TABLE IF EXISTS notifications;

-- Extension pgcrypto tidak saya drop karena bisa dipakai objek lain juga.
