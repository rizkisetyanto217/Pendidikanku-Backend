-- =========================================================
-- EXTENSIONS (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops (ILIKE search)

-- =========================================================
-- TABLE: class_rooms — FINAL (tanpa JSONPath)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_rooms (
  class_room_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_room_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas ruang
  class_room_name        TEXT        NOT NULL,
  class_room_code        TEXT,              -- opsional
  class_room_slug        VARCHAR(50),       -- opsional
  class_room_location    TEXT,
  class_room_capacity    INT CHECK (class_room_capacity >= 0),
  class_room_description TEXT,

  -- karakteristik
  class_room_is_virtual  BOOLEAN NOT NULL DEFAULT FALSE,
  class_room_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- Single image (2-slot + retensi)
  class_room_image_url                   TEXT,
  class_room_image_object_key            TEXT,
  class_room_image_url_old               TEXT,
  class_room_image_object_key_old        TEXT,
  class_room_image_delete_pending_until  TIMESTAMPTZ,

  -- daftar fasilitas (opsional)
  class_room_features    JSONB   NOT NULL DEFAULT '[]'::jsonb,

  -- virtual meeting links (JSONB array; validasi shape di aplikasi)
  class_room_virtual_links JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps standar
  class_room_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_deleted_at  TIMESTAMPTZ,

  -- =========================
  -- VALIDATIONS (sederhana, aman)
  -- =========================

  -- name tidak boleh hanya spasi
  CONSTRAINT chk_cr_name_not_blank
    CHECK (length(btrim(coalesce(class_room_name, ''))) > 0),

  -- slug lowercase aman (opsional)
  CONSTRAINT chk_cr_slug_format
    CHECK (class_room_slug IS NULL OR class_room_slug ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'),

  -- code alfanumerik plus _-. (opsional)
  CONSTRAINT chk_cr_code_format
    CHECK (class_room_code IS NULL OR class_room_code ~ '^[A-Za-z0-9._-]+$'),

  -- features & virtual_links harus array
  CONSTRAINT chk_cr_features_is_array
    CHECK (jsonb_typeof(class_room_features) = 'array'),
  CONSTRAINT chk_cr_vlinks_is_array
    CHECK (jsonb_typeof(class_room_virtual_links) = 'array')
);

-- Pastikan kalau sebelumnya ada constraint shape lama, dibuang aja
ALTER TABLE class_rooms DROP CONSTRAINT IF EXISTS chk_cr_vlinks_shape;

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

-- Fitur JSONB (tanpa JSONPath ops)
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin_alive
  ON class_rooms USING GIN (class_room_features)
  WHERE class_room_deleted_at IS NULL;

-- Virtual links JSONB (tanpa JSONPath ops)
CREATE INDEX IF NOT EXISTS idx_class_rooms_vlinks_gin_alive
  ON class_rooms USING GIN (class_room_virtual_links)
  WHERE class_room_deleted_at IS NULL;

-- Pencarian teks bebas (ILIKE) untuk name & location (trigram)
CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm_alive
  ON class_rooms USING GIN (class_room_name gin_trgm_ops)
  WHERE class_room_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm_alive
  ON class_rooms USING GIN (class_room_location gin_trgm_ops)
  WHERE class_room_deleted_at IS NULL;

-- Arsip waktu (scan besar efisien)
CREATE INDEX IF NOT EXISTS brin_class_rooms_created_at
  ON class_rooms USING BRIN (class_room_created_at);

CREATE INDEX IF NOT EXISTS brin_class_rooms_updated_at
  ON class_rooms USING BRIN (class_room_updated_at);

-- Buang index expression berbasis JSONPath kalau sudah pernah dibuat
DROP INDEX IF EXISTS idx_class_rooms_first_active_platform;
DROP INDEX IF EXISTS idx_class_rooms_has_active_vlink;

-- =========================================================
-- COMMENTS (dokumentasi skema)
-- =========================================================
COMMENT ON TABLE class_rooms IS 'Ruang kelas (fisik/virtual). Link meeting virtual disimpan sebagai JSONB array di class_room_virtual_links.';
COMMENT ON COLUMN class_rooms.class_room_features IS 'Array JSONB berisi fitur ruang (mis. ["ac","projector","whiteboard"]).';
COMMENT ON COLUMN class_rooms.class_room_virtual_links IS 'Array JSONB link virtual; validasi struktur dilakukan di layer aplikasi.';

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

  -- jenis aset
  class_room_url_kind            VARCHAR(24) NOT NULL,   -- 'banner'|'image'|'video'|'attachment'|'link'|'cover'

  -- lokasi file/link
  class_room_url_href            TEXT,
  class_room_url_object_key      TEXT,
  class_room_url_object_key_old  TEXT,

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
  CONSTRAINT chk_class_room_urls_location_present
    CHECK (
      class_room_url_deleted_at IS NOT NULL
      OR (COALESCE(NULLIF(trim(class_room_url_href), ''), class_room_url_object_key) IS NOT NULL)
    ),

  CONSTRAINT chk_class_room_urls_kind_allowed
    CHECK (class_room_url_kind IN ('banner','image','video','attachment','link','cover'))
);

-- =========================================================
-- INDEXES & CONSTRAINTS untuk class_room_urls
-- =========================================================

CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_primary_per_kind
  ON class_room_urls (class_room_url_room_id, class_room_url_kind)
  WHERE class_room_url_is_primary = TRUE AND class_room_url_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_label_ci
  ON class_room_urls (class_room_url_room_id, lower(class_room_url_label))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_label IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_object_key_ci
  ON class_room_urls (class_room_url_masjid_id, lower(class_room_url_object_key))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_object_key IS NOT NULL
    AND length(trim(class_room_url_object_key)) > 0;

CREATE UNIQUE INDEX IF NOT EXISTS uq_room_urls_href_ci
  ON class_room_urls (class_room_url_masjid_id, lower(class_room_url_href))
  WHERE class_room_url_deleted_at IS NULL
    AND class_room_url_href IS NOT NULL
    AND length(trim(class_room_url_href)) > 0;

CREATE INDEX IF NOT EXISTS idx_room_urls_room_kind_alive
  ON class_room_urls (class_room_url_room_id, class_room_url_kind, class_room_url_order)
  WHERE class_room_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_room_urls_label_trgm
  ON class_room_urls USING GIN (class_room_url_label gin_trgm_ops)
  WHERE class_room_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_room_urls_href_trgm
  ON class_room_urls USING GIN (class_room_url_href gin_trgm_ops)
  WHERE class_room_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_room_urls_tenant_alive
  ON class_room_urls (class_room_url_masjid_id, class_room_url_kind)
  WHERE class_room_url_deleted_at IS NULL;
