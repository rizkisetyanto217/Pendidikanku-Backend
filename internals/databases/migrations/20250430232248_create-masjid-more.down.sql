-- 1) Drop index anak (opsional; DROP TABLE juga akan menghapus index)
DROP INDEX IF EXISTS idx_tag_relations_tag_id;
DROP INDEX IF EXISTS idx_tag_relations_masjid_id;

-- 2) Drop child dulu
DROP TABLE IF EXISTS masjid_tag_relations;

-- 3) Tabel lain yang independen
DROP INDEX IF EXISTS idx_profile_teacher_dkm_masjid_id;
DROP TABLE IF EXISTS masjid_profile_teacher_dkm;

-- 4) Terakhir parent
DROP TABLE IF EXISTS masjid_tags;
