-- =========================================================
-- UP — USERS & USERS_PROFILE (from scratch, lengkap + idempotent)
-- Fokus: USERS (skinny + google_id + email_verified_at),
--        USERS_PROFILE (kolom eksplisit: users_profile_*)
--        Termasuk index & FTS.
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS (aman diulang) ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- opsional utk kombinasi tertentu

-- =========================================================
-- 1) USERS (skinny + google_id; TANPA security_question/answer)
-- =========================================================
CREATE TABLE IF NOT EXISTS users (
  id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_name          VARCHAR(50)  NOT NULL,
  full_name          VARCHAR(100),
  email              CITEXT       NOT NULL,
  password           VARCHAR(250),
  google_id          VARCHAR(255),
  is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
  email_verified_at  TIMESTAMPTZ,
  created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
  deleted_at         TIMESTAMPTZ,

  CONSTRAINT uq_users_email     UNIQUE (email),
  CONSTRAINT uq_users_google_id UNIQUE (google_id),
  CONSTRAINT ck_users_user_name_len CHECK (char_length(user_name) BETWEEN 3 AND 50),
  CONSTRAINT ck_users_full_name_len CHECK (full_name IS NULL OR char_length(full_name) BETWEEN 3 AND 100)
);

-- Indexes dasar & pencarian
CREATE INDEX IF NOT EXISTS idx_users_user_name        ON users(user_name);
CREATE INDEX IF NOT EXISTS idx_users_full_name        ON users(full_name);
CREATE INDEX IF NOT EXISTS idx_users_is_active        ON users(is_active) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_user_name_trgm   ON users USING gin (user_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_full_name_trgm   ON users USING gin (full_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_user_name_lower  ON users (lower(user_name));
CREATE INDEX IF NOT EXISTS idx_users_full_name_lower  ON users (lower(full_name));

-- Full Text Search (user_search)
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS user_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_name, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(full_name, '')), 'B')
  ) STORED;
CREATE INDEX IF NOT EXISTS idx_users_user_search ON users USING gin (user_search);



-- =========================================================
-- TABEL: users_profile
-- =========================================================
CREATE TABLE IF NOT EXISTS users_profile (
  users_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  users_profile_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Identitas dasar
  users_profile_donation_name   VARCHAR(50),
  users_profile_date_of_birth   DATE,
  user_profile_place_of_birth   VARCHAR(100),
  users_profile_gender          VARCHAR(10),
  users_profile_location        VARCHAR(100),
  users_profile_city            VARCHAR(100),
  users_profile_phone_number    VARCHAR(20),
  users_profile_bio             VARCHAR(300),

  -- Konten panjang & riwayat
  users_profile_biography_long  TEXT,
  users_profile_experience      TEXT,
  users_profile_certifications  TEXT,

  -- Sosial media utama
  users_profile_instagram_url   TEXT,
  users_profile_whatsapp_url    TEXT,
  users_profile_youtube_url    TEXT,
  users_profile_linkedin_url      TEXT,
  users_profile_github_url        TEXT,

  -- Privasi
  users_profile_is_public_profile BOOLEAN NOT NULL DEFAULT TRUE,

  -- Verifikasi
  users_profile_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  users_profile_verified_at    TIMESTAMPTZ,
  users_profile_verified_by    UUID,

  -- Pendidikan & pekerjaan
  users_profile_education   TEXT,
  users_profile_company     TEXT,
  users_profile_position    TEXT,

  users_profile_interests TEXT[],
  users_profile_skills TEXT[],

  -- Audit record standar
  users_profile_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_profile_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_profile_deleted_at  TIMESTAMPTZ,

  -- Constraints
  CONSTRAINT uq_users_profile_user_id UNIQUE (users_profile_user_id),
  CONSTRAINT ck_users_profile_gender CHECK (
    users_profile_gender IS NULL OR users_profile_gender IN ('male','female')
  ),
  CONSTRAINT ck_users_profile_slug_format CHECK (
    users_profile_slug IS NULL OR users_profile_slug ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  )
);

-- =========================================================
-- INDEXES
-- =========================================================

-- Slug unik (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_profile_slug_alive
  ON users_profile (users_profile_slug)
  WHERE users_profile_deleted_at IS NULL AND users_profile_slug IS NOT NULL;

-- Index bantu umum
CREATE INDEX IF NOT EXISTS idx_users_profile_user_id_alive
  ON users_profile(users_profile_user_id) WHERE users_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_profile_gender
  ON users_profile(users_profile_gender) WHERE users_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_profile_phone
  ON users_profile(users_profile_phone_number) WHERE users_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_profile_location
  ON users_profile(users_profile_location) WHERE users_profile_deleted_at IS NULL;

-- Telegram lookup
CREATE INDEX IF NOT EXISTS idx_users_profile_telegram
  ON users_profile(users_profile_telegram_username)
  WHERE users_profile_deleted_at IS NULL AND users_profile_telegram_username IS NOT NULL;

COMMIT;
