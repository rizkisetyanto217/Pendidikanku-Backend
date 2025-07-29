CREATE TABLE IF NOT EXISTS donations (
    donation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    donation_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    donation_name VARCHAR(50) NOT NULL, 
    donation_amount INTEGER NOT NULL CHECK (donation_amount > 0), 

    -- 🔹 Rincian pembagian donasi
    donation_amount_masjid INTEGER CHECK (donation_amount_masjid >= 0),
    donation_amount_masjidku INTEGER CHECK (donation_amount_masjidku >= 0),
    donation_amount_masjidku_to_masjid INTEGER CHECK (donation_amount_masjidku_to_masjid >= 0),
    donation_amount_masjidku_to_app INTEGER CHECK (donation_amount_masjidku_to_app >= 0),

    donation_message TEXT, 

    donation_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (
        donation_status IN ('pending', 'paid', 'expired', 'canceled', 'completed')
    ), 

    donation_order_id VARCHAR(100) NOT NULL UNIQUE CHECK (
        char_length(donation_order_id) <= 100
    ), 

    donation_target_type INT CHECK (donation_target_type IN (1, 2, 3, 4)) DEFAULT NULL,
    donation_target_id UUID DEFAULT NULL,

    donation_payment_token TEXT, 
    donation_payment_gateway VARCHAR(50) DEFAULT 'midtrans',
    donation_payment_method VARCHAR,

    donation_paid_at TIMESTAMP, 

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,                       
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    donation_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL
);


-- Index opsional untuk pencarian berdasarkan tipe target
CREATE INDEX IF NOT EXISTS idx_donations_target_type
  ON donations (donation_target_type);

-- Index opsional untuk pencarian berdasarkan id target
CREATE INDEX IF NOT EXISTS idx_donations_target_id
  ON donations (donation_target_id);


-- 🔍 Index untuk pencarian cepat order_id (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_donations_order_id_lower 
    ON donations (LOWER(donation_order_id));

-- 🔍 Index umum untuk pencarian berdasarkan user
CREATE INDEX IF NOT EXISTS idx_donations_user_id 
    ON donations (donation_user_id);

-- 🔍 Index untuk pencarian berdasarkan masjid_id
CREATE INDEX IF NOT EXISTS idx_donations_masjid_id
    ON donations (donation_masjid_id);


CREATE TABLE IF NOT EXISTS donation_likes (
  donation_like_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  donation_like_is_liked BOOLEAN DEFAULT TRUE,
  donation_like_donation_id UUID NOT NULL REFERENCES donations(donation_id) ON DELETE CASCADE,
  donation_like_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  donation_like_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  donation_like_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  CONSTRAINT unique_donation_like UNIQUE (donation_like_donation_id, donation_like_user_id)
);

CREATE INDEX IF NOT EXISTS idx_donation_likes_donation_id 
  ON donation_likes(donation_like_donation_id);

CREATE INDEX IF NOT EXISTS idx_donation_likes_user_id 
  ON donation_likes(donation_like_user_id);

CREATE INDEX IF NOT EXISTS idx_donation_likes_updated_at 
  ON donation_likes(donation_like_updated_at);
