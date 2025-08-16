-- =========================
-- DOWN (tanpa CASCADE)
-- =========================

-- Matikan trigger (opsional tapi lebih jelas)
DROP TRIGGER IF EXISTS trg_donations_sync_user_spp_paid ON donations;
DROP TRIGGER IF EXISTS trg_donations_set_timestamp        ON donations;
DROP TRIGGER IF EXISTS trg_user_spp_billings_set_timestamp ON user_spp_billings;
DROP TRIGGER IF EXISTS trg_spp_billings_set_timestamp      ON spp_billings;

-- Child dulu
DROP TABLE IF EXISTS donation_likes;

-- donations + fungsi2-nya
DROP TABLE IF EXISTS donations;
DROP FUNCTION IF EXISTS donations_sync_user_spp_paid();
DROP FUNCTION IF EXISTS set_donations_updated_at();

-- user_spp_billings + fungsi trigger
DROP TABLE IF EXISTS user_spp_billings;
DROP FUNCTION IF EXISTS set_user_spp_billing_updated_at();

-- spp_billings + fungsi trigger
DROP TABLE IF EXISTS spp_billings;
DROP FUNCTION IF EXISTS set_spp_billing_updated_at();

-- Catatan:
-- - Extension pgcrypto memang TIDAK dijatuhkan.
-- - Index/constraint akan ikut hilang saat tabel di-drop.
