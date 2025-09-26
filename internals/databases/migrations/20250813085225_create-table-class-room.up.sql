BEGIN;

-- =========================================================
-- EXTENSIONS (aman diulang)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram ops (ILIKE search)

-- =========================================================
-- TABLE: class_rooms — FINAL
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

  -- virtual meeting links (JSONB array; bukan tabel terpisah)
  class_room_virtual_links JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- timestamps standar GORM (isi/update oleh aplikasi)
  class_room_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_room_deleted_at  TIMESTAMPTZ,

  -- =========================
  -- VALIDATIONS
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

  -- features harus array
  CONSTRAINT chk_cr_features_is_array
    CHECK (jsonb_typeof(class_room_features) = 'array'),

  -- virtual_links harus array
  CONSTRAINT chk_cr_vlinks_is_array
    CHECK (jsonb_typeof(class_room_virtual_links) = 'array')
);

-- =========================================================
-- VALIDASI SHAPE ITEM JSON (jsonpath)
--  - platform: 'zoom' | 'google_meet' | 'microsoft_teams' | 'other'
--  - join_url: string non-kosong
--  - is_active: boolean (opsional)
--  - label/host_url/meeting_id/passcode/notes: string jika ada
--  - tags: array of strings jika ada
--  - time_window: object {from: string, to: string} jika ada
--  - schedule: array of {weekday,start,end,timezone} jika ada
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_cr_vlinks_shape'
  ) THEN
    ALTER TABLE class_rooms
    ADD CONSTRAINT chk_cr_vlinks_shape CHECK (
      NOT EXISTS (
        SELECT 1
        FROM jsonb_path_query_array(class_room_virtual_links, '$[*]') AS el
        WHERE NOT (
          -- platform wajib & valid
          (el ? 'platform')
          AND (el->>'platform') IN ('zoom','google_meet','microsoft_teams','other')
          -- join_url wajib non-kosong
          AND (el ? 'join_url') AND length(btrim(el->>'join_url')) > 0
          -- is_active opsional tapi jika ada harus boolean
          AND (NOT (el ? 'is_active') OR jsonb_typeof(el->'is_active') = 'boolean')
          -- string fields opsional jika ada
          AND (NOT (el ? 'label')       OR jsonb_typeof(el->'label')       = 'string')
          AND (NOT (el ? 'host_url')    OR jsonb_typeof(el->'host_url')    = 'string')
          AND (NOT (el ? 'meeting_id')  OR jsonb_typeof(el->'meeting_id')  = 'string')
          AND (NOT (el ? 'passcode')    OR jsonb_typeof(el->'passcode')    = 'string')
          AND (NOT (el ? 'notes')       OR jsonb_typeof(el->'notes')       = 'string')
          -- tags: array of strings
          AND (
            NOT (el ? 'tags')
            OR (
              jsonb_typeof(el->'tags') = 'array'
              AND NOT EXISTS (
                SELECT 1
                FROM jsonb_path_query_array(el->'tags', '$[*]') t
                WHERE jsonb_typeof(t) <> 'string'
              )
            )
          )
          -- time_window: object {from,to} string
          AND (
            NOT (el ? 'time_window')
            OR (
              jsonb_typeof(el->'time_window') = 'object'
              AND ((el->'time_window') ? 'from')
              AND ((el->'time_window') ? 'to')
              AND jsonb_typeof((el->'time_window')->'from') = 'string'
              AND jsonb_typeof((el->'time_window')->'to')   = 'string'
            )
          )
          -- schedule: array of objects {weekday,start,end,timezone}
          AND (
            NOT (el ? 'schedule')
            OR (
              jsonb_typeof(el->'schedule') = 'array'
              AND NOT EXISTS (
                SELECT 1
                FROM jsonb_path_query_array(el->'schedule', '$[*]') s
                WHERE NOT (
                  jsonb_typeof(s) = 'object'
                  AND (s ? 'weekday') AND (s->>'weekday') IN ('MON','TUE','WED','THU','FRI','SAT','SUN')
                  AND (s ? 'start') AND jsonb_typeof(s->'start') = 'string'
                  AND (s ? 'end')   AND jsonb_typeof(s->'end')   = 'string'
                  AND (s ? 'timezone') AND jsonb_typeof(s->'timezone') = 'string'
                )
              )
            )
          )
        )
      )
    );
  END IF;
END$$;

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

-- Fitur JSONB (query @> / ? / ?| / path) — features
CREATE INDEX IF NOT EXISTS idx_class_rooms_features_gin_alive
  ON class_rooms USING GIN (class_room_features jsonb_path_ops)
  WHERE class_room_deleted_at IS NULL;

-- Virtual links JSONB
CREATE INDEX IF NOT EXISTS idx_class_rooms_vlinks_gin_alive
  ON class_rooms USING GIN (class_room_virtual_links jsonb_path_ops)
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

-- (Opsional) Index expression: platform aktif pertama → filter cepat by platform
CREATE INDEX IF NOT EXISTS idx_class_rooms_first_active_platform
  ON class_rooms (
    (lower(COALESCE((jsonb_path_query_first(class_room_virtual_links, '$[*] ? (@.is_active == true)')->>'platform'), '')))
  )
  WHERE class_room_deleted_at IS NULL;

-- (Opsional) Index expression: punya link aktif? (boolean-ish)
CREATE INDEX IF NOT EXISTS idx_class_rooms_has_active_vlink
  ON class_rooms (
    (jsonb_path_exists(class_room_virtual_links, '$[*] ? (@.is_active == true)'))
  )
  WHERE class_room_deleted_at IS NULL;

-- =========================================================
-- COMMENTS (dokumentasi skema)
-- =========================================================
COMMENT ON TABLE class_rooms IS 'Ruang kelas (fisik/virtual). Link meeting virtual disimpan sebagai JSONB array di class_room_virtual_links.';
COMMENT ON COLUMN class_rooms.class_room_features IS 'Array JSONB berisi fitur ruang (mis. ["ac","projector","whiteboard"]).';
COMMENT ON COLUMN class_rooms.class_room_virtual_links IS 'Array JSONB link virtual; tiap item: {label?, platform, join_url, host_url?, meeting_id?, passcode?, notes?, is_active?, tags?, time_window?, schedule?}.';

-- =========================================================
-- CATATAN:
-- - Unik label/join_url DI DALAM array JSONB tidak dapat dijaga oleh UNIQUE index bawaan. Validasi di layer aplikasi atau pakai trigger (jika suatu saat diizinkan).
-- - Index “first_active_platform” hanya mengindeks platform dari elemen aktif pertama (sesuai jsonpath). Sesuaikan jika pola akses Anda berbeda.
-- - Tanpa trigger untuk updated_at: biarkan GORM yang mengisi.
-- =========================================================



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