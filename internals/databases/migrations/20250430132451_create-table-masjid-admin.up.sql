-- UUID helper
CREATE EXTENSION IF NOT EXISTS pgcrypto;


-- =========================
-- 1) masjid_admins
-- =========================
CREATE TABLE IF NOT EXISTS masjid_admins (
  masjid_admin_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_admin_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_admin_user_id   UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,
  masjid_admin_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- timestamps explicit
  masjid_admin_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_admin_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_admin_deleted_at TIMESTAMPTZ NULL
);

-- Admin yang sama tidak boleh dobel pada masjid yang sama (hanya baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjid_admins_masjid_user_alive
  ON masjid_admins (masjid_admin_masjid_id, masjid_admin_user_id)
  WHERE masjid_admin_deleted_at IS NULL;

-- Pola query umum (hanya baris hidup):
-- by user & active
CREATE INDEX IF NOT EXISTS idx_masjid_admins_user_active_alive
  ON masjid_admins (masjid_admin_user_id)
  WHERE masjid_admin_is_active = TRUE AND masjid_admin_deleted_at IS NULL;

-- by masjid & active
CREATE INDEX IF NOT EXISTS idx_masjid_admins_masjid_active_alive
  ON masjid_admins (masjid_admin_masjid_id)
  WHERE masjid_admin_is_active = TRUE AND masjid_admin_deleted_at IS NULL;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at_masjid_admins() RETURNS trigger AS $$
BEGIN
  NEW.masjid_admin_updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_admins ON masjid_admins;
CREATE TRIGGER trg_set_updated_at_masjid_admins
BEFORE UPDATE ON masjid_admins
FOR EACH ROW EXECUTE FUNCTION set_updated_at_masjid_admins();

-- =========================
-- 2) masjid_teachers
-- =========================
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teacher_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_teacher_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teacher_user_id    UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,

  masjid_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_deleted_at TIMESTAMPTZ NULL
);

-- Guru yang sama tidak boleh dobel pada masjid yang sama (hanya baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjid_teachers_masjid_user_alive
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Lookup semua masjid yang dia ajar (baris hidup)
CREATE INDEX IF NOT EXISTS idx_masjid_teachers_user_id_alive
  ON masjid_teachers (masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- (opsional kalau sering cari per masjid)
-- CREATE INDEX IF NOT EXISTS idx_masjid_teachers_masjid_id_alive
--   ON masjid_teachers (masjid_teacher_masjid_id)
--   WHERE masjid_teacher_deleted_at IS NULL;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at_masjid_teachers() RETURNS trigger AS $$
BEGIN
  NEW.masjid_teacher_updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_teachers ON masjid_teachers;
CREATE TRIGGER trg_set_updated_at_masjid_teachers
BEFORE UPDATE ON masjid_teachers
FOR EACH ROW EXECUTE FUNCTION set_updated_at_masjid_teachers();
