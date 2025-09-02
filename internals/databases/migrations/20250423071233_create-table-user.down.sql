-- =========================================================
-- DOWN MIGRATION — ROLE SYSTEM (From Scratch, no users.role)
-- Membatalkan artefak role-system: views, functions, indexes, tables
-- TIDAK menghapus tabel users / users_profile / set_updated_at()
-- =========================================================
BEGIN;

-- ------------------------------
-- 1) DROP VIEWS (yang bergantung ke functions)
-- ------------------------------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.views
             WHERE table_schema = 'public' AND table_name = 'v_user_roles_resolved') THEN
    EXECUTE 'DROP VIEW v_user_roles_resolved';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.views
             WHERE table_schema = 'public' AND table_name = 'v_user_roles_compact') THEN
    EXECUTE 'DROP VIEW v_user_roles_compact';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.views
             WHERE table_schema = 'public' AND table_name = 'v_users_with_roles') THEN
    EXECUTE 'DROP VIEW v_users_with_roles';
  END IF;
END$$;

-- ------------------------------
-- 2) DROP FUNCTIONS (helper & claim)
--    Urutan dari yang tergantung → dependency minimal
-- ------------------------------
-- Claim JSON
DROP FUNCTION IF EXISTS fn_user_roles_claim(uuid);

-- Primary role resolvers
DROP FUNCTION IF EXISTS fn_user_primary_role_in_scope(uuid, uuid);
DROP FUNCTION IF EXISTS fn_user_primary_role_global(uuid);

-- Role aggregation
DROP FUNCTION IF EXISTS fn_user_roles_in_scope(uuid, uuid);
DROP FUNCTION IF EXISTS fn_user_global_roles(uuid);

-- Access / grant / revoke
DROP FUNCTION IF EXISTS fn_user_has_role_scope(uuid, text, uuid);
DROP FUNCTION IF EXISTS fn_revoke_role(uuid, text, uuid);
DROP FUNCTION IF EXISTS fn_grant_role(uuid, text, uuid, uuid);

-- Priority helper
DROP FUNCTION IF EXISTS fn_role_priority(text);

-- Catatan: fungsi set_updated_at() sengaja tidak di-drop
-- karena bisa dipakai objek lain.

-- ------------------------------
-- 3) DROP INDEXES di user_roles (opsional; akan ikut ter-drop saat table drop)
--    Ditaruh eksplisit untuk idempotency & kejelasan.
-- ------------------------------
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND n.nspname = 'public' AND c.relname = 'uq_user_roles_scope_alive'
  ) THEN
    EXECUTE 'DROP INDEX uq_user_roles_scope_alive';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND n.nspname = 'public' AND c.relname = 'idx_user_roles_user_scope_alive'
  ) THEN
    EXECUTE 'DROP INDEX idx_user_roles_user_scope_alive';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND n.nspname = 'public' AND c.relname = 'idx_user_roles_role_alive'
  ) THEN
    EXECUTE 'DROP INDEX idx_user_roles_role_alive';
  END IF;
END$$;

-- ------------------------------
-- 4) DROP TABLES (user_roles dulu, roles belakangan)
-- ------------------------------
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;

-- Selesai: users & users_profile dibiarkan tetap ada.
COMMIT;
