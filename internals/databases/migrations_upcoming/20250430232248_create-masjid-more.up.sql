BEGIN;

-- ====== EXTENSIONS (sekali saja) ======
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- T1) masjid_profile_teacher_dkm  → TABLE
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_profile_teacher_dkm (
    masjid_profile_teacher_dkm_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_profile_teacher_dkm_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_profile_teacher_dkm_user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    masjid_profile_teacher_dkm_name        VARCHAR(100) NOT NULL,
    masjid_profile_teacher_dkm_role        VARCHAR(100) NOT NULL,
    masjid_profile_teacher_dkm_description TEXT,
    masjid_profile_teacher_dkm_message     TEXT,
    masjid_profile_teacher_dkm_image_url   TEXT,

    -- timestamps
    masjid_profile_teacher_dkm_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    masjid_profile_teacher_dkm_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    masjid_profile_teacher_dkm_deleted_at  TIMESTAMPTZ NULL
);

-- ---- Indexing & Optimize (T1) ----
CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_masjid_id
  ON masjid_profile_teacher_dkm (masjid_profile_teacher_dkm_masjid_id);

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_user_id
  ON masjid_profile_teacher_dkm (masjid_profile_teacher_dkm_user_id);

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_masjid_role_created_at_desc
  ON masjid_profile_teacher_dkm (
    masjid_profile_teacher_dkm_masjid_id,
    masjid_profile_teacher_dkm_role,
    masjid_profile_teacher_dkm_created_at DESC
  );

CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_masjid_created_at_desc
  ON masjid_profile_teacher_dkm (
    masjid_profile_teacher_dkm_masjid_id,
    masjid_profile_teacher_dkm_created_at DESC
  );

CREATE INDEX IF NOT EXISTS gin_profile_teacher_dkm_name_trgm
  ON masjid_profile_teacher_dkm
  USING gin (lower(masjid_profile_teacher_dkm_name) gin_trgm_ops);

CREATE UNIQUE INDEX IF NOT EXISTS ux_profile_teacher_dkm_masjid_user_role_alive
  ON masjid_profile_teacher_dkm (
    masjid_profile_teacher_dkm_masjid_id,
    masjid_profile_teacher_dkm_user_id,
    masjid_profile_teacher_dkm_role
  )
  WHERE masjid_profile_teacher_dkm_user_id IS NOT NULL
    AND masjid_profile_teacher_dkm_deleted_at IS NULL;


-- =========================================================
-- T2) masjid_tags  → TABLE
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_tags (
    masjid_tag_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_tag_name        VARCHAR(50) NOT NULL,
    masjid_tag_description TEXT,

    -- timestamps
    masjid_tag_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    masjid_tag_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    masjid_tag_deleted_at  TIMESTAMPTZ NULL
);

-- ---- Indexing & Optimize (T2) ----
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjid_tags_name_lower
  ON masjid_tags (lower(masjid_tag_name));

CREATE INDEX IF NOT EXISTS gin_masjid_tags_name_trgm
  ON masjid_tags USING gin (lower(masjid_tag_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_masjid_tags_created_at_desc
  ON masjid_tags (masjid_tag_created_at DESC);

DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_tags ON masjid_tags;
CREATE TRIGGER trg_set_updated_at_masjid_tags
BEFORE UPDATE ON masjid_tags
FOR EACH ROW EXECUTE FUNCTION set_updated_at_masjid_tags();


-- =========================================================
-- T3) masjid_tag_relations  → TABLE dulu
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_tag_relations (
    masjid_tag_relation_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_tag_relation_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_tag_relation_tag_id    UUID NOT NULL REFERENCES masjid_tags(masjid_tag_id) ON DELETE CASCADE,
    masjid_tag_relation_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (masjid_tag_relation_masjid_id, masjid_tag_relation_tag_id)
);

-- ---- Indexing & Optimize (T3) ----
-- FK lookups dua arah
CREATE INDEX IF NOT EXISTS idx_tag_relations_masjid_id
  ON masjid_tag_relations (masjid_tag_relation_masjid_id);

CREATE INDEX IF NOT EXISTS idx_tag_relations_tag_id
  ON masjid_tag_relations (masjid_tag_relation_tag_id);

-- List tag per masjid (terbaru)
CREATE INDEX IF NOT EXISTS idx_tag_relations_masjid_created_at_desc
  ON masjid_tag_relations (masjid_tag_relation_masjid_id, masjid_tag_relation_created_at DESC);

-- List masjid per tag (terbaru)
CREATE INDEX IF NOT EXISTS idx_tag_relations_tag_created_at_desc
  ON masjid_tag_relations (masjid_tag_relation_tag_id, masjid_tag_relation_created_at DESC);

COMMIT;
