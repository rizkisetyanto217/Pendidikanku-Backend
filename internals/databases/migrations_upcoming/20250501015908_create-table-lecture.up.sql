-- =========================================================
-- MIGRATION: lectures, user_lectures, lecture_schedules
-- with FTS + trigram + optimized indexes + triggers
-- =========================================================

-- =========================
-- ========== UP ===========
-- =========================
BEGIN;

-- Ekstensi yang dibutuhkan (idempotent)
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;      -- trigram index
CREATE EXTENSION IF NOT EXISTS btree_gin;    -- btree_gin ops (umum, aman disiapkan)

-- ---------------------------------------------------------
-- Generic trigger functions untuk updated_at (TIMESTAMPTZ)
-- ---------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_touch_updated_at_lecture()
RETURNS TRIGGER AS $$
BEGIN
  NEW.lecture_updated_at := CURRENT_TIMESTAMPTZ;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_updated_at_user_lectures()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_lecture_updated_at := CURRENT_TIMESTAMPTZ;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_touch_updated_at_lecture_schedules()
RETURNS TRIGGER AS $$
BEGIN
  NEW.lecture_schedules_updated_at := CURRENT_TIMESTAMPTZ;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------
-- TABEL: lectures
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lectures (
  lecture_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lecture_title                    VARCHAR(255) NOT NULL,
  lecture_slug                     VARCHAR(255) NOT NULL,              -- unik (case-insensitive)
  lecture_description              TEXT,
  total_lecture_sessions           INTEGER,
  lecture_image_url                TEXT,
  lecture_teachers                 JSONB,                               -- [{"id":"...","name":"..."}]
  lecture_masjid_id                UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_is_registration_required BOOLEAN NOT NULL DEFAULT FALSE,
  lecture_is_paid                  BOOLEAN NOT NULL DEFAULT FALSE,
  lecture_price                    INT,
  lecture_payment_deadline         TIMESTAMPTZ,

  lecture_capacity                 INT,

  lecture_is_active                BOOLEAN NOT NULL DEFAULT TRUE,
  lecture_is_certificate_generated BOOLEAN NOT NULL DEFAULT FALSE,

  lecture_created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  lecture_deleted_at               TIMESTAMPTZ NULL,

  -- Constraints
  CONSTRAINT lectures_price_nonneg    CHECK (lecture_price IS NULL OR lecture_price >= 0),
  CONSTRAINT lectures_capacity_nonneg CHECK (lecture_capacity IS NULL OR lecture_capacity >= 0)
);

-- Unique slug case-insensitive
CREATE UNIQUE INDEX IF NOT EXISTS ux_lectures_slug_ci ON lectures (LOWER(lecture_slug));

-- Index dasar & query umum
CREATE INDEX IF NOT EXISTS idx_lectures_masjid_id          ON lectures(lecture_masjid_id);
CREATE INDEX IF NOT EXISTS idx_lectures_created_at_desc     ON lectures(lecture_created_at DESC);

-- Per masjid + aktif + terbaru + belum terhapus
CREATE INDEX IF NOT EXISTS idx_lectures_masjid_active_recent_live
  ON lectures (lecture_masjid_id, lecture_is_active, lecture_created_at DESC)
  WHERE lecture_deleted_at IS NULL;

-- Terbaru per masjid (tanpa lihat aktif) + belum terhapus
CREATE INDEX IF NOT EXISTS idx_lectures_masjid_recent_live
  ON lectures (lecture_masjid_id, lecture_created_at DESC)
  WHERE lecture_deleted_at IS NULL;

-- JSONB teachers (opsional untuk filter by id/name)
CREATE INDEX IF NOT EXISTS idx_lectures_teachers_gin
  ON lectures USING GIN (lecture_teachers jsonb_path_ops);

-- Full-text search (judul + deskripsi)
ALTER TABLE lectures
  ADD COLUMN IF NOT EXISTS lecture_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(lecture_title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(lecture_description, '')), 'B')
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_lectures_tsv_gin
  ON lectures USING GIN (lecture_search_tsv);

-- Fuzzy search (trigram) untuk ILIKE cepat
CREATE INDEX IF NOT EXISTS idx_lectures_title_trgm
  ON lectures USING GIN (lecture_title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_lectures_slug_trgm
  ON lectures USING GIN (lecture_slug gin_trgm_ops);

-- Trigger updated_at
DROP TRIGGER IF EXISTS trg_lectures_touch ON lectures;
CREATE TRIGGER trg_lectures_touch
BEFORE UPDATE ON lectures
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_lecture();


-- ---------------------------------------------------------
-- TABEL: user_lectures
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_lectures (
  user_lecture_id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_lecture_grade_result              INT,  -- 0..100
  user_lecture_lecture_id                UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  user_lecture_user_id                   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_total_completed_sessions  INT  NOT NULL DEFAULT 0,

  -- untuk multi-masjid reporting/filter
  user_lecture_masjid_id                 UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- pendaftaran & pembayaran
  user_lecture_is_registered             BOOLEAN NOT NULL DEFAULT FALSE,
  user_lecture_has_paid                  BOOLEAN NOT NULL DEFAULT FALSE,
  user_lecture_paid_amount               INT,
  user_lecture_payment_time              TIMESTAMPTZ NULL,

  -- timestamps
  user_lecture_created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_lecture_updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_lecture_deleted_at                TIMESTAMPTZ NULL, -- soft delete

  -- unique hanya untuk row hidup (deleted_at IS NULL)
  CONSTRAINT user_lectures_grade_range            CHECK (user_lecture_grade_result IS NULL OR (user_lecture_grade_result BETWEEN 0 AND 100)),
  CONSTRAINT user_lectures_paid_amount_nonneg     CHECK (user_lecture_paid_amount IS NULL OR user_lecture_paid_amount >= 0),
  CONSTRAINT user_lectures_total_completed_nonneg CHECK (user_lecture_total_completed_sessions >= 0)
);

-- Index akses cepat
CREATE INDEX IF NOT EXISTS idx_user_lectures_lecture_id     ON user_lectures(user_lecture_lecture_id);
CREATE INDEX IF NOT EXISTS idx_user_lectures_user_id        ON user_lectures(user_lecture_user_id);
CREATE INDEX IF NOT EXISTS idx_user_lectures_masjid_id      ON user_lectures(user_lecture_masjid_id);

-- Query progress user per lecture
-- Unique partial (hanya row hidup)
DROP INDEX IF EXISTS idx_user_lectures_user_lecture_unique;
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_lectures_user_lecture_alive
  ON user_lectures(user_lecture_user_id, user_lecture_lecture_id)
  WHERE user_lecture_deleted_at IS NULL;

-- Query finansial: only paid
CREATE INDEX IF NOT EXISTS idx_user_lectures_paid_partial
  ON user_lectures(user_lecture_masjid_id, user_lecture_has_paid, user_lecture_payment_time)
  WHERE user_lecture_has_paid = TRUE AND user_lecture_deleted_at IS NULL;


-- ---------------------------------------------------------
-- TABEL: lecture_schedules
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS lecture_schedules (
  lecture_schedules_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  lecture_schedules_lecture_id         UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_schedules_title              VARCHAR(255) NOT NULL,

  lecture_schedules_day_of_week        INT  NOT NULL,        -- 0=Min .. 6=Sab
  lecture_schedules_start_time         TIME NOT NULL,
  lecture_schedules_end_time           TIME,

  lecture_schedules_place              TEXT,
  lecture_schedules_notes              TEXT,

  lecture_schedules_is_active          BOOLEAN NOT NULL DEFAULT TRUE,
  lecture_schedules_is_paid            BOOLEAN NOT NULL DEFAULT FALSE,
  lecture_schedules_price              INT,
  lecture_schedules_capacity           INT,
  lecture_schedules_is_registration_required BOOLEAN NOT NULL DEFAULT FALSE,

  lecture_schedules_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_schedules_updated_at         TIMESTAMPTZ,
  lecture_schedules_deleted_at         TIMESTAMPTZ,

  CONSTRAINT lecture_schedules_price_nonneg    CHECK (lecture_schedules_price IS NULL OR lecture_schedules_price >= 0),
  CONSTRAINT lecture_schedules_capacity_nonneg CHECK (lecture_schedules_capacity IS NULL OR lecture_schedules_capacity >= 0),
  CONSTRAINT lecture_schedules_time_order      CHECK (lecture_schedules_end_time IS NULL OR lecture_schedules_end_time > lecture_schedules_start_time),
  CONSTRAINT lecture_schedules_dow_range       CHECK (lecture_schedules_day_of_week BETWEEN 0 AND 6)
);

-- Index dasar & kalender rutin
CREATE INDEX IF NOT EXISTS idx_lecture_schedules_lecture_id
  ON lecture_schedules (lecture_schedules_lecture_id);

CREATE INDEX IF NOT EXISTS idx_lecture_schedules_day_time_live
  ON lecture_schedules (lecture_schedules_day_of_week, lecture_schedules_start_time)
  WHERE lecture_schedules_is_active = TRUE AND lecture_schedules_deleted_at IS NULL;

-- Hindari duplikasi slot jadwal pada satu lecture
CREATE UNIQUE INDEX IF NOT EXISTS ux_lecture_schedules_unique_slot
  ON lecture_schedules (lecture_schedules_lecture_id, lecture_schedules_day_of_week, lecture_schedules_start_time)
  WHERE lecture_schedules_deleted_at IS NULL;

-- Full‑text search schedules (title/place/notes)
ALTER TABLE lecture_schedules
  ADD COLUMN IF NOT EXISTS lecture_schedules_search_tsv tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(lecture_schedules_title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(lecture_schedules_place, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(lecture_schedules_notes, '')), 'C')
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_lecture_schedules_tsv_gin
  ON lecture_schedules USING GIN (lecture_schedules_search_tsv)
  WHERE lecture_schedules_deleted_at IS NULL;

-- Fuzzy (trigram) untuk ILIKE cepat
CREATE INDEX IF NOT EXISTS idx_lecture_schedules_title_trgm
  ON lecture_schedules USING GIN (lecture_schedules_title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_lecture_schedules_place_trgm
  ON lecture_schedules USING GIN (lecture_schedules_place gin_trgm_ops);

COMMIT;