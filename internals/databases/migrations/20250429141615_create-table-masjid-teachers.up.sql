BEGIN;

-- UUID helper
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =========================
-- masjid_teachers (minimal)
-- =========================
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teacher_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  masjid_teacher_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teacher_user_id   UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,

  masjid_teacher_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_deleted_at TIMESTAMPTZ
);

-- Unik: 1 user per masjid (hanya baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_masjid_user_alive
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Lookup cepat (baris hidup)
CREATE INDEX IF NOT EXISTS idx_mtj_user_alive
  ON masjid_teachers (masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_masjid_alive
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL;

COMMIT;
