BEGIN;

-- =========================
-- Prasyarat class_rooms
-- =========================
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram untuk GIN trgm

-- =========================================================
-- CLASS_ROOMS (timestamps standar GORM; tanpa trigger)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_rooms (
  class_room_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_room_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas ruang
  class_room_name        TEXT        NOT NULL,
  class_room_code        TEXT,              -- kode opsional
  class_room_slug        VARCHAR(50),       -- slug opsional
  class_room_location    TEXT,
  class_room_capacity    INT CHECK (class_room_capacity >= 0),
  class_room_description TEXT,

  -- karakteristik
  class_room_is_virtual  BOOLEAN NOT NULL DEFAULT FALSE,
  class_room_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- daftar fasilitas (opsional)
  class_room_features    JSONB   NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps standar GORM (isi/update oleh aplikasi)
  class_room_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_deleted_at  TIMESTAMPTZ,

  -- =========================
  -- Validasi ringan
  -- =========================
  -- name tidak boleh hanya spasi
  CONSTRAINT chk_cr_name_not_blank CHECK (length(btrim(coalesce(class_room_name, ''))) > 0),

  -- slug lowercase aman (opsional); boleh NULL
  CONSTRAINT chk_cr_slug_format CHECK (
    class_room_slug IS NULL OR class_room_slug ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  ),

  -- code alfanumerik plus _- .(opsional); boleh spasi? → tidak
  CONSTRAINT chk_cr_code_format CHECK (
    class_room_code IS NULL OR class_room_code ~ '^[A-Za-z0-9._-]+$'
  ),

  -- features harus array
  CONSTRAINT chk_cr_features_is_array CHECK (jsonb_typeof(class_room_features) = 'array')
);

-- =========================================================
-- INDEXES & UNIQUES (soft-delete aware)
-- =========================================================

-- Uniques per tenant (case-insensitive) → hanya baris alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_name_ci_alive
  ON class_rooms (class_room_masjid_id, lower(class_room_name))
  WHERE class_room_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_code_ci_alive
  ON class_rooms (class_room_masjid_id, lower(class_room_code))
  WHERE class_room_deleted_at IS NULL
    AND class_room_code IS NOT NULL
    AND length(btrim(class_room_code)) > 0;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_slug_ci_alive
  ON class_rooms (class_room_masjid_id, lower(class_room_slug))
  WHERE class_room_deleted_at IS NULL
    AND class_room_slug IS NOT NULL;

-- Lookups umum
CREATE INDEX IF NOT EXISTS idx_class_rooms_masjid_alive
  ON class_rooms (class_room_masjid_id)
  WHERE class_room_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_active_alive
  ON class_rooms (class_room_masjid_id, class_room_is_active)
  WHERE class_room_deleted_at IS NULL;

-- Fitur JSONB (query @> / ?/ ?| / path)
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin_alive
  ON class_rooms USING GIN (class_room_features jsonb_path_ops)
  WHERE class_room_deleted_at IS NULL;

-- Pencarian teks bebas (ILIKE) untuk name & location
CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm_alive
  ON class_rooms USING GIN (class_room_name gin_trgm_ops)
  WHERE class_room_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm_alive
  ON class_rooms USING GIN (class_room_location gin_trgm_ops)
  WHERE class_room_deleted_at IS NULL;

-- Arsip waktu (scan besar efisien)
CREATE INDEX IF NOT EXISTS brin_class_rooms_created_at
  ON class_rooms USING BRIN (class_room_created_at);

  

-- =========================================================
-- ENUM: virtual_platform_enum (idempotent via DO block)
-- =========================================================
DO $$
BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'virtual_platform_enum') THEN
CREATE TYPE virtual_platform_enum AS ENUM (
'zoom',
'google_meet',
'microsoft_teams',
'other'
);
END IF;
END$$;

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
class_room_virtual_link_label      TEXT NOT NULL,
class_room_virtual_link_join_url   TEXT NOT NULL,
class_room_virtual_link_host_url   TEXT,
class_room_virtual_link_meeting_id TEXT,
class_room_virtual_link_passcode   TEXT,
class_room_virtual_link_notes      TEXT,

-- platform
class_room_virtual_link_platform virtual_platform_enum NOT NULL,

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
ON class_room_virtual_links (
class_room_virtual_link_room_id,
lower(class_room_virtual_link_label)
)
WHERE class_room_virtual_link_deleted_at IS NULL;

-- Hindari duplikasi link pada tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_vlinks_url_ci
ON class_room_virtual_links (
class_room_virtual_link_masjid_id,
lower(class_room_virtual_link_join_url)
)
WHERE class_room_virtual_link_deleted_at IS NULL;

-- Query umum (by room + active, alive only)
CREATE INDEX IF NOT EXISTS idx_room_vlinks_active
ON class_room_virtual_links (
class_room_virtual_link_room_id,
class_room_virtual_link_is_active
)
WHERE class_room_virtual_link_deleted_at IS NULL;

-- Tambahan: filter cepat per platform (alive only)
CREATE INDEX IF NOT EXISTS idx_room_vlinks_platform_active
ON class_room_virtual_links (
class_room_virtual_link_platform,
class_room_virtual_link_is_active
)
WHERE class_room_virtual_link_deleted_at IS NULL;

-- Tambahan: kombinasi tenant + platform (alive only)
CREATE INDEX IF NOT EXISTS idx_room_vlinks_tenant_platform
ON class_room_virtual_links (
class_room_virtual_link_masjid_id,
class_room_virtual_link_platform
)
WHERE class_room_virtual_link_deleted_at IS NULL;


-- =========================================================
-- TABLE: class_room_urls  (lampiran/link untuk class_rooms)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_room_urls (
  class_room_url_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & owner
  class_room_url_masjid_id       UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_room_url_room_id         UUID NOT NULL
    REFERENCES class_rooms(class_room_id) ON DELETE CASCADE,

  -- jenis aset (selaras dengan post_url / *_urls)
  class_room_url_kind            VARCHAR(24) NOT NULL,   -- 'banner'|'image'|'video'|'attachment'|'link'|'cover'

  -- lokasi file/link
  class_room_url_href            TEXT,                   -- URL publik (boleh NULL jika pakai object storage)
  class_room_url_object_key      TEXT,                   -- object key aktif
  class_room_url_object_key_old  TEXT,                   -- object key lama (retensi in-place replace)

  -- tampilan & urutan
  class_room_url_label           VARCHAR(160),
  class_room_url_order           INT NOT NULL DEFAULT 0,
  class_room_url_is_primary      BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit & soft delete
  class_room_url_created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_url_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_url_deleted_at      TIMESTAMPTZ,
  class_room_url_delete_pending_until TIMESTAMPTZ,

  -- ====== Guards / Checks ======
  -- Minimal salah satu: href atau object_key harus ada (selama belum soft-deleted)
  CONSTRAINT chk_class_room_urls_location_present
    CHECK (
      class_room_url_deleted_at IS NOT NULL
      OR (COALESCE(NULLIF(trim(class_room_url_href), ''), class_room_url_object_key) IS NOT NULL)
    ),

  -- kind terbatas (opsional; comment out kalau ingin fleksibel penuh)
  CONSTRAINT chk_class_room_urls_kind_allowed
    CHECK (class_room_url_kind IN ('banner','image','video','attachment','link','cover'))
);

-- =========================================================
-- INDEXES & CONSTRAINTS
-- =========================================================

-- 1) Satu "primary" per room+kind (mis. 1 cover utama, 1 banner utama, dll) pada baris alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_primary_per_kind
  ON class_room_urls (class_room_url_room_id, class_room_url_kind)
  WHERE class_room_url_is_primary = TRUE AND class_room_url_deleted_at IS NULL;

-- 2) Hindari duplikasi label per room (case-insensitive) pada baris alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_label_ci
  ON class_room_urls (class_room_url_room_id, lower(class_room_url_label))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_label IS NOT NULL;

-- 3) Hindari duplikasi object_key per tenant (alive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_object_key_ci
  ON class_room_urls (class_room_url_masjid_id, lower(class_room_url_object_key))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_object_key IS NOT NULL
    AND length(trim(class_room_url_object_key)) > 0;

-- 4) (Opsional) Hindari duplikasi href per tenant (alive) — berguna untuk kind='link'
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_href_ci
  ON class_room_urls (class_room_url_masjid_id, lower(class_room_url_href))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_href IS NOT NULL
    AND length(trim(class_room_url_href)) > 0;

-- 5) Query umum: per room & kind & alive
CREATE INDEX IF NOT EXISTS idx_room_urls_room_kind_alive
  ON class_room_urls (class_room_url_room_id, class_room_url_kind, class_room_url_order)
  WHERE class_room_url_deleted_at IS NULL;

-- 6) Pencarian label / href (trigram) — cepat untuk search
CREATE INDEX IF NOT EXISTS idx_room_urls_label_trgm
  ON class_room_urls USING GIN (class_room_url_label gin_trgm_ops)
  WHERE class_room_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_room_urls_href_trgm
  ON class_room_urls USING GIN (class_room_url_href gin_trgm_ops)
  WHERE class_room_url_deleted_at IS NULL;

-- 7) Query per tenant
CREATE INDEX IF NOT EXISTS idx_room_urls_tenant_alive
  ON class_room_urls (class_room_url_masjid_id, class_room_url_kind)
  WHERE class_room_url_deleted_at IS NULL;


COMMIT;