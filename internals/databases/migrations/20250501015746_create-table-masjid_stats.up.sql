CREATE TABLE IF NOT EXISTS masjid_stats (
    masjid_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_stats_total_lectures INT DEFAULT 0,         -- jumlah kajian
    masjid_stats_total_sessions INT DEFAULT 0,         -- jumlah pertemuan dari semua kajian
    masjid_stats_total_participants INT DEFAULT 0,     -- total kehadiran (optional: bisa akumulatif)
    masjid_stats_total_donations BIGINT DEFAULT 0,     -- total nominal donasi (dalam satuan rupiah)
    masjid_stats_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_stats_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(masjid_stats_masjid_id)
);

-- Index untuk pencarian berdasarkan masjid_id
CREATE INDEX IF NOT EXISTS idx_masjid_stats_masjid_id ON masjid_stats(masjid_stats_masjid_id);
