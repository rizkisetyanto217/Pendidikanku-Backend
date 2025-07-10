-- 1. Drop relasi user-masjid follow (punya FK ke masjids)
DROP TABLE IF EXISTS user_follow_masjid CASCADE;

-- 2. Drop masjids_profiles (punya FK ke masjids)
DROP TABLE IF EXISTS masjids_profiles CASCADE;

-- 3. Drop masjids (induk utama)
DROP TABLE IF EXISTS masjids CASCADE;
