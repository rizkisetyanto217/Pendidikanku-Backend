
/* =========================================================
   3) CLASS_ATTENDANCE_EVENTS — window/metode absensi per event
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_attendance_events (
  class_attendance_events_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_attendance_events_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_attendance_events_event_id  UUID NOT NULL
    REFERENCES class_events(class_events_id) ON DELETE CASCADE,

  -- window absensi (jika NULL: bisa fallback ke jam event)
  class_attendance_events_open_at   TIMESTAMPTZ,
  class_attendance_events_close_at  TIMESTAMPTZ,

  -- konfigurasi metode
  class_attendance_events_method    VARCHAR(16), -- 'qr'|'manual'|'geo'|'hybrid'
  class_attendance_events_note      TEXT,

  -- geofence
  class_attendance_events_lat       DOUBLE PRECISION,
  class_attendance_events_lng       DOUBLE PRECISION,
  class_attendance_events_radius_m  INTEGER,

  -- audit
  class_attendance_events_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_events_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_events_deleted_at TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_class_attendance_events_method
    CHECK (class_attendance_events_method IS NULL OR class_attendance_events_method IN ('qr','manual','geo','hybrid')),
  CONSTRAINT chk_class_attendance_events_geo
    CHECK (
      (class_attendance_events_lat IS NULL AND class_attendance_events_lng IS NULL AND class_attendance_events_radius_m IS NULL)
      OR
      (class_attendance_events_lat IS NOT NULL AND class_attendance_events_lng IS NOT NULL AND class_attendance_events_radius_m IS NOT NULL AND class_attendance_events_radius_m > 0)
    ),
  CONSTRAINT chk_class_attendance_events_window
    CHECK (class_attendance_events_close_at IS NULL OR class_attendance_events_open_at IS NULL OR class_attendance_events_close_at >= class_attendance_events_open_at)
);

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_class_attendance_events_event
  ON class_attendance_events(class_attendance_events_event_id);
CREATE INDEX IF NOT EXISTS idx_class_attendance_events_masjid
  ON class_attendance_events(class_attendance_events_masjid_id);


/* =========================================================
   4) USER_CLASS_ATTENDANCE_EVENTS — RSVP/kehadiran per user
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_class_attendance_events (
  user_class_attendance_events_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_attendance_events_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_class_attendance_events_event_id   UUID NOT NULL
    REFERENCES class_events(class_events_id) ON DELETE CASCADE,

  -- identitas peserta (TEPAT SATU)
  user_class_attendance_events_user_id            UUID,
  user_class_attendance_events_masjid_student_id  UUID,
  user_class_attendance_events_guardian_id        UUID,

  -- RSVP & kehadiran
  user_class_attendance_events_rsvp_status        VARCHAR(16),   -- invited|going|maybe|declined|waitlist
  user_class_attendance_events_checked_in_at      TIMESTAMPTZ,
  user_class_attendance_events_no_show            BOOLEAN NOT NULL DEFAULT FALSE,

  -- tiket opsional
  user_class_attendance_events_ticket_code        VARCHAR(64),

  -- catatan
  user_class_attendance_events_note               TEXT,

  -- audit
  user_class_attendance_events_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_events_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_events_deleted_at         TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_user_class_attendance_events_identity_one
    CHECK (num_nonnulls(
      user_class_attendance_events_user_id,
      user_class_attendance_events_masjid_student_id,
      user_class_attendance_events_guardian_id
    ) = 1),
  CONSTRAINT chk_user_class_attendance_events_rsvp
    CHECK (user_class_attendance_events_rsvp_status IS NULL OR
           user_class_attendance_events_rsvp_status IN ('invited','going','maybe','declined','waitlist'))
);

-- uniqueness per event + identitas
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_attendance_events_unique_identity
ON user_class_attendance_events(
  user_class_attendance_events_event_id,
  COALESCE(user_class_attendance_events_user_id,           '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_attendance_events_masjid_student_id, '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_attendance_events_guardian_id,       '00000000-0000-0000-0000-000000000000'::uuid)
);

-- Indeks bantu query
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_event
  ON user_class_attendance_events(user_class_attendance_events_event_id);
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_masjid_rsvp
  ON user_class_attendance_events(user_class_attendance_events_masjid_id, user_class_attendance_events_rsvp_status);
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_checkedin
  ON user_class_attendance_events(user_class_attendance_events_event_id, user_class_attendance_events_checked_in_at);

-- (Opsional) Ambil daftar kehadiran per user dalam satu masjid
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_masjid_user
  ON user_class_attendance_events(user_class_attendance_events_masjid_id, user_class_attendance_events_user_id);



/* =========================================================
   6) USER_CLASS_ATTENDANCE_EVENTS_URLS — URL lampiran per kehadiran user
      dengan informasi pengirim (teacher/user)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_class_attendance_events_urls (
  user_class_attendance_events_url_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_attendance_events_url_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_class_attendance_events_url_attendance_id UUID NOT NULL
    REFERENCES user_class_attendance_events(user_class_attendance_events_id) ON DELETE CASCADE,

  -- pengirim
  user_class_attendance_events_url_sender_role    VARCHAR(16) NOT NULL, -- 'teacher' | 'user'
  user_class_attendance_events_url_sender_user_id   UUID,  -- jika role = 'user'
  user_class_attendance_events_url_sender_teacher_id UUID, -- jika role = 'teacher'
  user_class_attendance_events_url_message         TEXT,   -- opsional catatan singkat

  -- klasifikasi & label
  user_class_attendance_events_url_kind   VARCHAR(32) NOT NULL, -- 'image'|'file'|'video'|'audio'|'link'|'doc'...
  user_class_attendance_events_url_label  VARCHAR(160),

  -- storage (2-slot + retensi)
  user_class_attendance_events_url_url              TEXT,  -- aktif
  user_class_attendance_events_url_object_key       TEXT,
  user_class_attendance_events_url_url_old          TEXT,  -- kandidat hapus
  user_class_attendance_events_url_object_key_old   TEXT,
  user_class_attendance_events_url_delete_pending_until TIMESTAMPTZ,

  -- flag
  user_class_attendance_events_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  user_class_attendance_events_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_events_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_events_url_deleted_at TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_ucae_urls_sender_role
    CHECK (user_class_attendance_events_url_sender_role IN ('teacher','user')),
  CONSTRAINT chk_ucae_urls_sender_match
    CHECK (
      (user_class_attendance_events_url_sender_role = 'user'
        AND user_class_attendance_events_url_sender_user_id IS NOT NULL
        AND user_class_attendance_events_url_sender_teacher_id IS NULL)
      OR
      (user_class_attendance_events_url_sender_role = 'teacher'
        AND user_class_attendance_events_url_sender_teacher_id IS NOT NULL
        AND user_class_attendance_events_url_sender_user_id IS NULL)
    ),
  CONSTRAINT chk_ucae_urls_kind_nonempty
    CHECK (length(coalesce(user_class_attendance_events_url_kind,'')) > 0)
);

-- Indeks untuk query umum
CREATE INDEX IF NOT EXISTS idx_ucae_urls_attendance_kind
  ON user_class_attendance_events_urls(user_class_attendance_events_url_attendance_id,
                                       user_class_attendance_events_url_kind);

CREATE INDEX IF NOT EXISTS idx_ucae_urls_primary
  ON user_class_attendance_events_urls(user_class_attendance_events_url_attendance_id,
                                       user_class_attendance_events_url_is_primary);

CREATE INDEX IF NOT EXISTS idx_ucae_urls_masjid_sender
  ON user_class_attendance_events_urls(user_class_attendance_events_url_masjid_id,
                                       user_class_attendance_events_url_sender_role);

-- (Opsional) ambil cepat berdasarkan pengirim user/teacher
CREATE INDEX IF NOT EXISTS idx_ucae_urls_sender_user
  ON user_class_attendance_events_urls(user_class_attendance_events_url_sender_user_id);
CREATE INDEX IF NOT EXISTS idx_ucae_urls_sender_teacher
  ON user_class_attendance_events_urls(user_class_attendance_events_url_sender_teacher_id);

COMMIT;
