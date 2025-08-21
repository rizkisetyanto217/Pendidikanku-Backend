-- =========================================================
-- MIGRATION: events, event_sessions, user_event_registrations
-- Optimized indexing + search + triggers (TIMESTAMP)
-- =========================================================

-- =========================
-- ========== UP ===========
-- =========================
BEGIN;

-- Ekstensi yang dibutuhkan
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram indexes
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- umum / aman disiapkan

/* -------------------------------------------------------
   Trigger functions: updated_at per tabel
------------------------------------------------------- */
CREATE OR REPLACE FUNCTION fn_touch_events_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.event_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_event_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.event_session_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_user_event_regs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_event_registration_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;


/* -------------------------------------------------------
   TABEL: events
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS events (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_title      VARCHAR(255) NOT NULL,
  event_slug       VARCHAR(100)  NOT NULL,
  event_description TEXT,
  event_location   VARCHAR(255),
  event_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  event_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  event_updated_at TIMESTAMP,
  -- slug unik per masjid (case-insensitive)
  CONSTRAINT ux_events_slug_per_masjid_ci UNIQUE (event_masjid_id, event_slug)
);

-- Pastikan unik case-insensitive (pakai index di LOWER)
DROP INDEX IF EXISTS idx_event_slug;
CREATE UNIQUE INDEX IF NOT EXISTS ux_events_slug_per_masjid_lower
  ON events (event_masjid_id, LOWER(event_slug));

-- Full‑text search column (judul + deskripsi + lokasi)
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS event_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(event_title, '')), 'A')
    || setweight(to_tsvector('simple', coalesce(event_description, '')), 'B')
    || setweight(to_tsvector('simple', coalesce(event_location, '')), 'C')
  ) STORED;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_events_masjid_id        ON events(event_masjid_id);
CREATE INDEX IF NOT EXISTS idx_events_masjid_recent    ON events(event_masjid_id, event_created_at DESC);

-- FTS + Trigram
CREATE INDEX IF NOT EXISTS idx_events_tsv_gin          ON events USING GIN (event_search_tsv);
CREATE INDEX IF NOT EXISTS idx_events_title_trgm       ON events USING GIN (event_title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_events_slug_trgm        ON events USING GIN (event_slug  gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_events_touch ON events;
CREATE TRIGGER trg_events_touch
BEFORE UPDATE ON events
FOR EACH ROW EXECUTE FUNCTION fn_touch_events_updated_at();


/* -------------------------------------------------------
   TABEL: event_sessions
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS event_sessions (
  event_session_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_session_event_id     UUID NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
  event_session_slug         VARCHAR(100) NOT NULL,
  event_session_title        VARCHAR(255) NOT NULL,
  event_session_description  TEXT,
  event_session_start_time   TIMESTAMP NOT NULL,
  event_session_end_time     TIMESTAMP NOT NULL,
  event_session_location     VARCHAR(255),
  event_session_image_url    TEXT,
  event_session_capacity     INT,
  event_session_is_public    BOOLEAN NOT NULL DEFAULT TRUE,
  event_session_is_registration_required BOOLEAN NOT NULL DEFAULT FALSE,
  event_session_masjid_id    UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  event_session_created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
  event_session_created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  event_session_updated_at   TIMESTAMP,
  -- Constraints
  CONSTRAINT chk_event_session_capacity_nonneg CHECK (event_session_capacity IS NULL OR event_session_capacity >= 0),
  CONSTRAINT chk_event_session_time_order      CHECK (event_session_end_time > event_session_start_time)
);

-- Slug unik case-insensitive (global)
CREATE UNIQUE INDEX IF NOT EXISTS ux_event_sessions_slug_ci
  ON event_sessions (LOWER(event_session_slug));

-- Full‑text search (judul + deskripsi + lokasi)
ALTER TABLE event_sessions
  ADD COLUMN IF NOT EXISTS event_session_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(event_session_title, '')), 'A')
    || setweight(to_tsvector('simple', coalesce(event_session_description, '')), 'B')
    || setweight(to_tsvector('simple', coalesce(event_session_location, '')), 'C')
  ) STORED;

-- Indexes umum
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_id     ON event_sessions(event_session_event_id);
CREATE INDEX IF NOT EXISTS idx_event_sessions_start_time   ON event_sessions(event_session_start_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_start  ON event_sessions(event_session_event_id, event_session_start_time);
CREATE INDEX IF NOT EXISTS idx_event_sessions_masjid_start ON event_sessions(event_session_masjid_id, event_session_start_time);

-- Partial: public only
CREATE INDEX IF NOT EXISTS idx_event_sessions_public_start
  ON event_sessions(event_session_start_time)
  WHERE event_session_is_public = TRUE;

-- FTS + Trigram
CREATE INDEX IF NOT EXISTS idx_event_sessions_tsv_gin      ON event_sessions USING GIN (event_session_search_tsv);
CREATE INDEX IF NOT EXISTS idx_event_sessions_title_trgm   ON event_sessions USING GIN (event_session_title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_event_sessions_slug_trgm    ON event_sessions USING GIN (event_session_slug  gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_event_sessions_touch ON event_sessions;
CREATE TRIGGER trg_event_sessions_touch
BEFORE UPDATE ON event_sessions
FOR EACH ROW EXECUTE FUNCTION fn_touch_event_sessions_updated_at();


/* -------------------------------------------------------
   TABEL: user_event_registrations
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS user_event_registrations (
  user_event_registration_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_event_registration_event_session_id UUID NOT NULL REFERENCES event_sessions(event_session_id) ON DELETE CASCADE,
  user_event_registration_user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_event_registration_masjid_id        UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_event_registration_status           VARCHAR(50) NOT NULL DEFAULT 'registered',
  user_event_registration_registered_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_event_registration_updated_at       TIMESTAMP,
  UNIQUE(user_event_registration_event_session_id, user_event_registration_user_id),
  CONSTRAINT chk_user_event_registration_status CHECK (
    user_event_registration_status IN ('registered','waitlisted','cancelled','attended','no_show')
  )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_event_session_id
  ON user_event_registrations(user_event_registration_event_session_id);
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_user_id
  ON user_event_registrations(user_event_registration_user_id);
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_masjid_id
  ON user_event_registrations(user_event_registration_masjid_id);

-- Composite & partial
CREATE INDEX IF NOT EXISTS idx_user_event_regs_session_status
  ON user_event_registrations(user_event_registration_event_session_id, user_event_registration_status);
CREATE INDEX IF NOT EXISTS idx_user_event_regs_registered_only
  ON user_event_registrations(user_event_registration_event_session_id)
  WHERE user_event_registration_status = 'registered';

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_user_event_regs_touch ON user_event_registrations;
CREATE TRIGGER trg_user_event_regs_touch
BEFORE UPDATE ON user_event_registrations
FOR EACH ROW EXECUTE FUNCTION fn_touch_user_event_regs_updated_at();

COMMIT;