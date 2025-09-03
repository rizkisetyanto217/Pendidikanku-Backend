-- =========================================================
-- DOWN #2 — drop views, helpers, tables
-- =========================================================
BEGIN;

-- Views
DROP VIEW IF EXISTS v_users_with_roles;
DROP VIEW IF EXISTS v_user_roles_compact;
DROP VIEW IF EXISTS v_user_roles_resolved;

-- Functions
DROP FUNCTION IF EXISTS fn_user_roles_claim(uuid);
DROP FUNCTION IF EXISTS fn_user_primary_role_in_scope(uuid, uuid);
DROP FUNCTION IF EXISTS fn_user_primary_role_global(uuid);
DROP FUNCTION IF EXISTS fn_user_roles_in_scope(uuid, uuid);
DROP FUNCTION IF EXISTS fn_user_global_roles(uuid);
DROP FUNCTION IF EXISTS fn_user_has_role_scope(uuid, text, uuid);
DROP FUNCTION IF EXISTS fn_revoke_role(uuid, text, uuid);
DROP FUNCTION IF EXISTS fn_grant_role(uuid, text, uuid, uuid);
DROP FUNCTION IF EXISTS fn_role_priority(text);

-- Tables
DROP INDEX IF EXISTS idx_user_roles_role_alive;
DROP INDEX IF EXISTS idx_user_roles_user_scope_alive;
DROP INDEX IF EXISTS uq_user_roles_scope_alive;
DROP TABLE IF EXISTS user_roles;

DROP TABLE IF EXISTS roles;

COMMIT;
