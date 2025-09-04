-- =========================================================
-- MIGRATION: events, event_sessions, user_event_registrations
-- Dengan deleted_at + optimized indexing + triggers (alive-only unique)
-- =========================================================

BEGIN;

-- Ekstensi
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;


/* -------------------------------------------------------
   TABEL: events
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS events (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_title      VARCHAR(255) NOT NULL,
  event_slug       VARCHAR(100) NOT NULL,
  event_description TEXT,
  event_location   VARCHAR(255),
  event_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  event_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_deleted_at TIMESTAMPTZ,
  -- constraint unik lama akan di-drop & diganti partial unique
  CONSTRAINT ux_events_slug_per_masjid_ci UNIQUE (event_masjid_id, event_slug)
);

-- Drop index lama (kalau ada)
DROP INDEX IF EXISTS idx_event_slug;
CREATE INDEX IF NOT EXISTS idx_events_masjid_id     ON events(event_masjid_id);
CREATE INDEX IF NOT EXISTS idx_events_tsv_gin       ON events USING GIN (event_search_tsv);
CREATE INDEX IF NOT EXISTS idx_events_title_trgm    ON events USING GIN (event_title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_events_slug_trgm     ON events USING GIN (event_slug  gin_trgm_ops);

-- Full text search (generated)
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS event_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(event_title, '')), 'A')
    || setweight(to_tsvector('simple', coalesce(event_description, '')), 'B')
    || setweight(to_tsvector('simple', coalesce(event_location, '')), 'C')
  ) STORED;

-- Alive-only helper index
CREATE INDEX IF NOT EXISTS idx_events_masjid_recent_alive
  ON events(event_masjid_id, event_created_at DESC)
  WHERE event_deleted_at IS NULL;

-- Ganti UNIQUE → partial unique (alive only)
ALTER TABLE events
  DROP CONSTRAINT IF EXISTS ux_events_slug_per_masjid_ci;
DROP INDEX IF EXISTS ux_events_slug_per_masjid_lower;
CREATE UNIQUE INDEX IF NOT EXISTS ux_events_slug_per_masjid_lower_alive
  ON events (event_masjid_id, LOWER(event_slug))
  WHERE event_deleted_at IS NULL;


/* -------------------------------------------------------
   TABEL: event_sessions
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS event_sessions (
  event_session_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_session_event_id     UUID NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
  event_session_slug         VARCHAR(100) NOT NULL,
  event_session_title        VARCHAR(255) NOT NULL,
  event_session_description  TEXT,
  event_session_start_time   TIMESTAMPTZ NOT NULL,
  event_session_end_time     TIMESTAMPTZ NOT NULL,
  event_session_location     VARCHAR(255),
  event_session_image_url    TEXT,
  event_session_capacity     INT,
  event_session_is_public    BOOLEAN NOT NULL DEFAULT TRUE,
  event_session_is_registration_required BOOLEAN NOT NULL DEFAULT FALSE,
  event_session_masjid_id    UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  event_session_created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
  event_session_created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_session_updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_session_deleted_at   TIMESTAMPTZ,
  CONSTRAINT chk_event_session_capacity_nonneg CHECK (event_session_capacity IS NULL OR event_session_capacity >= 0),
  CONSTRAINT chk_event_session_time_order    CHECK (event_session_end_time > event_session_start_time)
);

-- FTS (generated)
ALTER TABLE event_sessions
  ADD COLUMN IF NOT EXISTS event_session_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(event_session_title, '')), 'A')
    || setweight(to_tsvector('simple', coalesce(event_session_description, '')), 'B')
    || setweight(to_tsvector('simple', coalesce(event_session_location, '')), 'C')
  ) STORED;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_id       ON event_sessions(event_session_event_id);
CREATE INDEX IF NOT EXISTS idx_event_sessions_start_time     ON event_sessions(event_session_start_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_start    ON event_sessions(event_session_event_id, event_session_start_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_masjid_start   ON event_sessions(event_session_masjid_id, event_session_start_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_public_start   ON event_sessions(event_session_start_time) WHERE event_session_is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_event_sessions_tsv_gin        ON event_sessions USING GIN (event_session_search_tsv);
CREATE INDEX IF NOT EXISTS idx_event_sessions_title_trgm     ON event_sessions USING GIN (event_session_title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_event_sessions_slug_trgm      ON event_sessions USING GIN (event_session_slug  gin_trgm_ops);

-- Alive-only helper index
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_start_alive
  ON event_sessions(event_session_event_id, event_session_start_time)
  WHERE event_session_deleted_at IS NULL;

-- Ganti UNIQUE slug → partial unique (alive only)
DROP INDEX IF EXISTS ux_event_sessions_slug_ci;
CREATE UNIQUE INDEX IF NOT EXISTS ux_event_sessions_slug_ci_alive
  ON event_sessions (LOWER(event_session_slug))
  WHERE event_session_deleted_at IS NULL;


/* -------------------------------------------------------
   TABEL: user_event_registrations
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS user_event_registrations (
  user_event_registration_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_event_registration_event_session_id UUID NOT NULL REFERENCES event_sessions(event_session_id) ON DELETE CASCADE,
  user_event_registration_user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_event_registration_masjid_id        UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_event_registration_status           VARCHAR(50) NOT NULL DEFAULT 'registered',
  user_event_registration_registered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_event_registration_updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_event_registration_deleted_at       TIMESTAMPTZ,
  -- unik inline lama akan di-drop & diganti partial unique
  UNIQUE(user_event_registration_event_session_id, user_event_registration_user_id),
  CONSTRAINT chk_user_event_registration_status CHECK (
    user_event_registration_status IN ('registered','waitlisted','cancelled','attended','no_show')
  )
);

-- Index umum
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_event_session_id
  ON user_event_registrations(user_event_registration_event_session_id);
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_user_id
  ON user_event_registrations(user_event_registration_user_id);
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_masjid_id
  ON user_event_registrations(user_event_registration_masjid_id);

CREATE INDEX IF NOT EXISTS idx_user_event_regs_session_status
  ON user_event_registrations(user_event_registration_event_session_id, user_event_registration_status);
CREATE INDEX IF NOT EXISTS idx_user_event_regs_registered_only
  ON user_event_registrations(user_event_registration_event_session_id)
  WHERE user_event_registration_status = 'registered';

-- Ganti UNIQUE daftar (session,user) → partial unique (alive only)
ALTER TABLE user_event_registrations
  DROP CONSTRAINT IF EXISTS user_event_registrations_user_event_registration_event_session_id_user__key;

CREATE UNIQUE INDEX IF NOT EXISTS ux_user_event_regs_session_user_alive
  ON user_event_registrations (user_event_registration_event_session_id, user_event_registration_user_id)
  WHERE user_event_registration_deleted_at IS NULL;

COMMIT;
