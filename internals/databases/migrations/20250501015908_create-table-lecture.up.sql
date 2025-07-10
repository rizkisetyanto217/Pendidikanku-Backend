CREATE TABLE IF NOT EXISTS lectures (
    lecture_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lecture_title VARCHAR(255) NOT NULL,
    lecture_description TEXT,
    total_lecture_sessions INTEGER, -- Total sesi jika kajian terbatas
    lecture_is_recurring BOOLEAN DEFAULT FALSE, -- Apakah kajian berulang?
    lecture_recurrence_interval INTEGER, -- Jumlah hari antar pertemuan jika berulang
    lecture_image_url TEXT, -- Gambar utama kajian
    lecture_teachers JSONB, -- List pengajar: [{"id": "...", "name": "..."}, ...]
    lecture_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    lecture_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Untuk pencarian berdasarkan masjid
CREATE INDEX IF NOT EXISTS idx_lecture_masjid_id ON lectures(lecture_masjid_id);

-- Untuk sorting berdasarkan waktu pembuatan
CREATE INDEX IF NOT EXISTS idx_lecture_created_at ON lectures(lecture_created_at DESC);

-- Kombinasi pencarian per masjid urut waktu (opsional tapi disarankan)
CREATE INDEX IF NOT EXISTS idx_lecture_masjid_created_at ON lectures(lecture_masjid_id, lecture_created_at DESC);


-- Tabel user_lectures: relasi user mengikuti kajian
CREATE TABLE IF NOT EXISTS user_lectures (
    user_lecture_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_lecture_grade_result INT, -- nilai hasil jika ada evaluasi
    user_lecture_lecture_id UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
    user_lecture_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_lecture_total_completed_sessions INT DEFAULT 0, -- jumlah sesi yang sudah dihadiri
    user_lecture_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_lecture_lecture_id, user_lecture_user_id) -- satu user tidak bisa dua kali ikut satu kajian
);

-- Index untuk pencarian cepat
CREATE INDEX IF NOT EXISTS idx_user_lecture_lecture_id ON user_lectures(user_lecture_lecture_id);
CREATE INDEX IF NOT EXISTS idx_user_lecture_user_id ON user_lectures(user_lecture_user_id);
