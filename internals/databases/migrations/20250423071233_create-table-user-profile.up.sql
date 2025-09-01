-- =========================================================
-- USERS & USERS_PROFILE (optimized + indexing)
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- useful utk GIN/GiST kombinasi

/* -------------------------------------------------------
   USERS
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS users (
  id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_name          VARCHAR(50)  NOT NULL,
  full_name          VARCHAR(100),
  email              CITEXT       NOT NULL,
  password           VARCHAR(250),
  google_id          VARCHAR(255),
  role               VARCHAR(20)  NOT NULL DEFAULT 'user',
  security_question  TEXT         NOT NULL,
  security_answer    VARCHAR(255) NOT NULL,
  is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
  deleted_at         TIMESTAMPTZ,

  -- Constraints
  CONSTRAINT uq_users_email     UNIQUE (email),
  CONSTRAINT uq_users_google_id UNIQUE (google_id),
  CONSTRAINT ck_users_user_name_len CHECK (char_length(user_name) BETWEEN 3 AND 50),
  CONSTRAINT ck_users_full_name_len CHECK (full_name IS NULL OR char_length(full_name) BETWEEN 3 AND 100),
  CONSTRAINT ck_users_role CHECK (role IN ('owner','user','teacher','treasurer','admin','dkm','author','student'))
);

-- Indexes dasar
CREATE INDEX IF NOT EXISTS idx_users_user_name        ON users(user_name);
CREATE INDEX IF NOT EXISTS idx_users_full_name        ON users(full_name);
CREATE INDEX IF NOT EXISTS idx_users_role             ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active        ON users(is_active) WHERE deleted_at IS NULL;

-- Search (trigram + prefix)
CREATE INDEX IF NOT EXISTS idx_users_user_name_trgm   ON users USING gin (user_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_full_name_trgm   ON users USING gin (full_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_user_name_lower  ON users (lower(user_name));
CREATE INDEX IF NOT EXISTS idx_users_full_name_lower  ON users (lower(full_name));

-- Full Text Search
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS user_search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_name, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(full_name, '')), 'B')
  ) STORED;
CREATE INDEX IF NOT EXISTS idx_users_user_search ON users USING gin (user_search);

-- Updated_at trigger
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_users ON users;
CREATE TRIGGER trg_set_updated_at_users
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();


/* -------------------------------------------------------
   USERS_PROFILE (inti umum)
------------------------------------------------------- */
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
  occupation     VARCHAR(50),
  phone_number  VARCHAR(20),
  bio           VARCHAR(300),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ,
  deleted_at    TIMESTAMPTZ,

  CONSTRAINT uq_users_profile_user_id UNIQUE (user_id),
  CONSTRAINT ck_users_profile_gender CHECK (gender IS NULL OR gender IN ('male','female'))
);

-- Indexing profile
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


/* -------------------------------------------------------
   USERS_PROFILE_FORMAL (tambahan sekolah formal)
------------------------------------------------------- */

-- Create table (fresh) dengan kolom phone sekalian
CREATE TABLE IF NOT EXISTS users_profile_formal (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  father_name    VARCHAR(50),
  father_phone   VARCHAR(20),   -- ✅ baru
  mother_name    VARCHAR(50),
  mother_phone   VARCHAR(20),   -- ✅ baru

  guardian       VARCHAR(50),
  guardian_phone VARCHAR(20),   -- ✅ baru

  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ,
  deleted_at     TIMESTAMPTZ,

  CONSTRAINT uq_users_profile_formal_user UNIQUE (user_id)
);

-- Jika tabel sudah ada dari versi lama, tambahkan kolom phone secara aman
ALTER TABLE users_profile_formal
  ADD COLUMN IF NOT EXISTS father_phone   VARCHAR(20),
  ADD COLUMN IF NOT EXISTS mother_phone   VARCHAR(20),
  ADD COLUMN IF NOT EXISTS guardian_phone VARCHAR(20);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_user_alive
  ON users_profile_formal(user_id) WHERE deleted_at IS NULL;


-- (Opsional) Index nomor HP supaya pencarian cepat
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_father_phone
  ON users_profile_formal(father_phone) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_profile_formal_mother_phone
  ON users_profile_formal(mother_phone) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_profile_formal_guardian_phone
  ON users_profile_formal(guardian_phone) WHERE deleted_at IS NULL;

-- Trigger updated_at (butuh fungsi set_updated_at() sudah ada)
DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_formal ON users_profile_formal;
CREATE TRIGGER trg_set_updated_at_users_profile_formal
BEFORE UPDATE ON users_profile_formal
FOR EACH ROW EXECUTE FUNCTION set_updated_at();



/* -------------------------------------------------------
   USERS_PROFILE_DOCUMENTS (dokumen/file upload)
------------------------------------------------------- */
CREATE TABLE IF NOT EXISTS users_profile_documents (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  doc_type       VARCHAR(50) NOT NULL,
  file_url       TEXT NOT NULL,
  file_trash_url TEXT,
  file_delete_pending_until TIMESTAMPTZ,
  uploaded_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at     TIMESTAMPTZ,

  CONSTRAINT uq_user_doc_type UNIQUE (user_id, doc_type)
);

-- Indexing documents
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_user_alive
  ON users_profile_documents(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_doctype
  ON users_profile_documents(doc_type) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_set_updated_at_users_profile_documents ON users_profile_documents;
CREATE TRIGGER trg_set_updated_at_users_profile_documents
BEFORE UPDATE ON users_profile_documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;


