CREATE TABLE IF NOT EXISTS masjid_admins (
    masjid_admins_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_admins_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_admins_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    masjid_admins_is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(masjid_admins_id, masjid_admins_user_id)
);


-- Index untuk pencarian cepat admin berdasarkan masjid
CREATE INDEX IF NOT EXISTS idx_admins_masjid_id ON masjid_admins(masjid_admins_id);

-- Index untuk pencarian cepat admin berdasarkan user
CREATE INDEX IF NOT EXISTS idx_admins_user_id ON masjid_admins(masjid_admins_user_id);

-- Index gabungan jika sering query WHERE masjid_id AND is_active
CREATE INDEX IF NOT EXISTS idx_admins_masjid_active ON masjid_admins(masjid_admins_id, masjid_admins_is_active);



-- Buat table masjid_teachers
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_teachers_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teachers_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  masjid_teachers_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  masjid_teachers_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT masjid_teachers_unique_masjid_user UNIQUE (masjid_teachers_masjid_id, masjid_teachers_user_id)
);

-- Index eksplisit
CREATE INDEX IF NOT EXISTS masjid_teachers_masjid_id_idx ON masjid_teachers(masjid_teachers_masjid_id);
CREATE INDEX IF NOT EXISTS masjid_teachers_user_id_idx ON masjid_teachers(masjid_teachers_user_id);
