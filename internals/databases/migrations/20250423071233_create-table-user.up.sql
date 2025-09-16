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


CREATE TABLE IF NOT EXISTS users_profile (
  user_profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_profile_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Identitas dasar
  user_profile_slug            VARCHAR(80),
  user_profile_donation_name   VARCHAR(50),
  user_profile_date_of_birth   DATE,
  user_profile_place_of_birth  VARCHAR(100),
  user_profile_gender          VARCHAR(10),
  user_profile_location        VARCHAR(100),
  user_profile_city            VARCHAR(100),
  user_profile_phone_number    VARCHAR(20),
  user_profile_bio             VARCHAR(300),

  -- Konten panjang & riwayat
  user_profile_biography_long  TEXT,
  user_profile_experience      TEXT,
  user_profile_certifications  TEXT,

  -- Sosial media
  user_profile_instagram_url   TEXT,
  user_profile_whatsapp_url    TEXT,
  user_profile_youtube_url     TEXT,
  user_profile_linkedin_url    TEXT,
  user_profile_github_url      TEXT,
  user_profile_telegram_username VARCHAR(50),

  -- Avatar (single file, 2-slot + retensi 30 hari)
  user_profile_avatar_url                   TEXT,
  user_profile_avatar_object_key            TEXT,
  user_profile_avatar_url_old               TEXT,
  user_profile_avatar_object_key_old        TEXT,
  user_profile_avatar_delete_pending_until  TIMESTAMPTZ,

  -- Privasi & verifikasi
  user_profile_is_public_profile BOOLEAN NOT NULL DEFAULT TRUE,
  user_profile_is_verified       BOOLEAN NOT NULL DEFAULT FALSE,
  user_profile_verified_at       TIMESTAMPTZ,
  user_profile_verified_by       UUID,

  -- Pendidikan & pekerjaan
  user_profile_education   TEXT,
  user_profile_company     TEXT,
  user_profile_position    TEXT,

  user_profile_interests   TEXT[],
  user_profile_skills      TEXT[],

  -- Audit
  user_profile_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_profile_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_profile_deleted_at  TIMESTAMPTZ,

  -- Constraints
  CONSTRAINT uq_user_profile_user_id UNIQUE (user_profile_user_id),
  CONSTRAINT ck_user_profile_gender CHECK (
    user_profile_gender IS NULL OR user_profile_gender IN ('male','female')
  ),
  CONSTRAINT ck_user_profile_slug_format CHECK (
    user_profile_slug IS NULL OR user_profile_slug ~ '^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$'
  )
);

-- Indexes (pakai prefix baru)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_profile_slug_alive
  ON users_profile (user_profile_slug)
  WHERE user_profile_deleted_at IS NULL AND user_profile_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_user_id_alive
  ON users_profile(user_profile_user_id) WHERE user_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_gender
  ON users_profile(user_profile_gender) WHERE user_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_phone
  ON users_profile(user_profile_phone_number) WHERE user_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_location
  ON users_profile(user_profile_location) WHERE user_profile_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_telegram
  ON users_profile(user_profile_telegram_username)
  WHERE user_profile_deleted_at IS NULL AND user_profile_telegram_username IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_profile_avatar_purge_due
  ON users_profile(user_profile_avatar_delete_pending_until)
  WHERE user_profile_avatar_object_key_old IS NOT NULL;

COMMIT;
