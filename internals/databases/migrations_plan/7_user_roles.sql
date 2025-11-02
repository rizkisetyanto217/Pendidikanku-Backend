-- =========================================================
-- UP #2 — ROLES & USER_ROLES (assignment + helpers + views)
-- Dependensi: tabel users & schools sudah ada (dibuat di UP #1)
-- =========================================================
BEGIN;

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
-- 4) USER_ROLES (assignment per user-per school, soft delete)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_roles (
  user_role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id      UUID NOT NULL REFERENCES roles(role_id) ON DELETE RESTRICT,
  school_id    UUID     REFERENCES schools(school_id) ON DELETE CASCADE, -- NULL = global
  assigned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  assigned_by  UUID REFERENCES users(id) ON DELETE SET NULL,
  deleted_at   TIMESTAMPTZ
);

-- Unik hanya untuk baris "alive"
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_scope_alive
  ON user_roles (user_id, role_id, school_id)
  WHERE deleted_at IS NULL;

-- Akses cepat
CREATE INDEX IF NOT EXISTS idx_user_roles_user_scope_alive
  ON user_roles (user_id, school_id) WHERE deleted_at IS NULL;

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
  p_school_id   uuid DEFAULT NULL,
  p_assigned_by uuid DEFAULT NULL
) RETURNS uuid AS $$
DECLARE
  v_role_id      uuid;
  v_user_role_id uuid;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    INSERT INTO roles(role_name) VALUES (p_role_name) RETURNING role_id INTO v_role_id;
  END IF;

  -- Revive bila ada yang soft-deleted pada scope sama
  UPDATE user_roles
     SET deleted_at = NULL,
         assigned_at = now(),
         assigned_by = COALESCE(p_assigned_by, assigned_by)
   WHERE user_id = p_user_id
     AND role_id = v_role_id
     AND ((school_id IS NULL AND p_school_id IS NULL) OR school_id = p_school_id)
     AND deleted_at IS NOT NULL
  RETURNING user_role_id INTO v_user_role_id;

  IF v_user_role_id IS NOT NULL THEN
    RETURN v_user_role_id;
  END IF;

  -- Insert baru jika belum ada baris alive
  INSERT INTO user_roles(user_id, role_id, school_id, assigned_at, assigned_by)
  SELECT p_user_id, v_role_id, p_school_id, now(), p_assigned_by
  WHERE NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id   = p_user_id
      AND ur.role_id   = v_role_id
      AND ((ur.school_id IS NULL AND p_school_id IS NULL) OR ur.school_id = p_school_id)
      AND ur.deleted_at IS NULL
  )
  RETURNING user_role_id INTO v_user_role_id;

  -- Jika ternyata sudah ada baris alive (race), ambil id-nya
  IF v_user_role_id IS NULL THEN
    SELECT user_role_id INTO v_user_role_id FROM user_roles ur
    WHERE ur.user_id   = p_user_id
      AND ur.role_id   = v_role_id
      AND ((ur.school_id IS NULL AND p_school_id IS NULL) OR ur.school_id = p_school_id)
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
  p_school_id uuid DEFAULT NULL
) RETURNS boolean AS $$
DECLARE
  v_role_id  uuid;
  v_affected int;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    RETURN FALSE;
  END IF;

  UPDATE user_roles
     SET deleted_at = now()
   WHERE user_id   = p_user_id
     AND role_id   = v_role_id
     AND ((school_id IS NULL AND p_school_id IS NULL) OR school_id = p_school_id)
     AND deleted_at IS NULL;

  GET DIAGNOSTICS v_affected = ROW_COUNT;
  RETURN v_affected > 0;
END;
$$ LANGUAGE plpgsql;

-- Cek role efektif (global OR scoped)
CREATE OR REPLACE FUNCTION fn_user_has_role_scope(
  p_user_id   uuid,
  p_role_name text,
  p_school_id uuid
) RETURNS boolean AS $$
DECLARE
  v_role_id uuid;
  v_exists  boolean;
BEGIN
  SELECT role_id INTO v_role_id FROM roles WHERE role_name = p_role_name;
  IF v_role_id IS NULL THEN
    RETURN FALSE;
  END IF;

  SELECT EXISTS (
    SELECT 1
    FROM user_roles ur
    WHERE ur.user_id   = p_user_id
      AND ur.role_id   = v_role_id
      AND ur.deleted_at IS NULL
      AND (ur.school_id = p_school_id OR ur.school_id IS NULL) -- global fallback
  )
  INTO v_exists;

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
        AND ur.school_id IS NULL
      GROUP BY r.role_name
      ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC
    ),
    ARRAY[]::text[]
  );
$$ LANGUAGE sql STABLE;

-- Roles di scope (include global fallback) → pakai GROUP BY (hindari DISTINCT+ORDER BY)
CREATE OR REPLACE FUNCTION fn_user_roles_in_scope(p_user_id uuid, p_school_id uuid)
RETURNS text[] AS $$
  SELECT COALESCE(
    ARRAY(
      SELECT r.role_name
      FROM user_roles ur
      JOIN roles r ON r.role_id = ur.role_id
      WHERE ur.user_id = p_user_id
        AND ur.deleted_at IS NULL
        AND (ur.school_id = p_school_id OR ur.school_id IS NULL)
      GROUP BY r.role_name
      ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC
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
    AND ur.school_id IS NULL
  GROUP BY r.role_name
  ORDER BY fn_role_priority(r.role_name) DESC, r.role_name ASC
  LIMIT 1;
$$ LANGUAGE sql STABLE;

-- Primary role (single) per scope (dengan global fallback) → subquery pakai GROUP BY
CREATE OR REPLACE FUNCTION fn_user_primary_role_in_scope(p_user_id uuid, p_school_id uuid)
RETURNS text AS $$
  SELECT role_name
  FROM (
    SELECT r.role_name
    FROM user_roles ur
    JOIN roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = p_user_id
      AND ur.deleted_at IS NULL
      AND (ur.school_id = p_school_id OR ur.school_id IS NULL)
    GROUP BY r.role_name
  ) AS t
  ORDER BY fn_role_priority(role_name) DESC, role_name ASC
  LIMIT 1;
$$ LANGUAGE sql STABLE;

-- Claim JSON untuk JWT → array subselect pakai GROUP BY
CREATE OR REPLACE FUNCTION fn_user_roles_claim(p_user_id uuid)
RETURNS jsonb AS $$
  WITH
  g AS (
    SELECT fn_user_global_roles(p_user_id) AS roles_global
  ),
  p AS (
    SELECT ur.school_id,
           ARRAY(
             SELECT r2.role_name
             FROM user_roles ur2
             JOIN roles r2 ON r2.role_id = ur2.role_id
             WHERE ur2.user_id = p_user_id
               AND ur2.deleted_at IS NULL
               AND ur2.school_id = ur.school_id
             GROUP BY r2.role_name
             ORDER BY fn_role_priority(r2.role_name) DESC, r2.role_name ASC
           ) AS roles
    FROM user_roles ur
    WHERE ur.user_id = p_user_id
      AND ur.deleted_at IS NULL
      AND ur.school_id IS NOT NULL
    GROUP BY ur.school_id
  )
  SELECT jsonb_build_object(
    'role_global', COALESCE((SELECT to_jsonb(roles_global) FROM g), '[]'::jsonb),
    'school_roles', COALESCE(
      (SELECT jsonb_agg(jsonb_build_object('school_id', school_id, 'roles', to_jsonb(roles)))
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
  ur.school_id,
  ur.assigned_at,
  ur.assigned_by
FROM user_roles ur
JOIN roles r ON r.role_id = ur.role_id
JOIN users u ON u.id = ur.user_id
WHERE ur.deleted_at IS NULL;

-- Compact per scope → array subselect pakai GROUP BY
CREATE OR REPLACE VIEW v_user_roles_compact AS
SELECT
  ur.user_id,
  ur.school_id,
  ARRAY(
    SELECT r2.role_name
    FROM user_roles ur2
    JOIN roles r2 ON r2.role_id = ur2.role_id
    WHERE ur2.user_id = ur.user_id
      AND ur2.deleted_at IS NULL
      AND COALESCE(ur2.school_id, '00000000-0000-0000-0000-000000000000'::uuid) =
          COALESCE(ur.school_id, '00000000-0000-0000-0000-000000000000'::uuid)
    GROUP BY r2.role_name
    ORDER BY fn_role_priority(r2.role_name) DESC, r2.role_name ASC
  ) AS roles
FROM user_roles ur
WHERE ur.deleted_at IS NULL
GROUP BY ur.user_id, ur.school_id;

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
