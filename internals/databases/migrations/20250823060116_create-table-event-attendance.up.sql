BEGIN;

/* =========================================================
   1) CLASS_ATTENDANCE_EVENTS — window/metode absensi per event
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_attendance_events (
  class_attendance_event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_attendance_event_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_attendance_event_event_id   UUID NOT NULL
    REFERENCES class_events(class_event_id) ON DELETE CASCADE,

  -- window absensi (jika NULL: fallback ke jam event)
  class_attendance_event_open_at    TIMESTAMPTZ,
  class_attendance_event_close_at   TIMESTAMPTZ,

  -- konfigurasi metode
  class_attendance_event_method     VARCHAR(16), -- 'qr'|'manual'|'geo'|'hybrid'
  class_attendance_event_note       TEXT,

  -- audit
  class_attendance_event_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_event_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_event_deleted_at TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_class_attendance_event_method
    CHECK (class_attendance_event_method IS NULL OR class_attendance_event_method IN ('qr','manual','geo','hybrid')),
  CONSTRAINT chk_class_attendance_event_window
    CHECK (
      class_attendance_event_close_at IS NULL
      OR class_attendance_event_open_at IS NULL
      OR class_attendance_event_close_at >= class_attendance_event_open_at
    )
);

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_class_attendance_events_event
  ON class_attendance_events(class_attendance_event_event_id);
CREATE INDEX IF NOT EXISTS idx_class_attendance_events_masjid
  ON class_attendance_events(class_attendance_event_masjid_id);



/* =========================================================
   2) CLASS_ATTENDANCE_EVENT_URLS — lampiran per attendance-event
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_attendance_event_urls (
  class_attendance_event_url_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & owner
  class_attendance_event_url_masjid_id           UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_attendance_event_url_attendance_event_id UUID NOT NULL
    REFERENCES class_attendance_events(class_attendance_event_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- jenis/peran aset (mis. 'banner','image','video','attachment','link')
  class_attendance_event_url_kind                VARCHAR(24) NOT NULL,

  -- lokasi file/link
  class_attendance_event_url_href                TEXT,        -- URL publik (boleh NULL jika murni object storage)
  class_attendance_event_url_object_key          TEXT,        -- object key aktif di storage
  class_attendance_event_url_object_key_old      TEXT,        -- object key lama (retensi in-place replace)

  -- tampilan
  class_attendance_event_url_label               VARCHAR(160),
  class_attendance_event_url_order               INT NOT NULL DEFAULT 0,
  class_attendance_event_url_is_primary          BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit & retensi
  class_attendance_event_url_created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_event_url_updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_event_url_deleted_at          TIMESTAMPTZ,          -- soft delete (versi-per-baris)
  class_attendance_event_url_delete_pending_until TIMESTAMPTZ,         -- tenggat purge

  -- guards
  CONSTRAINT chk_caeu_kind_nonempty
    CHECK (length(coalesce(class_attendance_event_url_kind,'')) > 0)
);

-- Indexing / optimization
CREATE INDEX IF NOT EXISTS idx_class_attendance_event_urls_owner_live
  ON class_attendance_event_urls (
    class_attendance_event_url_attendance_event_id,
    class_attendance_event_url_kind,
    class_attendance_event_url_is_primary DESC,
    class_attendance_event_url_order,
    class_attendance_event_url_created_at
  )
  WHERE class_attendance_event_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_attendance_event_urls_masjid_live
  ON class_attendance_event_urls (class_attendance_event_url_masjid_id)
  WHERE class_attendance_event_url_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_class_attendance_event_urls_primary_per_kind_alive
  ON class_attendance_event_urls (
    class_attendance_event_url_attendance_event_id,
    class_attendance_event_url_kind
  )
  WHERE class_attendance_event_url_deleted_at IS NULL
    AND class_attendance_event_url_is_primary = TRUE;

CREATE INDEX IF NOT EXISTS idx_class_attendance_event_urls_purge_due
  ON class_attendance_event_urls (class_attendance_event_url_delete_pending_until)
  WHERE class_attendance_event_url_delete_pending_until IS NOT NULL
    AND (
      (class_attendance_event_url_deleted_at IS NULL  AND class_attendance_event_url_object_key_old IS NOT NULL) OR
      (class_attendance_event_url_deleted_at IS NOT NULL AND class_attendance_event_url_object_key     IS NOT NULL)
    );

-- (opsional) trigram search untuk label
CREATE INDEX IF NOT EXISTS gin_class_attendance_event_url_label_trgm_live
  ON class_attendance_event_urls USING GIN (class_attendance_event_url_label gin_trgm_ops)
  WHERE class_attendance_event_url_deleted_at IS NULL;



/* =========================================================
   3) USER_CLASS_ATTENDANCE_EVENTS — RSVP/kehadiran per user
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_class_attendance_events (
  user_class_attendance_event_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_class_attendance_event_masjid_id         UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_class_attendance_event_event_id          UUID NOT NULL
    REFERENCES class_events(class_event_id) ON DELETE CASCADE,

  -- identitas peserta (TEPAT SATU)
  user_class_attendance_event_user_id           UUID,
  user_class_attendance_event_masjid_student_id UUID,
  user_class_attendance_event_guardian_id       UUID,

  -- RSVP & kehadiran
  user_class_attendance_event_rsvp_status       VARCHAR(16),   -- invited|going|maybe|declined|waitlist
  user_class_attendance_event_checked_in_at     TIMESTAMPTZ,
  user_class_attendance_event_no_show           BOOLEAN NOT NULL DEFAULT FALSE,

  -- tiket opsional
  user_class_attendance_event_ticket_code       VARCHAR(64),

  -- catatan
  user_class_attendance_event_note              TEXT,

  -- audit
  user_class_attendance_event_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_event_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_event_deleted_at        TIMESTAMPTZ,

  -- guards
  CONSTRAINT chk_user_class_attendance_event_identity_one
    CHECK (num_nonnulls(
      user_class_attendance_event_user_id,
      user_class_attendance_event_masjid_student_id,
      user_class_attendance_event_guardian_id
    ) = 1),
  CONSTRAINT chk_user_class_attendance_event_rsvp
    CHECK (user_class_attendance_event_rsvp_status IS NULL OR
           user_class_attendance_event_rsvp_status IN ('invited','going','maybe','declined','waitlist'))
);

-- Unik per event + identitas
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_class_attendance_events_unique_identity
ON user_class_attendance_events(
  user_class_attendance_event_event_id,
  COALESCE(user_class_attendance_event_user_id,           '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_attendance_event_masjid_student_id, '00000000-0000-0000-0000-000000000000'::uuid),
  COALESCE(user_class_attendance_event_guardian_id,       '00000000-0000-0000-0000-000000000000'::uuid)
);

-- Indeks bantu
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_event
  ON user_class_attendance_events(user_class_attendance_event_event_id);
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_masjid_rsvp
  ON user_class_attendance_events(user_class_attendance_event_masjid_id, user_class_attendance_event_rsvp_status);
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_checkedin
  ON user_class_attendance_events(user_class_attendance_event_event_id, user_class_attendance_event_checked_in_at);

-- (opsional) daftar kehadiran per user dalam satu masjid
CREATE INDEX IF NOT EXISTS idx_user_class_attendance_events_masjid_user
  ON user_class_attendance_events(user_class_attendance_event_masjid_id, user_class_attendance_event_user_id);


/* =========================================================
   4) USER_CLASS_ATTENDANCE_EVENTS_URLS — URL lampiran per kehadiran user
      (mendukung pengirim: teacher | user | school)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_class_attendance_events_urls (
  user_class_attendance_event_url_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope tenant
  user_class_attendance_event_url_masjid_id        UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- parent attendance
  user_class_attendance_event_url_attendance_id    UUID NOT NULL
    REFERENCES user_class_attendance_events(user_class_attendance_event_id) ON DELETE CASCADE,

  -- pengirim (teacher | user | school)
  user_class_attendance_event_url_sender_role      VARCHAR(16) NOT NULL,
  user_class_attendance_event_url_sender_user_id   UUID,  -- jika role = 'user'
  user_class_attendance_event_url_sender_teacher_id UUID, -- jika role = 'teacher'
  user_class_attendance_event_url_message          TEXT,

  -- klasifikasi & label
  user_class_attendance_event_url_kind             VARCHAR(32) NOT NULL,
  user_class_attendance_event_url_label            VARCHAR(160),

  -- storage (2-slot + retensi)
  user_class_attendance_event_url_url                  TEXT,      -- aktif
  user_class_attendance_event_url_object_key           TEXT,
  user_class_attendance_event_url_url_old              TEXT,      -- kandidat purge
  user_class_attendance_event_url_object_key_old       TEXT,
  user_class_attendance_event_url_delete_pending_until TIMESTAMPTZ,

  -- flag
  user_class_attendance_event_url_is_primary       BOOLEAN NOT NULL DEFAULT FALSE,

  -- audit
  user_class_attendance_event_url_created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_event_url_updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_event_url_deleted_at       TIMESTAMPTZ,

  /* =========================
     FOREIGN KEYS (tenant-safe)
     ========================= */

  -- teacher: FK komposit (id, masjid_id) → masjid_teachers
  CONSTRAINT fk_ucae_url_sender_teacher
    FOREIGN KEY (user_class_attendance_event_url_sender_teacher_id,
                 user_class_attendance_event_url_masjid_id)
    REFERENCES masjid_teachers(masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- user: langsung ke users(id)
  CONSTRAINT fk_ucae_url_sender_user
    FOREIGN KEY (user_class_attendance_event_url_sender_user_id)
    REFERENCES users(id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  /* =========================
     CHECK guards
     ========================= */
  CONSTRAINT chk_ucae_url_sender_role
    CHECK (user_class_attendance_event_url_sender_role IN ('teacher','student','dkm')),

  CONSTRAINT chk_ucae_url_sender_match
    CHECK (
      (user_class_attendance_event_url_sender_role = 'student'
        AND user_class_attendance_event_url_sender_user_id IS NOT NULL
        AND user_class_attendance_event_url_sender_teacher_id IS NULL)
      OR
      (user_class_attendance_event_url_sender_role = 'teacher'
        AND user_class_attendance_event_url_sender_teacher_id IS NOT NULL
        AND user_class_attendance_event_url_sender_user_id IS NULL)
      OR
      (user_class_attendance_event_url_sender_role = 'dkm'
        AND user_class_attendance_event_url_sender_user_id IS NULL
        AND user_class_attendance_event_url_sender_teacher_id IS NULL)
    ),

  CONSTRAINT chk_ucae_url_kind_nonempty
    CHECK (length(coalesce(user_class_attendance_event_url_kind,'')) > 0)
);

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_ucae_urls_attendance_kind
  ON user_class_attendance_events_urls (
    user_class_attendance_event_url_attendance_id,
    user_class_attendance_event_url_kind
  );

CREATE INDEX IF NOT EXISTS idx_ucae_urls_primary
  ON user_class_attendance_events_urls (
    user_class_attendance_event_url_attendance_id,
    user_class_attendance_event_url_is_primary
  );

CREATE INDEX IF NOT EXISTS idx_ucae_urls_masjid_sender
  ON user_class_attendance_events_urls (
    user_class_attendance_event_url_masjid_id,
    user_class_attendance_event_url_sender_role
  );

CREATE INDEX IF NOT EXISTS idx_ucae_urls_sender_user
  ON user_class_attendance_events_urls(user_class_attendance_event_url_sender_user_id);

CREATE INDEX IF NOT EXISTS idx_ucae_urls_sender_teacher
  ON user_class_attendance_events_urls(user_class_attendance_event_url_sender_teacher_id);

COMMIT;
