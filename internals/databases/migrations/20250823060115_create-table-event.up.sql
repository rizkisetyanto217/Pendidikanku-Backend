-- Extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()

/* =========================================================
   1) CLASS_EVENT_THEMES (per masjid)
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_event_themes (
  class_event_theme_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_theme_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas tema
  class_event_theme_code        VARCHAR(64)  NOT NULL,
  class_event_theme_name        VARCHAR(120) NOT NULL,

  -- warna: pilih salah satu dari preset atau custom hex
  class_event_theme_color        VARCHAR(32),   -- ex: "red", "blue", "green"
  class_event_theme_custom_color VARCHAR(16),   -- ex: "#FFAA33"

  class_event_theme_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_event_theme_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_theme_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_theme_deleted_at  TIMESTAMPTZ,

  CONSTRAINT uq_class_event_themes_masjid_code
    UNIQUE (class_event_theme_masjid_id, class_event_theme_code)
);

-- Index bantu listing
CREATE INDEX IF NOT EXISTS idx_class_event_themes_masjid_active
  ON class_event_themes (class_event_theme_masjid_id, class_event_theme_is_active);

CREATE INDEX IF NOT EXISTS idx_class_event_themes_masjid_name
  ON class_event_themes (class_event_theme_masjid_id, class_event_theme_name);


/* =========================================================
   2) ENUM delivery mode (online/offline/hybrid)
   ========================================================= */
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('online','offline','hybrid');
  END IF;
END$$;

/* =========================================================
   3) TABLE: class_events — event ad-hoc/special
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_events (
  class_event_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- referensi tema (opsional)
  class_event_theme_id         UUID
    REFERENCES class_event_themes(class_event_theme_id) ON DELETE SET NULL,

  -- link ke pola jadwal (opsional)
  class_event_schedule_id      UUID
    REFERENCES class_schedules(class_schedule_id) ON DELETE CASCADE,

  -- target minimal (opsional, salah satu)
  class_event_section_id       UUID,
  class_event_class_id         UUID,
  class_event_class_subject_id UUID,

  -- info inti
  class_event_title            VARCHAR(160) NOT NULL,
  class_event_desc             TEXT,

  -- waktu
  class_event_date             DATE NOT NULL,   -- start date
  class_event_end_date         DATE,            -- opsional multi-hari
  class_event_start_time       TIME,            -- NULL = all-day
  class_event_end_time         TIME,

  -- lokasi / delivery mode
  class_event_delivery_mode    class_delivery_mode_enum,
  class_event_room_id          UUID REFERENCES class_rooms(class_room_id),

  -- pengajar (internal / tamu)
  class_event_teacher_id       UUID,
  class_event_teacher_name     TEXT,
  class_event_teacher_desc     TEXT,

  -- lokasi file/link
  class_event_image_url               TEXT,
  class_event_image_object_key          TEXT,
  class_event_image_url_old               TEXT,
  class_event_image_object_key_old      TEXT,
  class_event_image_delete_pending_until TIMESTAMPTZ,

  -- kapasitas & RSVP
  class_event_capacity         INT,
  class_event_enrollment_policy VARCHAR(16),     -- 'open'|'invite'|'closed'

  -- status aktif
  class_event_is_active        BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_event_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_deleted_at       TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_class_event_enroll_policy
    CHECK (class_event_enrollment_policy IS NULL OR class_event_enrollment_policy IN ('open','invite','closed')),
  CONSTRAINT chk_class_event_capacity_nonneg
    CHECK (class_event_capacity IS NULL OR class_event_capacity >= 0),
  CONSTRAINT chk_class_event_date_range
    CHECK (class_event_end_date IS NULL OR class_event_end_date >= class_event_date)
);

-- Indexes class_events
CREATE INDEX IF NOT EXISTS idx_class_events_masjid_date
  ON class_events (class_event_masjid_id, class_event_date);

CREATE INDEX IF NOT EXISTS idx_class_events_active
  ON class_events (class_event_masjid_id, class_event_is_active, class_event_date);

CREATE INDEX IF NOT EXISTS idx_class_events_theme
  ON class_events (class_event_masjid_id, class_event_theme_id);

CREATE INDEX IF NOT EXISTS idx_class_events_delivery_mode
  ON class_events (class_event_masjid_id, class_event_delivery_mode);

CREATE INDEX IF NOT EXISTS idx_class_events_date_range
  ON class_events (class_event_masjid_id, class_event_date, class_event_end_date);

CREATE INDEX IF NOT EXISTS idx_class_events_room
  ON class_events (class_event_masjid_id, class_event_room_id);

CREATE INDEX IF NOT EXISTS idx_class_events_teacher
  ON class_events (class_event_masjid_id, class_event_teacher_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_events_id_tenant
  ON class_events (class_event_id, class_event_masjid_id);


/* =========================================================
   4) CLASS_EVENT_URLS — lampiran/URL fleksibel
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_event_urls (
  class_event_url_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_url_masjid_id            UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_event_url_event_id             UUID NOT NULL
    REFERENCES class_events(class_event_id) ON DELETE CASCADE,

  -- klasifikasi & label
  class_event_url_kind                 VARCHAR(32) NOT NULL, -- 'image'|'file'|'video'|'audio'|'link'|'banner'|'doc'...
  class_event_url_label                VARCHAR(160),

  -- storage (2-slot + retensi)
  class_event_url                  TEXT,        -- aktif
  class_event_url_object_key           TEXT,
  class_event_url_old              TEXT,        -- kandidat delete
  class_event_url_object_key_old       TEXT,
  class_event_url_delete_pending_until TIMESTAMPTZ, -- jadwal hard delete old

  -- flag
  class_event_url_is_primary           BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  class_event_url_created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_url_updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_url_deleted_at           TIMESTAMPTZ,

  CONSTRAINT chk_class_event_url_kind_nonempty
    CHECK (length(coalesce(class_event_url_kind,'')) > 0)
);

-- Indexes class_event_urls
CREATE INDEX IF NOT EXISTS idx_class_event_urls_event_kind
  ON class_event_urls (class_event_url_event_id, class_event_url_kind);

CREATE INDEX IF NOT EXISTS idx_class_event_urls_primary
  ON class_event_urls (class_event_url_event_id, class_event_url_is_primary);

CREATE INDEX IF NOT EXISTS idx_class_event_urls_masjid
  ON class_event_urls (class_event_url_masjid_id);

-- Unik satu primary per (event, kind) yang hidup
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_event_urls_primary_per_kind_alive
  ON class_event_urls (class_event_url_event_id, class_event_url_kind)
  WHERE class_event_url_deleted_at IS NULL
    AND class_event_url_is_primary = TRUE;

-- Kandidat purge (in-place replace & soft-deleted)
CREATE INDEX IF NOT EXISTS ix_class_event_urls_purge_due
  ON class_event_urls (class_event_url_delete_pending_until)
  WHERE class_event_url_delete_pending_until IS NOT NULL
    AND (
      (class_event_url_deleted_at IS NULL  AND class_event_url_object_key_old IS NOT NULL) OR
      (class_event_url_deleted_at IS NOT NULL AND class_event_url_object_key     IS NOT NULL)
    );