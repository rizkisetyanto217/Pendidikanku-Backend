CREATE TABLE IF NOT EXISTS lectures (
  lecture_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_title VARCHAR(255) NOT NULL,
  lecture_slug VARCHAR(255) UNIQUE NOT NULL, -- ✅ Slug unik untuk URL
  lecture_description TEXT,
  total_lecture_sessions INTEGER, -- Total sesi jika kajian terbatas
  lecture_image_url TEXT, -- Gambar utama kajian
  lecture_teachers JSONB, -- List pengajar: [{"id": "...", "name": "..."}, ...]
  lecture_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- ✅ Properti pendaftaran & pembayaran dipindah ke sini
  lecture_is_registration_required BOOLEAN DEFAULT FALSE,
  lecture_is_paid BOOLEAN DEFAULT FALSE,
  lecture_price INT, -- harga jika berbayar
  lecture_payment_deadline TIMESTAMP, -- batas bayar jika ada

  lecture_capacity INT,

  -- 📌 Status aktif
  lecture_is_active BOOLEAN DEFAULT TRUE,
  lecture_is_certificate_generated BOOLEAN DEFAULT FALSE,

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
CREATE UNIQUE INDEX IF NOT EXISTS idx_lecture_slug ON lectures(lecture_slug);




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



CREATE TABLE IF NOT EXISTS lecture_schedules (
  lecture_schedules_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Relasi ke lecture
  lecture_schedules_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_schedules_title VARCHAR(255) NOT NULL,

  -- Hari & waktu rutin
  lecture_schedules_day_of_week INT NOT NULL,        -- 0 = Minggu, 1 = Senin, ..., 6 = Sabtu
  lecture_schedules_start_time TIME NOT NULL,        -- '19:00:00'
  lecture_schedules_end_time TIME,                   -- opsional

  -- Lokasi & keterangan
  lecture_schedules_place TEXT,
  lecture_schedules_notes TEXT,                      -- "Setiap pekan ke-2"

  -- Pengaturan
  lecture_schedules_is_active BOOLEAN DEFAULT TRUE,
  lecture_schedules_is_paid BOOLEAN DEFAULT FALSE,
  lecture_schedules_price INT,
  lecture_schedules_capacity INT,
  lecture_schedules_is_registration_required BOOLEAN DEFAULT FALSE,

  -- Timestamps
  lecture_schedules_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  lecture_schedules_updated_at TIMESTAMP,
  lecture_schedules_deleted_at TIMESTAMP
);


CREATE INDEX IF NOT EXISTS idx_lecture_schedules_lecture_id 
  ON lecture_schedules (lecture_schedules_lecture_id);

CREATE INDEX IF NOT EXISTS idx_lecture_schedules_day_time 
  ON lecture_schedules (lecture_schedules_day_of_week, lecture_schedules_start_time);

CREATE INDEX IF NOT EXISTS idx_lecture_schedules_active 
  ON lecture_schedules (lecture_schedules_is_active);
