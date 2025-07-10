CREATE TABLE IF NOT EXISTS masjids (
    masjid_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_name VARCHAR(100) NOT NULL,
    masjid_bio_short TEXT,
    masjid_location TEXT,
    masjid_latitude DECIMAL(9,6),
    masjid_longitude DECIMAL(9,6),
    masjid_image_url TEXT,
    masjid_slug VARCHAR(100) UNIQUE NOT NULL,
    masjid_is_verified BOOLEAN DEFAULT FALSE,
    masjid_instagram_url TEXT,
    masjid_whatsapp_url TEXT,
    masjid_youtube_url TEXT,
    masjid_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    masjid_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    masjid_deleted_at TIMESTAMP
);

-- 🔍 Index untuk pencarian lokasi / geo
CREATE INDEX IF NOT EXISTS idx_masjids_location ON masjids(masjid_location);
CREATE INDEX IF NOT EXISTS idx_masjids_latlong ON masjids(masjid_latitude, masjid_longitude);

-- 🔍 Index untuk lookup berdasarkan slug (frontend URL)
CREATE UNIQUE INDEX IF NOT EXISTS idx_masjids_slug ON masjids(masjid_slug);

CREATE TABLE IF NOT EXISTS user_follow_masjid (
    follow_user_id UUID NOT NULL,
    follow_masjid_id UUID NOT NULL,
    follow_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follow_user_id, follow_masjid_id),
    FOREIGN KEY (follow_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (follow_masjid_id) REFERENCES masjids(masjid_id) ON DELETE CASCADE
);

-- 🔍 Index untuk cepat cari siapa saja follow masjid tertentu
CREATE INDEX IF NOT EXISTS idx_follow_masjid_id ON user_follow_masjid(follow_masjid_id);


CREATE TABLE IF NOT EXISTS masjids_profiles (
    masjid_profile_id SERIAL PRIMARY KEY,
    masjid_profile_story TEXT,
    masjid_profile_visi TEXT,
    masjid_profile_misi TEXT,
    masjid_profile_other TEXT,
    masjid_profile_founded_year INT,
    masjid_profile_masjid_id UUID UNIQUE REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_profile_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    masjid_profile_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    masjid_profile_deleted_at TIMESTAMP
);

-- 🔍 Index untuk lookup cepat ke profil masjid
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_masjid_id ON masjids_profiles(masjid_profile_masjid_id);
