-- Extensions (aman dijalankan berulang)
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram index
CREATE EXTENSION IF NOT EXISTS citext;    -- case-insensitive text

-- =========================
-- TABEL USERS
-- =========================
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_name VARCHAR(50) NOT NULL CHECK (LENGTH(user_name) BETWEEN 3 AND 50),
  full_name VARCHAR(100) CHECK (LENGTH(full_name) BETWEEN 3 AND 100), -- ✅ pindah ke sini
  email CITEXT UNIQUE NOT NULL,                                      -- case-insensitive unique
  password VARCHAR(250),
  google_id VARCHAR(255) UNIQUE,
  role VARCHAR(20) NOT NULL DEFAULT 'user'
      CHECK (role IN ('owner','user','teacher','treasurer','admin','dkm','author','student')),
  security_question TEXT NOT NULL,
  security_answer   VARCHAR(255) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indeks btree dasar
CREATE INDEX IF NOT EXISTS idx_users_user_name ON users(user_name);
CREATE INDEX IF NOT EXISTS idx_users_full_name ON users(full_name);
CREATE INDEX IF NOT EXISTS idx_users_role      ON users(role);

-- Trigram untuk substring search
CREATE INDEX IF NOT EXISTS idx_users_user_name_trgm
  ON users USING gin (user_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_users_full_name_trgm
  ON users USING gin (full_name gin_trgm_ops);

-- Prefix case-insensitive
CREATE INDEX IF NOT EXISTS idx_users_user_name_lower
  ON users (LOWER(user_name));

CREATE INDEX IF NOT EXISTS idx_users_full_name_lower
  ON users (LOWER(full_name));

-- Full Text Search untuk user_name + full_name
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS user_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_name,'')), 'A') ||
    setweight(to_tsvector('simple', coalesce(full_name,'')), 'B')
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_users_user_search
  ON users USING gin (user_search);

-- Trigger updated_at (khusus tabel users)
CREATE OR REPLACE FUNCTION set_updated_at_users() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
CREATE TRIGGER trg_set_updated_at_users
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at_users();

-- =========================
-- TABEL USERS_PROFILE (1:1)
-- =========================
CREATE TABLE IF NOT EXISTS users_profile (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  donation_name VARCHAR(50),
  date_of_birth DATE,
  gender        VARCHAR(10) CHECK (gender IN ('male','female')),
  phone_number  VARCHAR(20),
  bio           VARCHAR(300),
  father_name   VARCHAR(50),
  mother_name   VARCHAR(50),
  location      VARCHAR(50),
  occupation    VARCHAR(20),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP,
  deleted_at TIMESTAMP
);

-- Enforce 1-to-1
ALTER TABLE users_profile
  ADD CONSTRAINT users_profile_user_id_key UNIQUE (user_id);

-- Index cepat profile aktif
CREATE INDEX IF NOT EXISTS idx_users_profile_user_id_alive
  ON users_profile(user_id) WHERE deleted_at IS NULL;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at_users_profile() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile ON users_profile;
CREATE TRIGGER trg_set_updated_at_users_profile
BEFORE UPDATE ON users_profile
FOR EACH ROW EXECUTE FUNCTION set_updated_at_users_profile();
