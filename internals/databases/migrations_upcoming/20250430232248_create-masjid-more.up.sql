BEGIN;

-- ====== EXTENSIONS (sekali saja) ======
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- T1) school_profile_teacher_dkm  → TABLE
-- =========================================================
CREATE TABLE IF NOT EXISTS school_profile_teacher_dkm (
    school_profile_teacher_dkm_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_profile_teacher_dkm_school_id   UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
    school_profile_teacher_dkm_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    school_profile_teacher_dkm_name        VARCHAR(100) NOT NULL,
    school_profile_teacher_dkm_role        VARCHAR(100) NOT NULL,
    school_profile_teacher_dkm_description TEXT,
    school_profile_teacher_dkm_message     TEXT,
    school_profile_teacher_dkm_image_url   TEXT,

    -- timestamps
    school_profile_teacher_dkm_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    school_profile_teacher_dkm_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    school_profile_teacher_dkm_deleted_at  TIMESTAMPTZ NULL
);

-- ---- Indexing & Optimize (T1) ----
CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_school_id
  ON school_profile_teacher_dkm (school_profile_teacher_dkm_school_id);

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_user_id
  ON school_profile_teacher_dkm (school_profile_teacher_dkm_user_id);

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_school_role_created_at_desc
  ON school_profile_teacher_dkm (
    school_profile_teacher_dkm_school_id,
    school_profile_teacher_dkm_role,
    school_profile_teacher_dkm_created_at DESC
  );

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_school_created_at_desc
  ON school_profile_teacher_dkm (
    school_profile_teacher_dkm_school_id,
    school_profile_teacher_dkm_created_at DESC
  );

CREATE INDEX IF NOT EXISTS gin_profile_teacher_dkm_name_trgm
  ON school_profile_teacher_dkm
  USING gin (lower(school_profile_teacher_dkm_name) gin_trgm_ops);

CREATE UNIQUE INDEX IF NOT EXISTS ux_profile_teacher_dkm_school_user_role_alive
  ON school_profile_teacher_dkm (
    school_profile_teacher_dkm_school_id,
    school_profile_teacher_dkm_user_id,
    school_profile_teacher_dkm_role
  )
  WHERE school_profile_teacher_dkm_user_id IS NOT NULL
    AND school_profile_teacher_dkm_deleted_at IS NULL;


-- =========================================================
-- T2) school_tags  → TABLE
-- =========================================================
CREATE TABLE IF NOT EXISTS school_tags (
    school_tag_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_tag_name        VARCHAR(50) NOT NULL,
    school_tag_description TEXT,

    -- timestamps
    school_tag_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    school_tag_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    school_tag_deleted_at  TIMESTAMPTZ NULL
);

-- ---- Indexing & Optimize (T2) ----
CREATE UNIQUE INDEX IF NOT EXISTS ux_school_tags_name_lower
  ON school_tags (lower(school_tag_name));

CREATE INDEX IF NOT EXISTS gin_school_tags_name_trgm
  ON school_tags USING gin (lower(school_tag_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_school_tags_created_at_desc
  ON school_tags (school_tag_created_at DESC);

DROP TRIGGER IF EXISTS trg_set_updated_at_school_tags ON school_tags;
CREATE TRIGGER trg_set_updated_at_school_tags
BEFORE UPDATE ON school_tags
FOR EACH ROW EXECUTE FUNCTION set_updated_at_school_tags();


-- =========================================================
-- T3) school_tag_relations  → TABLE dulu
-- =========================================================
CREATE TABLE IF NOT EXISTS school_tag_relations (
    school_tag_relation_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_tag_relation_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
    school_tag_relation_tag_id    UUID NOT NULL REFERENCES school_tags(school_tag_id) ON DELETE CASCADE,
    school_tag_relation_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_tag_relation_school_id, school_tag_relation_tag_id)
);

-- ---- Indexing & Optimize (T3) ----
-- FK lookups dua arah
CREATE INDEX IF NOT EXISTS idx_tag_relations_school_id
  ON school_tag_relations (school_tag_relation_school_id);

CREATE INDEX IF NOT EXISTS idx_tag_relations_tag_id
  ON school_tag_relations (school_tag_relation_tag_id);

-- List tag per school (terbaru)
CREATE INDEX IF NOT EXISTS idx_tag_relations_school_created_at_desc
  ON school_tag_relations (school_tag_relation_school_id, school_tag_relation_created_at DESC);

-- List school per tag (terbaru)
CREATE INDEX IF NOT EXISTS idx_tag_relations_tag_created_at_desc
  ON school_tag_relations (school_tag_relation_tag_id, school_tag_relation_created_at DESC);

COMMIT;
