BEGIN;

-- =========================
-- DROP T3) school_tag_relations
-- =========================
DROP INDEX IF EXISTS idx_tag_relations_tag_created_at_desc;
DROP INDEX IF EXISTS idx_tag_relations_school_created_at_desc;
DROP INDEX IF EXISTS idx_tag_relations_tag_id;
DROP INDEX IF EXISTS idx_tag_relations_school_id;

DROP TABLE IF EXISTS school_tag_relations CASCADE;

-- =========================
-- DROP T2) school_tags
-- =========================
DROP TRIGGER IF EXISTS trg_set_updated_at_school_tags ON school_tags;
DROP FUNCTION IF EXISTS set_updated_at_school_tags;

DROP INDEX IF EXISTS idx_school_tags_created_at_desc;
DROP INDEX IF EXISTS gin_school_tags_name_trgm;
DROP INDEX IF EXISTS ux_school_tags_name_lower;

DROP TABLE IF EXISTS school_tags CASCADE;

-- =========================
-- DROP T1) school_profile_teacher_dkm
-- =========================
DROP TRIGGER IF EXISTS trg_set_updated_at_profile_teacher_dkm ON school_profile_teacher_dkm;
DROP FUNCTION IF EXISTS set_updated_at_profile_teacher_dkm;

DROP INDEX IF EXISTS ux_profile_teacher_dkm_school_user_role_alive;
DROP INDEX IF EXISTS gin_profile_teacher_dkm_name_trgm;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_school_created_at_desc;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_school_role_created_at_desc;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_user_id;
DROP INDEX IF EXISTS idx_profile_teacher_dkm_school_id;

DROP TABLE IF EXISTS school_profile_teacher_dkm CASCADE;

-- =========================
-- EXTENSIONS (opsional, biarkan kalau dipakai tabel lain)
-- =========================
-- DROP EXTENSION IF EXISTS pg_trgm;

COMMIT;
