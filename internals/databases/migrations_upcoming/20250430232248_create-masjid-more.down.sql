BEGIN;

-- =========================
-- DROP T3) masjid_tag_relations
-- =========================
DROP INDEX IF EXISTS idx_tag_relations_tag_created_at_desc;
DROP INDEX IF EXISTS idx_tag_relations_masjid_created_at_desc;
DROP INDEX IF EXISTS idx_tag_relations_tag_id;
DROP INDEX IF EXISTS idx_tag_relations_masjid_id;

DROP TABLE IF EXISTS masjid_tag_relations CASCADE;

-- =========================
-- DROP T2) masjid_tags
-- =========================
DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_tags ON masjid_tags;
DROP FUNCTION IF EXISTS set_updated_at_masjid_tags;

DROP INDEX IF EXISTS idx_masjid_tags_created_at_desc;
DROP INDEX IF EXISTS gin_masjid_tags_name_trgm;
DROP INDEX IF EXISTS ux_masjid_tags_name_lower;

DROP TABLE IF EXISTS masjid_tags CASCADE;

-- =========================
-- DROP T1) masjid_profile_teacher_dkm
-- =========================
DROP TRIGGER IF EXISTS trg_set_updated_at_profile_teacher_dkm ON masjid_profile_teacher_dkm;
DROP FUNCTION IF EXISTS set_updated_at_profile_teacher_dkm;

DROP INDEX IF EXISTS ux_profile_teacher_dkm_masjid_user_role_alive;
DROP INDEX IF EXISTS gin_profile_teacher_dkm_name_trgm;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_masjid_created_at_desc;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_masjid_role_created_at_desc;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_user_id;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_masjid_id;

DROP TABLE IF EXISTS masjid_profile_teacher_dkm CASCADE;

-- =========================
-- EXTENSIONS (opsional, biarkan kalau dipakai tabel lain)
-- =========================
-- DROP EXTENSION IF EXISTS pg_trgm;

COMMIT;
