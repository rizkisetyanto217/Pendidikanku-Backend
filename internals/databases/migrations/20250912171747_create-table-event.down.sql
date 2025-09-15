BEGIN;

-- =============== DROP VIEWS ===============
DROP VIEW IF EXISTS v_masjid_primary_urls;
DROP VIEW IF EXISTS v_effective_url_kinds;

-- =============== DROP TRIGGERS (on tenant_entity_urls) ===============
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tbi_urls_ensure_single_primary') THEN
    DROP TRIGGER tbi_urls_ensure_single_primary ON tenant_entity_urls;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tau_urls_sync_primary_cache') THEN
    DROP TRIGGER tau_urls_sync_primary_cache ON tenant_entity_urls;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tad_urls_sync_primary_cache') THEN
    DROP TRIGGER tad_urls_sync_primary_cache ON tenant_entity_urls;
  END IF;
END$$;

-- =============== DROP FUNCTIONS ===============
DROP FUNCTION IF EXISTS trg_urls_ensure_single_primary() CASCADE;
DROP FUNCTION IF EXISTS trg_urls_sync_primary_cache()   CASCADE;

-- =============== DROP TABLES (reverse dependency) ===============
-- cache terlebih dahulu (tidak direferensi FK)
DROP TABLE IF EXISTS tenant_primary_urls;

-- tabel utama URL
DROP TABLE IF EXISTS tenant_entity_urls;

-- override preset per masjid (referensi ke system_url_kinds)
DROP TABLE IF EXISTS masjid_kind_settings;

-- custom kinds per masjid
DROP TABLE IF EXISTS masjid_custom_url_kinds;

-- preset kinds global
DROP TABLE IF EXISTS system_url_kinds;

-- =============== DROP TYPES ===============
DROP TYPE IF EXISTS url_entity_enum;

COMMIT;
