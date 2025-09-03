-- =========================================================
-- UP #1 — USERS & USERS_PROFILE (tanpa kolom role)
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS (sekali saja di awal project) ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- opsional utk kombinasi tertentu

-- ---------- SHARED: updated_at trigger ----------
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =========================================================
-- 1) USERS (tanpa kolom role)
-- =========================================================
CREATE TABLE IF NOT EXISTS users (
  id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_name          VARCHAR(50)  NOT NULL,
  full_name          VARCHAR(100),
  email              CITEXT       NOT NULL,
  password           VARCHAR(250),
  google_id          VARCHAR(255),
  security_question  TEXT         NOT NULL,
  security_answer    VARCHAR(255) NOT NULL,
  is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
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

DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
CREATE TRIGGER trg_set_updated_at_users
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- =========================================================
-- 2) USERS_PROFILE
-- =========================================================
CREATE TABLE IF NOT EXISTS users_profile (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  donation_name VARCHAR(50),
  photo_url     VARCHAR(255),
  photo_trash_url TEXT,
  photo_delete_pending_until TIMESTAMPTZ,
  date_of_birth DATE,
  gender        VARCHAR(10),
  location      VARCHAR(100),
  occupation    VARCHAR(50),
  phone_number  VARCHAR(20),
  bio           VARCHAR(300),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at    TIMESTAMPTZ,

  CONSTRAINT uq_users_profile_user_id UNIQUE (user_id),
  CONSTRAINT ck_users_profile_gender CHECK (gender IS NULL OR gender IN ('male','female'))
);

CREATE INDEX IF NOT EXISTS idx_users_profile_user_id_alive
  ON users_profile(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_gender
  ON users_profile(gender) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_phone
  ON users_profile(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_location
  ON users_profile(location) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;
CREATE TRIGGER trg_set_updated_at_users_profile
BEFORE UPDATE ON users_profile
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
