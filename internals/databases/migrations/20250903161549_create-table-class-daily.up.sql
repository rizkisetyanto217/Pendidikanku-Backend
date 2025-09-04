BEGIN;

-- untuk gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =========================================================
-- CLASS_DAILY (minimal, tanpa relasi ke class_schedules/CAS)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_daily (
  class_daily_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope tenant & tanggal
  class_daily_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_daily_date      DATE NOT NULL,

  -- section wajib (persist agar stabil walau sumber berubah)
  class_daily_section_id UUID NOT NULL REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- flag aktif
  class_daily_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  -- helper (1=Senin .. 7=Minggu)
  class_daily_day_of_week INT GENERATED ALWAYS AS ((EXTRACT(ISODOW FROM class_daily_date))::INT) STORED,

  -- timestamps dikelola aplikasi/GORM (eksplisit)
  class_daily_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_daily_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_daily_deleted_at TIMESTAMPTZ
);

-- =========================
-- Indexing & Uniques (alive)
-- =========================

-- unik: satu daily per (masjid, section, date) saat belum soft-deleted
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_daily_masjid_section_date
  ON class_daily (class_daily_masjid_id, class_daily_section_id, class_daily_date)
  WHERE class_daily_deleted_at IS NULL;

-- indeks bantu navigasi
CREATE INDEX IF NOT EXISTS idx_class_daily_masjid_date
  ON class_daily (class_daily_masjid_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_daily_section_date
  ON class_daily (class_daily_section_id, class_daily_date DESC)
  WHERE class_daily_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_daily_active
  ON class_daily (class_daily_is_active)
  WHERE class_daily_is_active AND class_daily_deleted_at IS NULL;

COMMIT;
