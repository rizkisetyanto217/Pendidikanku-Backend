-- =========================================================
-- UP MIGRATION — ROLE SYSTEM (From Scratch, no users.role)
-- - users, users_profile
-- - roles (master), user_roles (assignment)
-- - indexes, helper functions, views
-- - JSON claim builder for JWT
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS ----------
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

-- =========================================================
-- 3) ROLES (master)
-- =========================================================
CREATE TABLE IF NOT EXISTS roles (
  role_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  role_name VARCHAR(32) NOT NULL UNIQUE
);

-- Seed default roles
INSERT INTO roles(role_name) VALUES
  ('owner'),('admin'),('treasurer'),('dkm'),('teacher'),('author'),('student'),('user')
ON CONFLICT (role_name) DO NOTHING;

-- =========================================================
-- 4) USER_ROLES (assignment per user-per masjid, soft delete)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_roles (
  user_role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id      UUID NOT NULL REFERENCES roles(role_id) ON DELETE RESTRICT,
  masjid_id    UUID     REFERENCES masjids(masjid_id) ON DELETE CASCADE, -- NULL = global
  assigned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  assigned_by  UUID REFERENCES users(id) ON DELETE SET NULL,
  deleted_at   TIMESTAMPTZ
);

-- Unik hanya untuk baris "alive"
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_scope_alive
  ON user_roles (user_id, role_id, masjid_id)
  WHERE deleted_at IS NULL;

-- Akses cepat
CREATE INDEX IF NOT EXISTS idx_user_roles_user_scope_alive
  ON user_roles (user_id, masjid_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_roles_role_alive
  ON user_roles (role_id) WHERE deleted_at IS NULL;

-- =========================================================
-- 5) HELPER FUNCTIONS (grant / revoke / has-role)
-- =========================================================
-- Prioritas peran untuk sorting/primary
CREATE OR REPLACE FUNCTION fn_role_priority(p_role_name text)
RETURNS int AS $$
BEGIN
  RETURN CASE lower(p_role_name)
    WHEN 'owner'     THEN 100
    WHEN 'admin'     THEN 90
    WHEN 'treasurer' THEN 80
    WHEN 'dkm'       THEN 70
    WHEN 'teacher'   THEN 60
    WHEN 'author'    THEN 50
    WHEN 'student'   THEN 20
    WHEN 'user'      THEN 10
    ELSE 0
  END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Grant (idempotent; revive jika soft-deleted)
CREATE OR REPLACE FUNCTION fn_grant_role(
  p_user_id     uuid,
  p_role_name   text,
  p_masjid_id   uuid DEFAULT NULL,
  p_assigned_by uuid DEFAULT NULL
) RETURNS uuid AS $$
DECLARE
  v_role_id uuid;
  v_user_role_id uuid;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    INSERT INTO roles(role_name) VALUES (p_role_name) RETURNING role_id INTO v_role_id;
  END IF;

  UPDATE user_roles
     SET deleted_at = NULL,
         assigned_at = now(),
         assigned_by = COALESCE(p_assigned_by, assigned_by)
   WHERE user_id = p_user_id
     AND role_id = v_role_id
     AND ((masjid_id IS NULL AND p_masjid_id IS NULL) OR masjid_id = p_masjid_id)
     AND deleted_at IS NOT NULL
  RETURNING user_role_id INTO v_user_role_id;

  IF v_user_role_id IS NOT NULL THEN
    RETURN v_user_role_id;
  END IF;

  INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at, assigned_by)
  SELECT p_user_id, v_role_id, p_masjid_id, now(), p_assigned_by
  WHERE NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id   = p_user_id
      AND ur.role_id   = v_role_id
      AND ((ur.masjid_id IS NULL AND p_masjid_id IS NULL) OR ur.masjid_id = p_masjid_id)
      AND ur.deleted_at IS NULL
  )
  RETURNING user_role_id INTO v_user_role_id;

  IF v_user_role_id IS NULL THEN
    SELECT user_role_id INTO v_user_role_id FROM user_roles ur
    WHERE ur.user_id = p_user_id
      AND ur.role_id = v_role_id
      AND ((ur.masjid_id IS NULL AND p_masjid_id IS NULL) OR ur.masjid_id = p_masjid_id)
      AND ur.deleted_at IS NULL
    LIMIT 1;
  END IF;

  RETURN v_user_role_id;
END;
$$ LANGUAGE plpgsql;

-- Revoke (soft delete)
CREATE OR REPLACE FUNCTION fn_revoke_role(
  p_user_id   uuid,
  p_role_name text,
  p_masjid_id uuid DEFAULT NULL
) RETURNS boolean AS $$
DECLARE
  v_role_id uuid;
  v_affected int;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    RETURN FALSE;
  END IF;

  UPDATE user_roles
     SET deleted_at = now()
   WHERE user_id = p_user_id
     AND role_id = v_role_id
     AND ((masjid_id IS NULL AND p_masjid_id IS NULL) OR masjid_id = p_masjid_id)
     AND deleted_at IS NULL;

  GET DIAGNOSTICS v_affected = ROW_COUNT;
  RETURN v_affected > 0;
END;
$$ LANGUAGE plpgsql;

-- Cek role efektif (global OR scoped)
CREATE OR REPLACE FUNCTION fn_user_has_role_scope(
  p_user_id   uuid,
  p_role_name text,
  p_masjid_id uuid
) RETURNS boolean AS $$
DECLARE
  v_role_id uuid;
  v_exists boolean;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    RETURN FALSE;
  END IF;

  SELECT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = p_user_id
      AND ur.role_id = v_role_id
      AND ur.deleted_at IS NULL
      AND (ur.masjid_id = p_masjid_id OR ur.masjid_id IS NULL) -- global fallback
  ) INTO v_exists;

  RETURN v_exists;
END;
$$ LANGUAGE plpgsql;

-- =========================================================
-- 6) ROLE AGGREGATION HELPERS
-- =========================================================
-- Global roles (array) terurut by priority desc, name asc
CREATE OR REPLACE FUNCTION fn_user_global_roles(p_user_id uuid)
RETURNS text[] AS $$
  SELECT COALESCE(
    ARRAY(
      SELECT r.role_name
      FROM user_roles ur
      JOIN roles r ON r.role_id = ur.role_id
      WHERE ur.user_id = p_user_id
        AND ur.deleted_at IS NULL
        AND ur.masjid_id IS NULL
      GROUP BY r.role_name
      ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC
    ),
    ARRAY[]::text[]
  );
$$ LANGUAGE sql STABLE;

-- Roles di scope (include global fallback)
CREATE OR REPLACE FUNCTION fn_user_roles_in_scope(p_user_id uuid, p_masjid_id uuid)
RETURNS text[] AS $$
  WITH scope_roles AS (
    SELECT r.role_name
    FROM user_roles ur
    JOIN roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = p_user_id
      AND ur.deleted_at IS NULL
      AND (ur.masjid_id = p_masjid_id OR ur.masjid_id IS NULL)
    GROUP BY r.role_name
  )
  SELECT COALESCE(
    ARRAY(
      SELECT role_name
      FROM scope_roles
      ORDER BY fn_role_priority(role_name) DESC, role_name ASC
    ),
    ARRAY[]::text[]
  );
$$ LANGUAGE sql STABLE;

-- Primary role (single) global
CREATE OR REPLACE FUNCTION fn_user_primary_role_global(p_user_id uuid)
RETURNS text AS $$
  SELECT r.role_name
  FROM user_roles ur
  JOIN roles r ON r.role_id = ur.role_id
  WHERE ur.user_id = p_user_id
    AND ur.deleted_at IS NULL
    AND ur.masjid_id IS NULL
  GROUP BY r.role_name
  ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC
  LIMIT 1;
$$ LANGUAGE sql STABLE;

-- Primary role (single) per scope (dengan global fallback)
CREATE OR REPLACE FUNCTION fn_user_primary_role_in_scope(p_user_id uuid, p_masjid_id uuid)
RETURNS text AS $$
  WITH scope_roles AS (
    SELECT r.role_name
    FROM user_roles ur
    JOIN roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = p_user_id
      AND ur.deleted_at IS NULL
      AND (ur.masjid_id = p_masjid_id OR ur.masjid_id IS NULL)
    GROUP BY r.role_name
  )
  SELECT role_name
  FROM scope_roles
  ORDER BY fn_role_priority(role_name) DESC, role_name ASC
  LIMIT 1;
$$ LANGUAGE sql STABLE;

-- Claim JSON untuk JWT (ringkas & cepat)
-- Output:
-- {
--   "role_global": ["admin",...],
--   "masjid_roles": [{"masjid_id":"...","roles":["dkm","teacher"]}, ...]
-- }
CREATE OR REPLACE FUNCTION fn_user_roles_claim(p_user_id uuid)
RETURNS jsonb AS $$
  WITH
  g AS (
    SELECT fn_user_global_roles(p_user_id) AS roles_global
  ),
  p AS (
    SELECT ur.masjid_id,
           ARRAY_AGG(DISTINCT r.role_name
                     ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC) AS roles
    FROM user_roles ur
    JOIN roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = p_user_id
      AND ur.deleted_at IS NULL
      AND ur.masjid_id IS NOT NULL
    GROUP BY ur.masjid_id
  )
  SELECT jsonb_build_object(
    'role_global', COALESCE((SELECT to_jsonb(roles_global) FROM g), '[]'::jsonb),
    'masjid_roles', COALESCE(
      (SELECT jsonb_agg(jsonb_build_object('masjid_id', masjid_id, 'roles', to_jsonb(roles)))
       FROM p),
      '[]'::jsonb
    )
  );
$$ LANGUAGE sql STABLE;

-- =========================================================
-- 7) VIEWS (untuk admin & konsumsi aplikasi)
-- =========================================================
-- Resolved assignments (alive)
CREATE OR REPLACE VIEW v_user_roles_resolved AS
SELECT
  ur.user_role_id,
  ur.user_id,
  u.user_name,
  u.full_name,
  ur.role_id,
  r.role_name,
  ur.masjid_id,
  ur.assigned_at,
  ur.assigned_by
FROM user_roles ur
JOIN roles r ON r.role_id = ur.role_id
JOIN users u ON u.id = ur.user_id
WHERE ur.deleted_at IS NULL;

-- Compact per scope
CREATE OR REPLACE VIEW v_user_roles_compact AS
SELECT
  ur.user_id,
  ur.masjid_id,
  ARRAY_AGG(r.role_name ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC) AS roles
FROM user_roles ur
JOIN roles r ON r.role_id = ur.role_id
WHERE ur.deleted_at IS NULL
GROUP BY ur.user_id, ur.masjid_id;

-- Users + roles_global (ringkas)
CREATE OR REPLACE VIEW v_users_with_roles AS
SELECT
  u.id,
  u.user_name,
  u.full_name,
  u.email,
  u.is_active,
  fn_user_global_roles(u.id) AS roles_global
FROM users u;

COMMIT;
