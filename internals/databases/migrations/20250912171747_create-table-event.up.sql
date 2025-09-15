BEGIN;

-- ======================================
-- Extensions (aman diulang)
-- ======================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()

-- ======================================
-- CLASS_EVENT_THEMES (per masjid)
-- ======================================
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

-- index bantu listing
CREATE INDEX IF NOT EXISTS idx_class_event_themes_masjid_active
  ON class_event_themes(class_event_themes_masjid_id, class_event_themes_is_active);

-- ======================================
-- CLASS_EVENTS — event ad-hoc/special
-- ======================================
CREATE TABLE IF NOT EXISTS class_events (
  class_events_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_events_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- referensi tema (opsional)
  class_events_theme_id         UUID
    REFERENCES class_event_themes(class_event_themes_id) ON DELETE SET NULL,

  -- target minimal (opsional, salah satu)
  class_events_section_id       UUID,
  class_events_class_id         UUID,
  class_events_class_subject_id UUID,

  -- info inti
  class_events_title            VARCHAR(160) NOT NULL,
  class_events_desc             TEXT,
  class_events_category         VARCHAR(80),

  -- waktu
  class_events_timezone         TEXT,                -- ex: "Asia/Jakarta"
  class_events_date             DATE NOT NULL,       -- start date
  class_events_end_date         DATE,                -- opsional multi-hari
  class_events_start_time       TIME,                -- null utk all-day
  class_events_end_time         TIME,
  class_events_is_all_day       BOOLEAN NOT NULL DEFAULT FALSE,

  -- lokasi / modality
  class_events_modality         VARCHAR(16),         -- 'onsite'|'online'|'hybrid'
  class_events_room_id          UUID,                -- jika onsite
  class_events_meeting_url      TEXT,                -- jika online/hybrid

  -- pengisi
  class_events_teacher_id       UUID,

  -- kapasitas & RSVP
  class_events_capacity         INT,                 -- NULL = tanpa batas
  class_events_enrollment_policy VARCHAR(16),        -- 'open'|'invite'|'closed'

  -- publikasi
  class_events_is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  class_events_publish_at       TIMESTAMPTZ,

  -- media utama
  class_events_banner_url       TEXT,

  -- audit
  class_events_created_by_user_id UUID,
  class_events_updated_by_user_id UUID,

  class_events_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_events_deleted_at       TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_class_events_modality
    CHECK (class_events_modality IS NULL OR class_events_modality IN ('onsite','online','hybrid')),
  CONSTRAINT chk_class_events_enroll_policy
    CHECK (class_events_enrollment_policy IS NULL OR class_events_enrollment_policy IN ('open','invite','closed')),
  CONSTRAINT chk_class_events_capacity_nonneg
    CHECK (class_events_capacity IS NULL OR class_events_capacity >= 0),
  CONSTRAINT chk_class_events_date_range
    CHECK (class_events_end_date IS NULL OR class_events_end_date >= class_events_date)
);

-- index yang sering dipakai
CREATE INDEX IF NOT EXISTS idx_class_events_masjid_date
  ON class_events(class_events_masjid_id, class_events_date);
CREATE INDEX IF NOT EXISTS idx_class_events_publish
  ON class_events(class_events_masjid_id, class_events_is_published, class_events_date);
CREATE INDEX IF NOT EXISTS idx_class_events_theme
  ON class_events(class_events_masjid_id, class_events_theme_id);
CREATE INDEX IF NOT EXISTS idx_class_events_modality
  ON class_events(class_events_masjid_id, class_events_modality);

-- ======================================
-- USER_CLASS_EVENT_ATTENDANCES — RSVP/kehadiran
-- ======================================
CREATE TABLE IF NOT EXISTS user_class_event_attendances (
  user_class_event_attendances_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_event_attendances_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_class_event_attendances_event_id   UUID NOT NULL
    REFERENCES class_events(class_events_id) ON DELETE CASCADE,

  -- identitas peserta (TEPAT SATU)
  user_class_event_attendances_user_id            UUID,
  user_class_event_attendances_masjid_student_id  UUID,
  user_class_event_attendances_guardian_id        UUID,

  -- RSVP & kehadiran
  user_class_event_attendances_rsvp_status        VARCHAR(16),   -- invited|going|maybe|declined|waitlist
  user_class_event_attendances_checked_in_at      TIMESTAMPTZ,
  user_class_event_attendances_no_show            BOOLEAN NOT NULL DEFAULT FALSE,

  -- tiket opsional
  user_class_event_attendances_ticket_code        VARCHAR(64),

  -- catatan
  user_class_event_attendances_note               TEXT,

  -- audit
  user_class_event_attendances_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_event_attendances_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_event_attendances_deleted_at         TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_user_class_event_attendances_identity_one
    CHECK (num_nonnulls(
      user_class_event_attendances_user_id,
      user_class_event_attendances_masjid_student_id,
      user_class_event_attendances_guardian_id
    ) = 1),
  CONSTRAINT chk_user_class_event_attendances_rsvp
    CHECK (user_class_event_attendances_rsvp_status IS NULL OR
           user_class_event_attendances_rsvp_status IN ('invited','going','maybe','declined','waitlist'))
);

-- uniqueness & index
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_event_attendances_unique_identity
ON user_class_event_attendances(
  user_class_event_attendances_event_id,
  COALESCE(user_class_event_attendances_user_id,           '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_event_attendances_masjid_student_id, '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_event_attendances_guardian_id,       '00000000-0000-0000-0000-000000000000'::uuid)
);

CREATE INDEX IF NOT EXISTS idx_user_class_event_attendances_event
  ON user_class_event_attendances(user_class_event_attendances_event_id);
CREATE INDEX IF NOT EXISTS idx_user_class_event_attendances_masjid_rsvp
  ON user_class_event_attendances(user_class_event_attendances_masjid_id, user_class_event_attendances_rsvp_status);
CREATE INDEX IF NOT EXISTS idx_user_class_event_attendances_checkedin
  ON user_class_event_attendances(user_class_event_attendances_event_id, user_class_event_attendances_checked_in_at);

-- ======================================
-- CLASS_EVENT_URLS — lampiran/URL fleksibel (2-slot + retensi)
-- ======================================
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

CREATE INDEX IF NOT EXISTS idx_class_event_urls_event_kind
  ON class_event_urls(class_event_url_event_id, class_event_url_kind);
CREATE INDEX IF NOT EXISTS idx_class_event_urls_primary
  ON class_event_urls(class_event_url_event_id, class_event_url_is_primary);

COMMIT;
