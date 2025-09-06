BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- untuk GIN trigram (opsional ILIKE)

-- =========================================================
-- BERSIHKAN DATA & SKEMA LAMA
-- =========================================================
-- Drop total agar benar-benar fresh (akan ikut menjatuhkan FK yang mereferensikan table ini)
DROP TABLE IF EXISTS class_sections CASCADE;

-- =========================================================
-- CREATE: class_sections (versi sederhana, tanpa trigger)
-- =========================================================
CREATE TABLE class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi inti
  class_sections_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_sections_class_id      UUID NOT NULL,
  class_sections_teacher_id    UUID,
  class_sections_class_room_id UUID,

  -- Identitas
  class_sections_slug  VARCHAR(160) NOT NULL,
  class_sections_name  VARCHAR(100) NOT NULL,
  class_sections_code  VARCHAR(50),

  -- Jadwal simple (teks bebas, contoh: "Jumat 19:00–21:00")
  class_sections_schedule VARCHAR(200),

  -- Kapasitas & counter (dikelola backend)
  class_sections_capacity       INT,
  class_sections_total_students INT NOT NULL DEFAULT 0,

  -- Group link (cukup URL saja)
  class_sections_group_url TEXT,

  -- Status & audit
  class_sections_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_sections_deleted_at TIMESTAMPTZ,

  -- ======================
  -- CHECK guards ringan
  -- ======================
  CONSTRAINT ck_sections_capacity_nonneg
    CHECK (class_sections_capacity IS NULL OR class_sections_capacity >= 0),
  CONSTRAINT ck_sections_total_nonneg
    CHECK (class_sections_total_students >= 0),
  CONSTRAINT ck_sections_total_le_capacity
    CHECK (class_sections_capacity IS NULL OR class_sections_total_students <= class_sections_capacity),
  CONSTRAINT ck_sections_group_url_scheme
    CHECK (class_sections_group_url IS NULL OR class_sections_group_url ~* '^(https?)://'),

  -- ======================
  -- FK KOMPOSIT tenant-safe
  -- ======================
  CONSTRAINT fk_sections_class_same_masjid
    FOREIGN KEY (class_sections_class_id, class_sections_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_sections_teacher_same_masjid
    FOREIGN KEY (class_sections_teacher_id, class_sections_masjid_id)
    REFERENCES masjid_teachers (masjid_teacher_id, masjid_teacher_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  CONSTRAINT fk_sections_room_same_masjid
    FOREIGN KEY (class_sections_class_room_id, class_sections_masjid_id)
    REFERENCES class_rooms (class_room_id, class_rooms_masjid_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Pair unik untuk join multi-tenant aman
  CONSTRAINT uq_class_sections_id_masjid
    UNIQUE (class_sections_id, class_sections_masjid_id)
);

-- =========================================================
-- INDEXES & UNIQUES
-- =========================================================

-- Unik: slug per masjid (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_slug_per_masjid_alive
  ON class_sections (class_sections_masjid_id, LOWER(class_sections_slug))
  WHERE class_sections_deleted_at IS NULL;

-- Unik: nama section per class (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name_alive
  ON class_sections (class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

-- (Opsional) Unik: code per class (alive only, case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_code_per_class_alive
  ON class_sections (class_sections_class_id, LOWER(class_sections_code))
  WHERE class_sections_deleted_at IS NULL
    AND class_sections_code IS NOT NULL;

-- Lookup dasar FKs
CREATE INDEX IF NOT EXISTS idx_sections_masjid     ON class_sections (class_sections_masjid_id);
CREATE INDEX IF NOT EXISTS idx_sections_class      ON class_sections (class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_sections_teacher    ON class_sections (class_sections_teacher_id);
CREATE INDEX IF NOT EXISTS idx_sections_class_room ON class_sections (class_sections_class_room_id);

-- Listing umum: tenant-first + filter + pagination stabil
CREATE INDEX IF NOT EXISTS ix_sections_masjid_active_created
  ON class_sections (class_sections_masjid_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sections_class_active_created
  ON class_sections (class_sections_class_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

-- (Opsional) Listing by teacher/room
CREATE INDEX IF NOT EXISTS ix_sections_teacher_active_created
  ON class_sections (class_sections_teacher_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_sections_room_active_created
  ON class_sections (class_sections_class_room_id, class_sections_is_active, class_sections_created_at DESC)
  WHERE class_sections_deleted_at IS NULL;

-- Pencarian teks cepat (ILIKE) pada name/slug (opsional; butuh pg_trgm)
CREATE INDEX IF NOT EXISTS gin_sections_name_trgm_alive
  ON class_sections USING GIN (LOWER(class_sections_name) gin_trgm_ops)
  WHERE class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_sections_slug_trgm_alive
  ON class_sections USING GIN (LOWER(class_sections_slug) gin_trgm_ops)
  WHERE class_sections_deleted_at IS NULL;

-- Lookup by group_url (opsional)
CREATE INDEX IF NOT EXISTS idx_sections_group_url_alive
  ON class_sections (class_sections_group_url)
  WHERE class_sections_deleted_at IS NULL
    AND class_sections_group_url IS NOT NULL;

-- BRIN untuk tabel besar berbasis waktu (ringan)
CREATE INDEX IF NOT EXISTS brin_sections_created_at
  ON class_sections USING BRIN (class_sections_created_at);

-- =========================================================
-- PULIHKAN FK DARI TABEL LAIN (JIKA ADA), TANPA TRIGGER
-- =========================================================
-- user_class_sections → class_sections (komposit)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='user_class_sections' AND table_schema=current_schema()) THEN
    -- buang FK lama jika (masih) ada dengan nama lain
    IF EXISTS (
      SELECT 1 FROM information_schema.table_constraints
      WHERE table_name='user_class_sections' AND constraint_type='FOREIGN KEY' AND constraint_name='fk_ucs_section_masjid_pair'
    ) THEN
      ALTER TABLE user_class_sections DROP CONSTRAINT fk_ucs_section_masjid_pair;
    END IF;

    -- tambah lagi FK komposit yang benar
    ALTER TABLE user_class_sections
      ADD CONSTRAINT fk_ucs_section_masjid_pair
      FOREIGN KEY (user_class_sections_section_id, user_class_sections_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- class_schedules → class_sections (komposit)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='class_schedules' AND table_schema=current_schema()) THEN
    IF EXISTS (
      SELECT 1 FROM information_schema.table_constraints
      WHERE table_name='class_schedules' AND constraint_type='FOREIGN KEY' AND constraint_name='fk_cs_section_same_masjid'
    ) THEN
      ALTER TABLE class_schedules DROP CONSTRAINT fk_cs_section_same_masjid;
    END IF;

    ALTER TABLE class_schedules
      ADD CONSTRAINT fk_cs_section_same_masjid
      FOREIGN KEY (class_schedule_section_id, class_schedule_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

COMMIT;
