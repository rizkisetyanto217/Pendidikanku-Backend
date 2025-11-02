-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =====================================================================
-- OPTIONAL HELPER INDEXES (aman jika tabel tujuan memang ada)
-- (TIDAK DIUBAH)
-- =====================================================================
DO $do$
BEGIN
  -- class_sections
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'class_sections') THEN
    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_class_section_id_tenant
             ON class_sections (class_section_id, class_section_school_id)';

    IF EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_name = 'class_sections' AND column_name = 'class_sections_deleted_at'
    ) THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS ix_class_sections_tenant_alive
               ON class_sections (class_section_school_id, class_section_id)
               WHERE class_sections_deleted_at IS NULL';
    END IF;

    IF EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_name = 'class_sections' AND column_name = 'class_section_is_active'
    ) THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS ix_class_sections_tenant_active
               ON class_sections (class_section_school_id, class_section_id)
               WHERE class_section_is_active = TRUE';
    END IF;
  END IF;

  -- school_teachers
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'school_teachers') THEN
    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_school_teachers_id_tenant
             ON school_teachers (school_teacher_id, school_teacher_school_id)';

    IF EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_name = 'school_teachers' AND column_name = 'school_teacher_deleted_at'
    ) THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS ix_school_teachers_alive
               ON school_teachers (school_teacher_school_id, school_teacher_id)
               WHERE school_teacher_deleted_at IS NULL';
    END IF;
  END IF;
END
$do$;

-- =====================================================================
-- TABLE: post_themes  (pluralized)
-- =====================================================================
CREATE TABLE IF NOT EXISTS post_themes (
  post_theme_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_theme_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  post_theme_kind      VARCHAR(24) NOT NULL
    CHECK (post_theme_kind IN ('announcement','material','post','other')),

  post_theme_parent_id UUID NULL REFERENCES post_themes(post_theme_id) ON DELETE SET NULL,

  post_theme_name      VARCHAR(80)  NOT NULL,
  post_theme_slug      VARCHAR(120) NOT NULL,

  post_theme_color        VARCHAR(20),
  post_theme_custom_color VARCHAR(20),
  post_theme_description  TEXT,
  post_theme_is_active    BOOLEAN NOT NULL DEFAULT TRUE,

  post_theme_icon_url                  TEXT,
  post_theme_icon_object_key           TEXT,
  post_theme_icon_url_old              TEXT,
  post_theme_icon_object_key_old       TEXT,
  post_theme_icon_delete_pending_until TIMESTAMPTZ,

  post_theme_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_theme_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_theme_deleted_at TIMESTAMPTZ
);

-- Indexing post_themes (disesuaikan ON ke tabel baru)
CREATE UNIQUE INDEX IF NOT EXISTS uq_post_theme_name_per_tenant_kind_alive
  ON post_themes (post_theme_school_id, post_theme_kind, lower(post_theme_name))
  WHERE post_theme_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_post_theme_slug_per_tenant_kind_alive
  ON post_themes (post_theme_school_id, post_theme_kind, lower(post_theme_slug))
  WHERE post_theme_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_post_theme_tenant_kind_active_alive
  ON post_themes (post_theme_school_id, post_theme_kind, post_theme_is_active)
  WHERE post_theme_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_post_theme_parent_alive
  ON post_themes (post_theme_parent_id)
  WHERE post_theme_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_post_theme_name_trgm_alive
  ON post_themes USING GIN (post_theme_name gin_trgm_ops)
  WHERE post_theme_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_post_theme_icon_purge_due
  ON post_themes (post_theme_icon_delete_pending_until)
  WHERE post_theme_icon_object_key_old IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_post_theme_id_tenant
  ON post_themes (post_theme_id, post_theme_school_id);



-- =====================================================================
-- TABLE: posts  (pluralized)
-- =====================================================================
CREATE TABLE IF NOT EXISTS posts (
  post_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,

  post_kind VARCHAR(24) NOT NULL
    CHECK (post_kind IN ('announcement','material','post','other')),

  is_dkm_sender BOOLEAN NOT NULL DEFAULT FALSE,
  post_created_by_teacher_id UUID NULL,

  post_class_section_id UUID NULL,

  post_theme_id UUID NULL,

  post_slug    VARCHAR(160),
  post_title   VARCHAR(200) NOT NULL,
  post_date    DATE NOT NULL,
  post_content TEXT NOT NULL,

  post_excerpt TEXT,
  post_meta    JSONB,

  post_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  post_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_deleted_at TIMESTAMPTZ,

  post_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(post_title,   '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(post_content, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(post_excerpt, '')), 'C')
  ) STORED,

  post_section_ids UUID[] NOT NULL DEFAULT '{}',

  post_is_published  BOOLEAN NOT NULL DEFAULT FALSE,
  post_published_at  TIMESTAMPTZ,
  post_audience_snapshot JSONB,

  CONSTRAINT fk_post_created_by_teacher_same_tenant
    FOREIGN KEY (post_created_by_teacher_id, post_school_id)
    REFERENCES school_teachers (school_teacher_id, school_teacher_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_post_section_same_tenant
    FOREIGN KEY (post_class_section_id, post_school_id)
    REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_post_theme_same_tenant
    FOREIGN KEY (post_theme_id, post_school_id)
    REFERENCES post_themes (post_theme_id, post_theme_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- Indexing posts
CREATE UNIQUE INDEX IF NOT EXISTS uq_post_id_tenant
  ON posts (post_id, post_school_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_post_slug_per_tenant_alive
  ON posts (post_school_id, lower(post_slug))
  WHERE post_deleted_at IS NULL AND post_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_post_tenant_kind_date_live
  ON posts (post_school_id, post_kind, post_date DESC)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_post_theme_live
  ON posts (post_theme_id)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_post_created_by_teacher_live
  ON posts (post_created_at DESC, post_created_by_teacher_id)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_post_search_gin_live
  ON posts USING GIN (post_search)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE;

CREATE INDEX IF NOT EXISTS ix_post_title_trgm_live
  ON posts USING GIN (post_title gin_trgm_ops)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE;

CREATE INDEX IF NOT EXISTS brin_post_created_at
  ON posts USING BRIN (post_created_at);

CREATE INDEX IF NOT EXISTS ix_post_section_ids_gin_live
  ON posts USING GIN (post_section_ids)
  WHERE post_deleted_at IS NULL AND post_is_active = TRUE AND post_kind = 'announcement';



-- =====================================================================
-- TABLE: post_urls  (pluralized)
-- =====================================================================
CREATE TABLE IF NOT EXISTS post_urls (
  post_url_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_url_school_id  UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
  post_url_post_id    UUID NOT NULL REFERENCES posts(post_id) ON DELETE CASCADE,

  post_url_kind       VARCHAR(24) NOT NULL,
  -- lokasi file/link
  post_url               TEXT,
  post_url_object_key          TEXT,
  post_url_old               TEXT,
  post_url_object_key_old      TEXT,
  post_url_delete_pending_until TIMESTAMPTZ,

  post_url_label      VARCHAR(160),
  post_url_order      INT NOT NULL DEFAULT 0,
  post_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  post_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  post_url_deleted_at TIMESTAMPTZ
);

-- Indexing post_urls
CREATE INDEX IF NOT EXISTS ix_post_url_by_owner_live
  ON post_urls (post_url_post_id, post_url_kind, post_url_is_primary DESC, post_url_order, post_url_created_at)
  WHERE post_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_post_url_by_school_live
  ON post_urls (post_url_school_id)
  WHERE post_url_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_post_url_primary_per_kind_alive
  ON post_urls (post_url_post_id, post_url_kind)
  WHERE post_url_deleted_at IS NULL
    AND post_url_is_primary = TRUE;

CREATE INDEX IF NOT EXISTS ix_post_url_purge_due
  ON post_urls (post_url_delete_pending_until)
  WHERE post_url_delete_pending_until IS NOT NULL
    AND (
      (post_url_deleted_at IS NULL  AND post_url_object_key_old IS NOT NULL) OR
      (post_url_deleted_at IS NOT NULL AND post_url_object_key     IS NOT NULL)
    );

CREATE INDEX IF NOT EXISTS gin_post_url_label_trgm_live
  ON post_urls USING GIN (post_url_label gin_trgm_ops)
  WHERE post_url_deleted_at IS NULL;

-- =====================================================================
-- TRIGGER: pastikan theme.kind == post.kind (disesuaikan ON ke 'posts')
-- =====================================================================
CREATE OR REPLACE FUNCTION trg_post_theme_kind_match()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE v_theme_kind TEXT;
BEGIN
  IF NEW.post_theme_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT pt.post_theme_kind
  INTO v_theme_kind
  FROM post_themes pt
  WHERE pt.post_theme_id = NEW.post_theme_id
    AND pt.post_theme_deleted_at IS NULL;

  IF v_theme_kind IS NULL THEN
    RAISE EXCEPTION 'Tema tidak ditemukan atau non-aktif';
  END IF;

  IF v_theme_kind <> NEW.post_kind THEN
    RAISE EXCEPTION 'Theme kind (%) tidak cocok dengan post kind (%)', v_theme_kind, NEW.post_kind;
  END IF;

  RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS tg_post_theme_kind_match ON posts;
CREATE TRIGGER tg_post_theme_kind_match
BEFORE INSERT OR UPDATE OF post_theme_id, post_kind
ON posts
FOR EACH ROW
EXECUTE FUNCTION trg_post_theme_kind_match();

-- Guard tenant (ON posts, refer ke post_themes)
CREATE OR REPLACE FUNCTION trg_post_theme_tenant_guard()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE v_theme_school UUID;
BEGIN
  IF NEW.post_theme_id IS NULL THEN RETURN NEW; END IF;

  SELECT post_theme_school_id INTO v_theme_school
  FROM post_themes
  WHERE post_theme_id = NEW.post_theme_id
    AND post_theme_deleted_at IS NULL;

  IF v_theme_school IS NULL OR v_theme_school <> NEW.post_school_id THEN
    RAISE EXCEPTION 'post_theme belongs to different school';
  END IF;

  RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS tg_post_theme_tenant_guard ON posts;
CREATE TRIGGER tg_post_theme_tenant_guard
BEFORE INSERT OR UPDATE OF post_theme_id, post_school_id
ON posts
FOR EACH ROW
EXECUTE FUNCTION trg_post_theme_tenant_guard();

-- =====================================================================
-- FUNCTIONS: RESOLVER & PUBLISH (update referensi ke posts/post_themes)
-- =====================================================================

CREATE OR REPLACE FUNCTION resolve_teacher_sections_by_csst(
  p_school_id UUID,
  p_teacher_id UUID,
  p_class_subjects_id UUID DEFAULT NULL
) RETURNS UUID[]
LANGUAGE sql
AS $$
  SELECT COALESCE(ARRAY(
    SELECT DISTINCT csst.class_section_subject_teacher_section_id
    FROM class_section_subject_teachers csst
    WHERE csst.class_section_subject_teacher_school_id = p_school_id
      AND csst.class_section_subject_teacher_teacher_id = p_teacher_id
      AND csst.class_section_subject_teacher_is_active = TRUE
      AND csst.class_section_subject_teacher_deleted_at IS NULL
      AND (p_class_subjects_id IS NULL
           OR csst.class_section_subject_teacher_class_subject_id = p_class_subjects_id)
  ), '{}');
$$;

CREATE OR REPLACE FUNCTION upsert_post_meta_subjects(
  p_post_id UUID,
  p_subject_ids UUID[]
) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE v_meta JSONB;
BEGIN
  IF p_subject_ids IS NULL OR array_length(p_subject_ids,1) IS NULL THEN
    RETURN;
  END IF;

  SELECT post_meta INTO v_meta FROM posts WHERE post_id = p_post_id;
  v_meta := COALESCE(v_meta, '{}'::jsonb);

  v_meta := v_meta || jsonb_build_object(
    'class_subjects_ids',
    (
      SELECT to_jsonb(ARRAY(
        SELECT DISTINCT sid FROM UNNEST(p_subject_ids) AS sid
      ))
    )
  );

  UPDATE posts
  SET post_meta = v_meta,
      post_updated_at = NOW()
  WHERE post_id = p_post_id;
END $$;

CREATE OR REPLACE FUNCTION publish_post_with_sections(
  p_post_id UUID,
  p_section_ids UUID[],
  p_fill_inbox BOOLEAN DEFAULT TRUE
) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE v_kind TEXT;
BEGIN
  SELECT post_kind INTO v_kind FROM posts WHERE post_id = p_post_id AND post_deleted_at IS NULL;
  IF v_kind IS DISTINCT FROM 'announcement' THEN
    RAISE EXCEPTION 'publish_post_with_sections hanya untuk post_kind=announcement';
  END IF;

  UPDATE posts
  SET post_section_ids = COALESCE(p_section_ids, '{}'),
      post_is_published = TRUE,
      post_published_at = NOW(),
      post_audience_snapshot = jsonb_build_object(
        'sections', COALESCE(array_length(COALESCE(p_section_ids,'{}'::uuid[]), 1), 0)
      ),
      post_updated_at = NOW()
  WHERE post_id = p_post_id
    AND post_deleted_at IS NULL
    AND post_is_active = TRUE;
END $$;

CREATE OR REPLACE FUNCTION publish_post_for_all_sections(
  p_post_id UUID,
  p_school_id UUID,
  p_fill_inbox BOOLEAN DEFAULT TRUE
) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE v_kind TEXT; v_section_ids UUID[]; v_sql TEXT;
BEGIN
  SELECT post_kind INTO v_kind FROM posts WHERE post_id = p_post_id AND post_deleted_at IS NULL;
  IF v_kind IS DISTINCT FROM 'announcement' THEN
    RAISE EXCEPTION 'publish_post_for_all_sections hanya untuk post_kind=announcement';
  END IF;

  v_sql := '
    SELECT ARRAY(
      SELECT cs.class_section_id
      FROM class_sections cs
      WHERE cs.class_section_school_id = $1
  ';
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_sections' AND column_name='class_section_deleted_at') THEN
    v_sql := v_sql || ' AND cs.class_sections_deleted_at IS NULL ';
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_sections' AND column_name='class_section_is_active') THEN
    v_sql := v_sql || ' AND cs.class_section_is_active = TRUE ';
  END IF;

  v_sql := v_sql || ' ) ';

  EXECUTE v_sql USING p_school_id INTO v_section_ids;
  v_section_ids := COALESCE(v_section_ids, '{}');

  PERFORM publish_post_with_sections(p_post_id, v_section_ids, TRUE);
END $$;

CREATE OR REPLACE FUNCTION publish_post_via_csst(
  p_post_id UUID,
  p_school_id UUID,
  p_teacher_id UUID,
  p_class_subjects_id UUID DEFAULT NULL,
  p_fill_inbox BOOLEAN DEFAULT TRUE
) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE v_kind TEXT; v_section_ids UUID[]; v_subject_ids UUID[];
BEGIN
  SELECT post_kind INTO v_kind FROM posts WHERE post_id = p_post_id AND post_deleted_at IS NULL;
  IF v_kind IS DISTINCT FROM 'announcement' THEN
    RAISE EXCEPTION 'publish_post_via_csst hanya untuk post_kind=announcement';
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='class_section_subject_teachers') THEN
    RAISE EXCEPTION 'Tabel class_section_subject_teachers tidak tersedia';
  END IF;

  v_section_ids := resolve_teacher_sections_by_csst(p_school_id, p_teacher_id, p_class_subjects_id);
  PERFORM publish_post_with_sections(p_post_id, v_section_ids, TRUE);

  IF p_class_subjects_id IS NOT NULL THEN
    v_subject_ids := ARRAY[p_class_subjects_id]::uuid[];
  ELSE
    SELECT COALESCE(ARRAY(
      SELECT DISTINCT csst.class_section_subject_teacher_class_subject_id
      FROM class_section_subject_teachers csst
      WHERE csst.class_section_subject_teacher_school_id = p_school_id
        AND csst.class_section_subject_teacher_teacher_id = p_teacher_id
        AND csst.class_section_subject_teacher_is_active = TRUE
        AND csst.class_section_subject_teacher_deleted_at IS NULL
    ), '{}') INTO v_subject_ids;
  END IF;

  PERFORM upsert_post_meta_subjects(p_post_id, v_subject_ids);
END $$;

CREATE OR REPLACE FUNCTION publish_post_by_csst_ids(
  p_post_id UUID,
  p_csst_ids UUID[],
  p_fill_inbox BOOLEAN DEFAULT TRUE
) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE v_kind TEXT; v_section_ids UUID[];
BEGIN
  SELECT post_kind INTO v_kind FROM posts WHERE post_id = p_post_id AND post_deleted_at IS NULL;
  IF v_kind IS DISTINCT FROM 'announcement' THEN
    RAISE EXCEPTION 'publish_post_by_csst_ids hanya untuk post_kind=announcement';
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='class_section_subject_teachers') THEN
    RAISE EXCEPTION 'Tabel class_section_subject_teachers tidak tersedia';
  END IF;

  IF p_csst_ids IS NULL OR array_length(p_csst_ids, 1) IS NULL THEN
    PERFORM publish_post_with_sections(p_post_id, '{}', TRUE);
    RETURN;
  END IF;

  SELECT ARRAY(
    SELECT DISTINCT csst.class_section_subject_teacher_section_id
    FROM class_section_subject_teachers csst
    WHERE csst.class_section_subject_teacher_id = ANY(p_csst_ids)
      AND csst.class_section_subject_teacher_deleted_at IS NULL
      AND csst.class_section_subject_teacher_is_active = TRUE
  ) INTO v_section_ids;

  PERFORM publish_post_with_sections(p_post_id, COALESCE(v_section_ids, '{}'), TRUE);
END $$;

-- =====================================================================
-- (REFERENCE) FEED QUERY (TIDAK DIUBAH selain nama tabel)
-- =====================================================================
-- SELECT p.*
-- FROM posts p
-- WHERE p.post_deleted_at IS NULL
--   AND p.post_is_active = TRUE
--   AND p.post_is_published = TRUE
--   AND p.post_kind = 'announcement'
--   AND p.post_section_ids && ARRAY[$section_id_1, $section_id_2