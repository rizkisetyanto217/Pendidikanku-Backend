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

  -- ✅ Properti pendaftaran & pembayaran dipindah ke sini
  lecture_is_registration_required BOOLEAN DEFAULT FALSE,
  lecture_is_paid BOOLEAN DEFAULT FALSE,
  lecture_price INT, -- harga jika berbayar
  lecture_payment_deadline TIMESTAMP, -- batas bayar jika ada

  lecture_capacity INT,
  lecture_is_public BOOLEAN DEFAULT TRUE, -- sesi tetap bisa publik/privat

  -- 📌 Status aktif
  lecture_is_active BOOLEAN DEFAULT TRUE,

  lecture_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  lecture_updated_at TIMESTAMP,
  lecture_deleted_at TIMESTAMP

);

CREATE INDEX IF NOT EXISTS idx_lecture_masjid_id ON lectures(lecture_masjid_id);
CREATE INDEX IF NOT EXISTS idx_lecture_created_at ON lectures(lecture_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lecture_masjid_active_created_at 
  ON lectures (lecture_masjid_id, lecture_is_active, lecture_created_at DESC);
-- ✅ Untuk query kajian terbaru per masjid (tanpa peduli aktif)
CREATE INDEX IF NOT EXISTS idx_lecture_masjid_created_at 
  ON lectures (lecture_masjid_id, lecture_created_at DESC);



CREATE TABLE IF NOT EXISTS user_lectures (
  user_lecture_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_lecture_grade_result INT, -- nilai hasil jika ada evaluasi
  user_lecture_lecture_id UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  user_lecture_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_total_completed_sessions INT DEFAULT 0, -- jumlah sesi yang sudah dihadiri

  -- 🔗 Masjid
  user_lecture_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Pendaftaran & Pembayaran
  user_lecture_is_registered BOOLEAN DEFAULT FALSE,
  user_lecture_has_paid BOOLEAN DEFAULT FALSE,
  user_lecture_paid_amount INT,
  user_lecture_payment_time TIMESTAMP,

  user_lecture_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  user_lecture_updated_at TIMESTAMP,

  UNIQUE(user_lecture_lecture_id, user_lecture_user_id)
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_lecture_lecture_id ON user_lectures(user_lecture_lecture_id);
CREATE INDEX IF NOT EXISTS idx_user_lecture_user_id ON user_lectures(user_lecture_user_id);
CREATE INDEX IF NOT EXISTS idx_user_lecture_masjid_id ON user_lectures(user_lecture_masjid_id);
