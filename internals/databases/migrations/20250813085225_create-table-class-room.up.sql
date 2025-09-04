BEGIN;

-- =========================
-- Prasyarat khusus class_rooms
-- =========================
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram untuk GIN trgm

-- =========================================================
-- CLASS_ROOMS (timestamps standar GORM; tanpa trigger)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_rooms (
  class_room_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant / scope
  class_rooms_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas ruang
  class_rooms_name      TEXT NOT NULL,
  class_rooms_code      TEXT,
  class_rooms_location  TEXT,
  class_rooms_floor     INT,
  class_rooms_capacity  INT CHECK (class_rooms_capacity >= 0),

  -- karakteristik
  class_rooms_is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
  class_rooms_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- daftar fasilitas (opsional)
  class_rooms_features  JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps standar GORM (isi/update oleh aplikasi)
  class_rooms_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_rooms_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_rooms_deleted_at TIMESTAMPTZ
);

-- Uniques per tenant (case-insensitive) → hanya baris alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_name_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_name))
  WHERE class_rooms_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_rooms_tenant_code_ci
  ON class_rooms (class_rooms_masjid_id, lower(class_rooms_code))
  WHERE class_rooms_deleted_at IS NULL
    AND class_rooms_code IS NOT NULL
    AND length(trim(class_rooms_code)) > 0;

-- Indeks bantu
CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_active
  ON class_rooms (class_rooms_masjid_id, class_rooms_is_active)
  WHERE class_rooms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin
  ON class_rooms USING GIN (class_rooms_features jsonb_path_ops)
  WHERE class_rooms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_name_trgm
  ON class_rooms USING GIN (class_rooms_name gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_location_trgm
  ON class_rooms USING GIN (class_rooms_location gin_trgm_ops)
  WHERE class_rooms_deleted_at IS NULL;

COMMIT;
