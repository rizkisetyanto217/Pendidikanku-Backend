BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram (GIN trgm)

-- =========================================================
-- ENUM
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'virtual_provider_enum') THEN
    CREATE TYPE virtual_provider_enum AS ENUM (
      'zoom',
      'google_meet',
      'microsoft_teams',
      'webex',
      'custom',
      'rtmp'
    );
  END IF;
END$$;

-- =========================================================
-- TABLE: class_rooms
-- =========================================================
CREATE TABLE IF NOT EXISTS class_rooms (
  class_room_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant
  class_rooms_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas
  class_rooms_name TEXT NOT NULL,
  class_rooms_code TEXT,
  class_rooms_slug TEXT,
  class_rooms_description TEXT,

  -- lokasi fisik ringkas (abaikan jika virtual)
  class_rooms_location TEXT,
  class_rooms_capacity INT CHECK (class_rooms_capacity >= 0),

  -- status/visibilitas
  class_rooms_is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
  class_rooms_is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  class_rooms_is_public  BOOLEAN NOT NULL DEFAULT FALSE,

  -- fleksibel
  class_rooms_features JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps standar GORM
  class_rooms_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_rooms_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_rooms_deleted_at TIMESTAMPTZ
);

-- =======================
-- INDEXES & OPTIMIZATION
-- =======================

-- Unik per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_name_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_name))
  WHERE class_rooms_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_code_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_code))
  WHERE class_rooms_deleted_at IS NULL
    AND class_rooms_code IS NOT NULL AND length(trim(class_rooms_code)) > 0;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_slug_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_slug))
  WHERE class_rooms_deleted_at IS NULL
    AND class_rooms_slug IS NOT NULL AND length(trim(class_rooms_slug)) > 0;

-- Query umum
CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_active
  ON class_rooms (class_rooms_masjid_id, class_rooms_is_active)
  WHERE class_rooms_deleted_at IS NULL;

-- Pencarian cepat
CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm
  ON class_rooms USING GIN (class_rooms_name gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm
  ON class_rooms USING GIN (class_rooms_location gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

-- Filter fitur (JSONB)
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin
  ON class_rooms USING GIN (class_rooms_features jsonb_path_ops)
  WHERE class_rooms_deleted_at IS NULL;

-- =========================================================
-- TABLE: class_room_virtual_links
-- =========================================================
CREATE TABLE IF NOT EXISTS class_room_virtual_links (
  class_room_virtual_link_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope
  class_room_virtual_link_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_room_virtual_link_room_id UUID NOT NULL
    REFERENCES class_rooms(class_room_id) ON DELETE CASCADE,

  -- identitas link
  class_room_virtual_link_label    TEXT NOT NULL,
  class_room_virtual_link_provider virtual_provider_enum NOT NULL DEFAULT 'custom',
  class_room_virtual_link_join_url TEXT NOT NULL,
  class_room_virtual_link_host_url TEXT,
  class_room_virtual_link_meeting_id TEXT,
  class_room_virtual_link_passcode   TEXT,
  class_room_virtual_link_notes      TEXT,

  -- status
  class_room_virtual_link_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- timestamps
  class_room_virtual_link_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_virtual_link_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_virtual_link_deleted_at TIMESTAMPTZ
);

-- =======================
-- INDEXES & OPTIMIZATION
-- =======================

-- Unik per room: label (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_vlinks_label_ci
  ON class_room_virtual_links (class_room_virtual_link_room_id, lower(class_room_virtual_link_label))
  WHERE class_room_virtual_link_deleted_at IS NULL;

-- Hindari duplikasi link pada tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_vlinks_url_ci
  ON class_room_virtual_links (class_room_virtual_link_masjid_id, lower(class_room_virtual_link_join_url))
  WHERE class_room_virtual_link_deleted_at IS NULL;

-- Query umum
CREATE INDEX IF NOT EXISTS idx_room_vlinks_active
  ON class_room_virtual_links (class_room_virtual_link_room_id, class_room_virtual_link_is_active)
  WHERE class_room_virtual_link_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_room_vlinks_provider
  ON class_room_virtual_links (class_room_virtual_link_room_id, class_room_virtual_link_provider)
  WHERE class_room_virtual_link_deleted_at IS NULL;

COMMIT;