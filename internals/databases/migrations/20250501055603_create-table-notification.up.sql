CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_title VARCHAR(255) NOT NULL,
    notification_description TEXT,
    
    -- Integer yang mewakili tipe notifikasi (ditentukan enum di level kode)
    notification_type INT NOT NULL,
    
    notification_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    notification_tags TEXT[], -- Array tag misalnya ['kajian', 'informasi']
    
    notification_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    notification_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_type ON notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_notification_masjid_id ON notifications(notification_masjid_id);


-- ============================
-- NOTIFICATION USERS
-- ============================
CREATE TABLE IF NOT EXISTS notification_users (
    notification_users_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Relasi
    notification_users_notification_id UUID NOT NULL REFERENCES notifications(notification_id) ON DELETE CASCADE,
    notification_users_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Informasi Masjid (redundan tapi efisien untuk filter cepat)
    notification_users_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

    -- Status
    notification_users_read BOOLEAN DEFAULT FALSE,
    notification_users_sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    notification_users_read_at TIMESTAMP,

    -- Unik per user per notifikasi
    UNIQUE(notification_users_notification_id, notification_users_user_id)
);

-- 🔍 Index
CREATE INDEX IF NOT EXISTS idx_notification_users_user_id ON notification_users(notification_users_user_id);
CREATE INDEX IF NOT EXISTS idx_notification_users_notification_id ON notification_users(notification_users_notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_users_read ON notification_users(notification_users_read);
CREATE INDEX IF NOT EXISTS idx_notification_users_masjid_id ON notification_users(notification_users_masjid_id);