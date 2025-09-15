BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- GIN trigram untuk ILIKE/FTS opsional

-- ENUMS tambahan (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_visibility_enum') THEN
    CREATE TYPE class_visibility_enum AS ENUM ('public','unlisted','private');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'class_delivery_mode_enum') THEN
    CREATE TYPE class_delivery_mode_enum AS ENUM ('online','offline','hybrid');
  END IF;
END$$;

-- =========================================================
-- BERSIHKAN DATA & SKEMA LAMA
-- =========================================================
DROP TABLE IF EXISTS class_sections CASCADE;

-- =========================================================
-- CREATE: class_sections (final, lengkap)
-- =========================================================
CREATE TABLE class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi inti
  class_sections_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_sections_class_id      UUID NOT NULL,
  class_sections_teacher_id    UUID,
  class_sections_assistant_teacher_id UUID,
  class_sections_class_room_id UUID,

  -- Identitas
  class_sections_slug  VARCHAR(160) NOT NULL,
  class_sections_name  VARCHAR(100) NOT NULL,
  class_sections_code  VARCHAR(50),

  -- Jadwal (dua bentuk: string bebas & terstruktur)
  class_sections_schedule             VARCHAR(200), -- contoh: "Jumat 19:00–21:00"
  class_sections_delivery_mode        class_delivery_mode_enum,
  class_sections_timezone             VARCHAR(40),  -- ex: 'Asia/Jakarta'
  class_sections_rrule                TEXT,         -- ex: FREQ=WEEKLY;BYDAY=FR
  class_sections_duration_minutes     SMALLINT,     -- 0..32767
  class_sections_default_meeting_day  SMALLINT,     -- 0=Sun..6=Sat

  -- Tanggal efektif & jumlah sesi
  class_sections_start_date     DATE,
  class_sections_end_date       DATE,

  -- Meeting / Group
  class_sections_group_url                TEXT,


  -- Kapasitas & enrollment
  class_sections_capacity       INT,
  class_sections_total_students INT NOT NULL DEFAULT 0,
  class_sections_enrollment_requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
  class_sections_invite_only                  BOOLEAN NOT NULL DEFAULT FALSE,
  class_sections_invite_code                  VARCHAR(32),

  -- Kehadiran
  class_sections_attendance_required       BOOLEAN  NOT NULL DEFAULT TRUE,

  -- Visibility & publishing
  class_sections_is_active      BOOLEAN NOT NULL DEFAULT TRUE,

  -- Audit & optimistic locking
  class_sections_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_deleted_at  TIMESTAMPTZ,

  -- Tenant-safe pair
  CONSTRAINT uq_class_sections_id_masjid UNIQUE (class_sections_id, class_sections_masjid_id),

  -- ======================
  -- CHECK guards
  -- ======================
  CONSTRAINT ck_sections_capacity_nonneg
    CHECK (class_sections_capacity IS NULL OR class_sections_capacity >= 0),
  CONSTRAINT ck_sections_total_nonneg
    CHECK (class_sections_total_students >= 0),
  CONSTRAINT ck_sections_total_le_capacity
    CHECK (class_sections_capacity IS NULL OR class_sections_total_students <= class_sections_capacity),

  CONSTRAINT ck_sections_group_url_scheme
    CHECK (class_sections_group_url IS NULL OR class_sections_group_url ~* '^(https?)://'),

  CONSTRAINT ck_sections_waitlist_nonneg
    CHECK (class_sections_waitlist_count >= 0),

  CONSTRAINT ck_sections_duration_nonneg
    CHECK (class_sections_duration_minutes IS NULL OR class_sections_duration_minutes >= 0),

  CONSTRAINT ck_sections_meeting_day_range
    CHECK (class_sections_default_meeting_day IS NULL OR class_sections_default_meeting_day BETWEEN 0 AND 6),

  CONSTRAINT ck_sections_lat_lng_bounds
    CHECK (
      (class_sections_location_lat IS NULL OR (class_sections_location_lat BETWEEN -90  AND 90)) AND
      (class_sections_location_lng IS NULL OR (class_sections_location_lng BETWEEN -180 AND 180))
    ),

  CONSTRAINT ck_sections_publish_window
    CHECK (class_sections_publish_at IS NULL OR class_sections_unpublish_at IS NULL OR class_sections_unpublish_at >= class_sections_publish_at),

  CONSTRAINT ck_sections_slug_fmt
    CHECK (class_sections_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),

  CONSTRAINT ck_sections_code_fmt
    CHECK (class_sections_code IS NULL OR class_sections_code ~ '^[A-Z0-9._-]+$'),

  CONSTRAINT ck_sections_reg_window
    CHECK (
      class_sections_registration_opens_at IS NULL
      OR class_sections_registration_closes_at IS NULL
      OR class_sections_registration_closes_at >= class_sections_registration_opens_at
    ),

  CONSTRAINT ck_sections_age_range
    CHECK (class_sections_min_age IS NULL OR class_sections_max_age IS NULL OR class_sections_max_age >= class_sections_min_age),

  CONSTRAINT ck_sections_meetings_nonneg
    CHECK (class_sections_total_meetings IS NULL OR class_sections_total_meetings >= 0),

  CONSTRAINT ck_sections_grace_nonneg
    CHECK (class_sections_attendance_grace_minutes IS NULL OR class_sections_attendance_grace_minutes >= 0),

  CONSTRAINT ck_sections_dates_order
    CHECK (class_sections_start_date IS NULL OR class_sections_end_date IS NULL OR class_sections_end_date >= class_sections_start_date),

  CONSTRAINT ck_sections_rating_bounds
    CHECK (class_sections_rating_avg IS NULL OR (class_sections_rating_avg >= 0 AND class_sections_rating_avg <= 5)),

  -- ======================
  -- FK KOMPOSIT tenant-safe
  -- ======================
  CONSTRAINT fk_sections_class_same_masjid
    FOREIGN KEY (class_sections_class_id, class_sections_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_sections_teacher_same_masjid
    FOREIGN KEY (class_sections_teacher_id, class_sections_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_sections_room_same_masjid
    FOREIGN KEY (class_sections_class_room_id, class_sections_masjid_id)
    REFERENCES class_rooms (class_room_id, class_rooms_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =========================================================
-- INDEXES & UNIQUES
-- =========================================================

-- Unik: slug per masjid (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_slug_per_masjid_alive
  ON class_sections (class_sections_masjid_id, LOWER(class_sections_slug))
  WHERE class_sections_deleted_at IS NULL;

-- Unik: nama section per class (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name_alive
  ON class_sections (class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

-- Unik: code per class (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_code_per_class_alive
  ON class_sections (class_sections_class_id, LOWER(class_sections_code))
  WHERE class_sections_deleted_at IS NULL
    AND class_sections_code IS NOT NULL;

-- Unik: external_ref per masjid (untuk sinkronisasi)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_external_ref_per_masjid
  ON class_sections (class_sections_masjid_id, LOWER(class_sections_external_ref))
  WHERE class_sections_deleted_at IS NULL
    AND class_sections_external_ref IS NOT NULL;

-- Lookup dasar FKs
CREATE INDEX IF NOT EXISTS idx_sections_masjid     ON class_sections (class_sections_masjid_id);
CREATE INDEX IF NOT EXISTS idx_sections_class      ON class_sections (class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_sections_teacher    ON class_sections (class_sections_teacher_id);
CREATE INDEX IF NOT EXISTS idx_sections_assistant_teacher ON class_sections (class_sections_assistant_teacher_id);
CREATE INDEX IF NOT EXISTS idx_sections_class_room ON class_sections (class_sections_class_room_id);

-- Listing umum
CREATE INDEX IF NOT EXISTS ix_sections_masjid_active_created
  ON class_sections (class_sections_masjid_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sections_class_active_created
  ON class_sections (class_sections_class_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

-- Listing by teacher/room
CREATE INDEX IF NOT EXISTS ix_sections_teacher_active_created
  ON class_sections (class_sections_teacher_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sections_room_active_created
  ON class_sections (class_sections_class_room_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

-- Visibility & publish
CREATE INDEX IF NOT EXISTS idx_sections_visibility_alive
  ON class_sections (class_sections_visibility)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sections_publish_window_alive
  ON class_sections (class_sections_publish_at, class_sections_unpublish_at)
  WHERE class_sections_deleted_at IS NULL;

-- Window pendaftaran
CREATE INDEX IF NOT EXISTS ix_sections_reg_window_alive
  ON class_sections (class_sections_registration_opens_at, class_sections_registration_closes_at)
  WHERE class_sections_deleted_at IS NULL;

-- Rating & tanggal efektif
CREATE INDEX IF NOT EXISTS idx_sections_rating_alive
  ON class_sections (class_sections_rating_avg DESC, class_sections_rating_count DESC)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sections_dates_alive
  ON class_sections (class_sections_start_date, class_sections_end_date)
  WHERE class_sections_deleted_at IS NULL;

-- Tagging & meta
CREATE INDEX IF NOT EXISTS gin_sections_tags_alive
  ON class_sections USING GIN (class_sections_tags)
  WHERE class_sections_deleted_at IS NULL;

-- Meeting/provider
CREATE INDEX IF NOT EXISTS idx_sections_meeting_provider_alive
  ON class_sections (class_sections_meeting_platform, class_sections_meeting_provider_event_id)
  WHERE class_sections_deleted_at IS NULL;

-- Pencarian teks cepat
CREATE INDEX IF NOT EXISTS gin_sections_name_trgm_alive
  ON class_sections USING GIN (LOWER(class_sections_name) gin_trgm_ops)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_sections_slug_trgm_alive
  ON class_sections USING GIN (LOWER(class_sections_slug) gin_trgm_ops)
  WHERE class_sections_deleted_at IS NULL;

-- Lokasi & timezone
CREATE INDEX IF NOT EXISTS idx_sections_timezone_alive
  ON class_sections (class_sections_timezone)
  WHERE class_sections_deleted_at IS NULL;

-- Optimistic locking
CREATE INDEX IF NOT EXISTS idx_sections_row_version_alive
  ON class_sections (class_sections_row_version)
  WHERE class_sections_deleted_at IS NULL;

-- BRIN untuk tabel besar berbasis waktu
CREATE INDEX IF NOT EXISTS brin_sections_created_at
  ON class_sections USING BRIN (class_sections_created_at);

COMMIT;
