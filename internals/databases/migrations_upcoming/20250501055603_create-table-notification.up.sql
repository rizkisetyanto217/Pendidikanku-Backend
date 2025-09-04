-- Dependensi untuk gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================
-- NOTIFICATIONS
-- ============================
CREATE TABLE IF NOT EXISTS notifications (
    notification_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_title       VARCHAR(255) NOT NULL,
    notification_description TEXT,

    -- tipe ditentukan di level kode
    notification_type        INT NOT NULL,

    -- NULL = notifikasi global (tidak terikat masjid tertentu)
    notification_masjid_id   UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,

    -- tag untuk filter/search
    notification_tags        TEXT[] NOT NULL DEFAULT '{}',

    notification_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_deleted_at  TIMESTAMPTZ
);

-- Indexing yang efektif untuk query umum (alive only):
-- 1) List per masjid (ORDER BY terbaru)
CREATE INDEX IF NOT EXISTS idx_notifications_masjid_created_alive
  ON notifications (notification_masjid_id, notification_created_at DESC)
  WHERE notification_deleted_at IS NULL;

-- 2) Filter per masjid + tipe (ORDER BY terbaru)
CREATE INDEX IF NOT EXISTS idx_notifications_masjid_type_created_alive
  ON notifications (notification_masjid_id, notification_type, notification_created_at DESC)
  WHERE notification_deleted_at IS NULL;

-- 3) Notifikasi global (masjid_id IS NULL) terbaru
CREATE INDEX IF NOT EXISTS idx_notifications_global_created_alive
  ON notifications (notification_created_at DESC)
  WHERE notification_masjid_id IS NULL AND notification_deleted_at IS NULL;

-- 4) Pencarian/penyaringan berdasarkan tag
CREATE INDEX IF NOT EXISTS idx_notifications_tags_gin_alive
  ON notifications USING GIN (notification_tags)
  WHERE notification_deleted_at IS NULL;


-- ============================
-- NOTIFICATION USERS
-- ============================
CREATE TABLE IF NOT EXISTS notification_users (
    notification_users_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    notification_users_notification_id UUID NOT NULL REFERENCES notifications(notification_id) ON DELETE CASCADE,
    notification_users_user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- redundan untuk filter cepat per masjid
    notification_users_masjid_id       UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

    -- status baca
    notification_users_read            BOOLEAN NOT NULL DEFAULT FALSE,
    notification_users_sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_users_read_at         TIMESTAMPTZ NULL,

    -- unik per user per notifikasi
    UNIQUE (notification_users_notification_id, notification_users_user_id)
);

-- Trigger: set read_at ketika read=true; kosongkan saat read=false
CREATE OR REPLACE FUNCTION set_notification_users_read_at()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.notification_users_read IS TRUE AND (OLD.notification_users_read IS DISTINCT FROM TRUE) THEN
    NEW.notification_users_read_at := COALESCE(NEW.notification_users_read_at, NOW());
  ELSIF NEW.notification_users_read IS NOT TRUE THEN
    NEW.notification_users_read_at := NULL;
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_notification_users_set_read_at ON notification_users;
CREATE TRIGGER trg_notification_users_set_read_at
BEFORE UPDATE ON notification_users
FOR EACH ROW EXECUTE FUNCTION set_notification_users_read_at();

-- Indexing untuk akses cepat:
-- 1) Hitung/list unread per user
CREATE INDEX IF NOT EXISTS idx_nu_user_unread
  ON notification_users (notification_users_user_id)
  WHERE notification_users_read = FALSE;

-- 2) List notifikasi user per masjid (terbaru dulu)
CREATE INDEX IF NOT EXISTS idx_nu_user_masjid_sent
  ON notification_users (notification_users_user_id, notification_users_masjid_id, notification_users_sent_at DESC);

-- 3) Fanout lookup per notification_id
CREATE INDEX IF NOT EXISTS idx_nu_notification_id
  ON notification_users (notification_users_notification_id);
