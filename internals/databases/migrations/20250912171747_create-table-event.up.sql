BEGIN;

-- ======================================
-- Extensions (aman diulang)
-- ======================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()

/* =========================================================
   1) CLASS_EVENT_THEMES (per masjid)
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_event_themes (
  class_event_themes_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_themes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- identitas tema
  class_event_themes_code      VARCHAR(64)  NOT NULL,   -- unik per masjid (slug/kode pendek)
  class_event_themes_name      VARCHAR(120) NOT NULL,   -- nama tampilan
  class_event_themes_color_hex VARCHAR(16),             -- opsional (mis. "#1E88E5" atau token)
  class_event_themes_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_event_themes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_themes_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_themes_deleted_at TIMESTAMPTZ,

  CONSTRAINT uq_class_event_themes_masjid_code
    UNIQUE (class_event_themes_masjid_id, class_event_themes_code)
);

-- Index bantu listing (filter per masjid + active)
CREATE INDEX IF NOT EXISTS idx_class_event_themes_masjid_active
  ON class_event_themes(class_event_themes_masjid_id, class_event_themes_is_active);

-- (Opsional) Index untuk pencarian nama tema per masjid
CREATE INDEX IF NOT EXISTS idx_class_event_themes_masjid_name
  ON class_event_themes(class_event_themes_masjid_id, class_event_themes_name);



-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Enum delivery mode
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('online','offline','hybrid');
  END IF;
END$$;

-- =========================================================
-- TABLE: class_events — event ad-hoc/special
-- =========================================================
CREATE TABLE IF NOT EXISTS class_events (
  class_events_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_events_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- referensi tema (opsional)
  class_events_theme_id         UUID
    REFERENCES class_event_themes(class_event_themes_id) ON DELETE SET NULL,

  -- link ke pola jadwal (opsional)
  class_events_schedule_id      UUID
    REFERENCES class_schedules(class_schedule_id) ON DELETE CASCADE,

  -- target minimal (opsional, salah satu)
  class_events_section_id       UUID,
  class_events_class_id         UUID,
  class_events_class_subject_id UUID,

  -- info inti
  class_events_title            VARCHAR(160) NOT NULL,
  class_events_desc             TEXT,

  -- waktu
  class_events_date             DATE NOT NULL,   -- start date
  class_events_end_date         DATE,            -- opsional multi-hari
  class_events_start_time       TIME,            -- NULL = all-day
  class_events_end_time         TIME,

  -- lokasi / delivery mode
  class_events_delivery_mode    class_delivery_mode_enum,  -- online|offline|hybrid
  class_events_room_id          UUID REFERENCES class_rooms(class_room_id),                      -- jika offline/hybrid
  -- (aktifkan FK ke class_rooms jika perlu)
  -- REFERENCES class_rooms(class_room_id) ON DELETE SET NULL,

  -- pengajar (internal / tamu)
  class_events_teacher_id       UUID,   -- referensi guru internal (opsional)
  class_events_teacher_name     TEXT,   -- pengajar tamu / override nama
  class_events_teacher_desc     TEXT,   -- deskripsi singkat pengajar

  -- kapasitas & RSVP
  class_events_capacity         INT,                 -- NULL = tanpa batas
  class_events_enrollment_policy VARCHAR(16),        -- 'open'|'invite'|'closed'

  -- status aktif
  class_events_is_active        BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_events_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_deleted_at       TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_class_events_enroll_policy
    CHECK (class_events_enrollment_policy IS NULL OR class_events_enrollment_policy IN ('open','invite','closed')),
  CONSTRAINT chk_class_events_capacity_nonneg
    CHECK (class_events_capacity IS NULL OR class_events_capacity >= 0),
  CONSTRAINT chk_class_events_date_range
    CHECK (class_events_end_date IS NULL OR class_events_end_date >= class_events_date)
);

-- =========================================================
-- Indexes
-- =========================================================

-- Tenant + tanggal (list/kalender)
CREATE INDEX IF NOT EXISTS idx_class_events_masjid_date
  ON class_events (class_events_masjid_id, class_events_date);

-- Status aktif + urut tanggal
CREATE INDEX IF NOT EXISTS idx_class_events_active
  ON class_events (class_events_masjid_id, class_events_is_active, class_events_date);

-- Tema & delivery mode (filtering)
CREATE INDEX IF NOT EXISTS idx_class_events_theme
  ON class_events (class_events_masjid_id, class_events_theme_id);

CREATE INDEX IF NOT EXISTS idx_class_events_delivery_mode
  ON class_events (class_events_masjid_id, class_events_delivery_mode);

-- Rentang tanggal (opsional kalau sering query range)
CREATE INDEX IF NOT EXISTS idx_class_events_date_range
  ON class_events (class_events_masjid_id, class_events_date, class_events_end_date);

-- Room (kalender ruangan)
CREATE INDEX IF NOT EXISTS idx_class_events_room
  ON class_events (class_events_masjid_id, class_events_room_id);

-- Guru internal (filter per guru)
CREATE INDEX IF NOT EXISTS idx_class_events_teacher
  ON class_events (class_events_masjid_id, class_events_teacher_id);

-- (Opsional, berguna untuk FK tenant-safe dari tabel lain)
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_events_id_tenant
  ON class_events (class_events_id, class_events_masjid_id);


/* =========================================================
   5) CLASS_EVENT_URLS — lampiran/URL fleksibel (2-slot + retensi)
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_event_urls (
  class_event_url_id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_event_url_masjid_id              UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_event_url_event_id               UUID NOT NULL
    REFERENCES class_events(class_events_id) ON DELETE CASCADE,

  -- klasifikasi & label
  class_event_url_kind                   VARCHAR(32) NOT NULL, -- 'image'|'file'|'video'|'audio'|'link'|'banner'|'doc'...
  class_event_url_label                  VARCHAR(160),

  -- storage (2-slot + retensi)
  class_event_url_url                    TEXT,        -- aktif
  class_event_url_object_key             TEXT,
  class_event_url_url_old                TEXT,        -- kandidat delete
  class_event_url_object_key_old         TEXT,
  class_event_url_delete_pending_until   TIMESTAMPTZ, -- jadwal hard delete old

  -- flag
  class_event_url_is_primary             BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  class_event_url_created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_url_updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_event_url_deleted_at             TIMESTAMPTZ,

  -- guard
  CONSTRAINT chk_class_event_url_kind_nonempty
    CHECK (length(coalesce(class_event_url_kind,'')) > 0)
);

-- Indeks untuk akses cepat per event/kind/primary
CREATE INDEX IF NOT EXISTS idx_class_event_urls_event_kind
  ON class_event_urls(class_event_url_event_id, class_event_url_kind);
CREATE INDEX IF NOT EXISTS idx_class_event_urls_primary
  ON class_event_urls(class_event_url_event_id, class_event_url_is_primary);
-- (Opsional) ambil semua URL per masjid
CREATE INDEX IF NOT EXISTS idx_class_event_urls_masjid
  ON class_event_urls(class_event_url_masjid_id);

