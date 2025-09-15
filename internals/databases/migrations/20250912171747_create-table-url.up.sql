BEGIN;

-- ===== Extensions =====
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram untuk search label (opsional)

-- ===== ENUM: owner entity =====
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'url_entity_enum') THEN
    CREATE TYPE url_entity_enum AS ENUM (
      'masjid',
      'announcement',
      'book',
      'post',
      'lecture',
      'other'
    );
  END IF;
END$$;

-- ===== (Opsional) Preset kinds global =====
CREATE TABLE IF NOT EXISTS system_url_kinds (
  system_kind_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  system_kind_entity      url_entity_enum NOT NULL,
  system_kind_code        VARCHAR(64)     NOT NULL,
  system_kind_name        VARCHAR(120)    NOT NULL,
  system_is_singleton     BOOLEAN NOT NULL DEFAULT FALSE,
  system_support_primary  BOOLEAN NOT NULL DEFAULT TRUE,
  system_is_locked        BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (system_kind_entity, system_kind_code)
);

-- ===== (Opsional) Custom kinds per masjid =====
CREATE TABLE IF NOT EXISTS masjid_custom_url_kinds (
  custom_kind_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  custom_kind_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  custom_kind_entity      url_entity_enum NOT NULL,
  custom_kind_code        VARCHAR(64)     NOT NULL,
  custom_kind_name        VARCHAR(120)    NOT NULL,
  custom_is_singleton     BOOLEAN NOT NULL DEFAULT FALSE,
  custom_support_primary  BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE (custom_kind_masjid_id, custom_kind_entity, custom_kind_code)
);

-- ===== (Opsional) Override preset per masjid =====
CREATE TABLE IF NOT EXISTS masjid_kind_settings (
  kind_setting_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind_setting_masjid_id      UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  kind_setting_system_kind_id UUID NOT NULL REFERENCES system_url_kinds(system_kind_id) ON DELETE CASCADE,
  is_enabled                  BOOLEAN NOT NULL DEFAULT TRUE,
  display_name_override       VARCHAR(120),
  UNIQUE (kind_setting_masjid_id, kind_setting_system_kind_id)
);

-- ===== (Opsional) View gabungan preset + custom per masjid =====
CREATE OR REPLACE VIEW v_effective_url_kinds AS
WITH sys_matrix AS (
  SELECT
    m.masjid_id,
    s.system_kind_entity       AS entity,
    s.system_kind_code         AS code,
    s.system_kind_name         AS system_name,
    s.system_is_singleton      AS is_singleton,
    s.system_support_primary   AS support_primary,
    s.system_is_locked         AS is_locked,
    COALESCE(ms.is_enabled, TRUE) AS is_enabled,
    ms.display_name_override
  FROM masjids m
  CROSS JOIN system_url_kinds s
  LEFT JOIN masjid_kind_settings ms
    ON ms.kind_setting_masjid_id = m.masjid_id
   AND ms.kind_setting_system_kind_id = s.system_kind_id
)
SELECT
  masjid_id,
  entity,
  code,
  COALESCE(display_name_override, system_name) AS name,
  is_singleton,
  support_primary,
  TRUE  AS is_enabled,
  is_locked,
  'system'::text AS source
FROM sys_matrix
WHERE is_enabled = TRUE

UNION ALL
SELECT
  c.custom_kind_masjid_id   AS masjid_id,
  c.custom_kind_entity      AS entity,
  c.custom_kind_code        AS code,
  c.custom_kind_name        AS name,
  c.custom_is_singleton     AS is_singleton,
  c.custom_support_primary  AS support_primary,
  TRUE                      AS is_enabled,
  FALSE                     AS is_locked,
  'custom'::text            AS source
FROM masjid_custom_url_kinds c;

-- ===== Tabel utama: URL per-entity (dengan size per objek) =====
CREATE TABLE IF NOT EXISTS tenant_entity_urls (
  url_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & owner
  url_masjid_id            UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  url_entity               url_entity_enum NOT NULL,
  url_entity_id            UUID NOT NULL,             -- id owner (masjid_id/announcement_id/...)

  -- klasifikasi & konten
  url_kind_code            VARCHAR(64) NOT NULL,      -- validasi di service vs v_effective_url_kinds (opsional)
  url_label                VARCHAR(120),
  url_href                 TEXT NOT NULL,

  -- metadata file (tracking size)
  url_mime                 VARCHAR(80),
  url_file_size_bytes      BIGINT,                    -- diisi setelah upload (HEAD OSS)

  -- pengelompokan (album memori, dsb)
  url_group_id             UUID,
  url_group_code           VARCHAR(64),

  -- flags
  url_is_primary           BOOLEAN NOT NULL DEFAULT FALSE,
  url_is_active            BOOLEAN NOT NULL DEFAULT TRUE,

  -- trash lifecycle (opsional)
  url_trash_href           TEXT,
  url_delete_pending_until TIMESTAMPTZ,

  -- audit
  url_created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  url_updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  url_deleted_at           TIMESTAMPTZ,

  -- checks
  CONSTRAINT urls_trash_pair_chk CHECK (
    (url_trash_href IS NULL AND url_delete_pending_until IS NULL)
    OR
    (url_trash_href IS NOT NULL AND url_delete_pending_until IS NOT NULL)
  ),
  CONSTRAINT urls_group_xor_chk CHECK (
    (url_group_id IS NULL OR url_group_code IS NULL)
  ),
  CONSTRAINT chk_url_filesize_nonneg CHECK (
    url_file_size_bytes IS NULL OR url_file_size_bytes >= 0
  )
);

-- ===== Indexes inti =====
CREATE INDEX IF NOT EXISTS ix_urls_owner_alive
  ON tenant_entity_urls (url_masjid_id, url_entity, url_entity_id)
  WHERE url_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_urls_owner_href_ci_alive
  ON tenant_entity_urls (url_masjid_id, url_entity, url_entity_id, LOWER(url_href))
  WHERE url_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_urls_primary_per_kind_alive
  ON tenant_entity_urls (url_masjid_id, url_entity, url_entity_id, url_kind_code)
  WHERE url_is_active = TRUE
    AND url_is_primary = TRUE
    AND url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_urls_owner_kind_alive
  ON tenant_entity_urls (url_masjid_id, url_entity, url_entity_id, url_kind_code, url_created_at DESC)
  WHERE url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_urls_delete_due_alive
  ON tenant_entity_urls (url_delete_pending_until)
  WHERE url_delete_pending_until IS NOT NULL
    AND url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_urls_label_trgm_alive
  ON tenant_entity_urls USING GIN (url_label gin_trgm_ops)
  WHERE url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_urls_kind_size_alive
  ON tenant_entity_urls (url_masjid_id, url_kind_code, url_file_size_bytes)
  WHERE url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_urls_need_size
  ON tenant_entity_urls (url_masjid_id, url_kind_code, url_created_at DESC)
  WHERE url_file_size_bytes IS NULL AND url_deleted_at IS NULL;

-- ===== Tabel cache: primary per kind (read cepat) =====
CREATE TABLE IF NOT EXISTS tenant_primary_urls (
  purl_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purl_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  purl_entity     url_entity_enum NOT NULL,
  purl_entity_id  UUID NOT NULL,
  purl_kind_code  VARCHAR(64) NOT NULL,
  purl_href       TEXT NOT NULL,
  purl_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code)
);

CREATE INDEX IF NOT EXISTS ix_purl_owner_kind
  ON tenant_primary_urls (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code);

-- ===== Backfill cache dari data primary yang sudah ada =====
INSERT INTO tenant_primary_urls (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code, purl_href, purl_updated_at)
SELECT url_masjid_id, url_entity, url_entity_id, url_kind_code, url_href, NOW()
FROM tenant_entity_urls
WHERE url_is_primary = TRUE
  AND url_is_active  = TRUE
  AND url_deleted_at IS NULL
ON CONFLICT (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code)
DO UPDATE SET purl_href = EXCLUDED.purl_href, purl_updated_at = NOW();

-- ===== Trigger: ensure single primary (unset others) =====
CREATE OR REPLACE FUNCTION trg_urls_ensure_single_primary()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF (TG_OP IN ('INSERT','UPDATE'))
     AND NEW.url_is_primary = TRUE
     AND NEW.url_is_active  = TRUE
     AND NEW.url_deleted_at IS NULL
  THEN
    UPDATE tenant_entity_urls
       SET url_is_primary = FALSE,
           url_updated_at = NOW()
     WHERE url_masjid_id = NEW.url_masjid_id
       AND url_entity    = NEW.url_entity
       AND url_entity_id = NEW.url_entity_id
       AND url_kind_code = NEW.url_kind_code
       AND url_id       <> COALESCE(NEW.url_id, '00000000-0000-0000-0000-000000000000'::uuid)
       AND url_deleted_at IS NULL;
  END IF;
  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='tbi_urls_ensure_single_primary') THEN
    CREATE TRIGGER tbi_urls_ensure_single_primary
      BEFORE INSERT OR UPDATE OF url_is_primary, url_is_active, url_deleted_at
      ON tenant_entity_urls
      FOR EACH ROW EXECUTE FUNCTION trg_urls_ensure_single_primary();
  END IF;
END$$;

-- ===== Trigger: sync cache primary (after DML) =====
CREATE OR REPLACE FUNCTION trg_urls_sync_primary_cache()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  v_masjid UUID;
  v_entity url_entity_enum;
  v_owner  UUID;
  v_kind   VARCHAR(64);
  v_href   TEXT;
BEGIN
  IF TG_OP = 'DELETE' THEN
    v_masjid := OLD.url_masjid_id; v_entity := OLD.url_entity; v_owner := OLD.url_entity_id; v_kind := OLD.url_kind_code;
    IF OLD.url_is_primary IS TRUE THEN
      DELETE FROM tenant_primary_urls
       WHERE purl_masjid_id = v_masjid AND purl_entity = v_entity
         AND purl_entity_id = v_owner  AND purl_kind_code = v_kind;
    END IF;
    RETURN NULL;
  END IF;

  v_masjid := NEW.url_masjid_id; v_entity := NEW.url_entity; v_owner := NEW.url_entity_id; v_kind := NEW.url_kind_code; v_href := NEW.url_href;

  IF NEW.url_is_primary = TRUE AND NEW.url_is_active = TRUE AND NEW.url_deleted_at IS NULL THEN
    INSERT INTO tenant_primary_urls (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code, purl_href, purl_updated_at)
    VALUES (v_masjid, v_entity, v_owner, v_kind, v_href, NOW())
    ON CONFLICT (purl_masjid_id, purl_entity, purl_entity_id, purl_kind_code)
    DO UPDATE SET purl_href = EXCLUDED.purl_href, purl_updated_at = NOW();
  ELSE
    DELETE FROM tenant_primary_urls
     WHERE purl_masjid_id = v_masjid AND purl_entity = v_entity
       AND purl_entity_id = v_owner  AND purl_kind_code = v_kind
       AND purl_href = v_href;
  END IF;

  RETURN NEW;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='tau_urls_sync_primary_cache') THEN
    CREATE TRIGGER tau_urls_sync_primary_cache
      AFTER INSERT OR UPDATE OF url_is_primary, url_is_active, url_href, url_deleted_at
      ON tenant_entity_urls
      FOR EACH ROW EXECUTE FUNCTION trg_urls_sync_primary_cache();
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='tad_urls_sync_primary_cache') THEN
    CREATE TRIGGER tad_urls_sync_primary_cache
      AFTER DELETE ON tenant_entity_urls
      FOR EACH ROW EXECUTE FUNCTION trg_urls_sync_primary_cache();
  END IF;
END$$;

-- ===== View: primary masjid (baca cache; join sumber untuk trash/ttl) =====
CREATE OR REPLACE VIEW v_masjid_primary_urls AS
SELECT
  p.purl_masjid_id            AS masjid_id,
  p.purl_kind_code            AS type,
  p.purl_href                 AS file_url,
  u.url_trash_href            AS trash_url,
  u.url_delete_pending_until  AS delete_pending_until,
  COALESCE(u.url_updated_at, p.purl_updated_at) AS updated_at
FROM tenant_primary_urls p
LEFT JOIN tenant_entity_urls u
  ON u.url_masjid_id = p.purl_masjid_id
 AND u.url_entity    = p.purl_entity
 AND u.url_entity_id = p.purl_entity_id
 AND u.url_kind_code = p.purl_kind_code
 AND u.url_href      = p.purl_href
 AND u.url_deleted_at IS NULL
WHERE p.purl_entity = 'masjid';

COMMIT;
