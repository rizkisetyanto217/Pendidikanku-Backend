BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- untuk pencarian teks (kalau nanti perlu)

-- =========================================================
-- TABEL: user_classes (FINAL, jejak joined/left di sini)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  user_classes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_classes_user_id  UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_classes_class_id  UUID NOT NULL,
  user_classes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  user_classes_term_id UUID NOT NULL,

  -- relasi opsional ke masjid_students
  user_classes_masjid_student_id UUID,

  -- status enrolment
  user_classes_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_classes_status IN ('active','inactive','ended')),

  -- jejak waktu enrolment per kelas
  user_classes_joined_at TIMESTAMPTZ,
  user_classes_left_at   TIMESTAMPTZ,

  user_classes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_classes_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_classes_deleted_at TIMESTAMPTZ,

  -- Guard tanggal (left >= joined)
  CONSTRAINT chk_uc_dates CHECK (
    user_classes_left_at IS NULL
    OR user_classes_joined_at IS NULL
    OR user_classes_left_at >= user_classes_joined_at
  )
);

-- =======================
-- FK tenant-safe (tanpa trigger)
-- =======================
DO $$
BEGIN
  -- classes (pair)
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uc_class_masjid_pair') THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_class_masjid_pair
      FOREIGN KEY (user_classes_class_id, user_classes_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;

  -- academic_terms (pair)
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uc_term_masjid_pair') THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_term_masjid_pair
      FOREIGN KEY (user_classes_term_id, user_classes_masjid_id)
      REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;

  -- masjid_students (single)
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uc_masjid_student') THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_masjid_student
      FOREIGN KEY (user_classes_masjid_student_id)
      REFERENCES masjid_students(masjid_student_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- =======================
-- Unique & Indexes (tanpa trigger)
-- =======================

-- Pair unik untuk join aman multi-tenant
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_classes_id_masjid') THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT uq_user_classes_id_masjid
      UNIQUE (user_classes_id, user_classes_masjid_id);
  END IF;
END$$;

-- Cegah enrol aktif ganda (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uc_active_per_user_class_term
  ON user_classes (user_classes_user_id, user_classes_class_id, user_classes_term_id, user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

-- Lookups umum (tenant-first) + pagination by created_at
CREATE INDEX IF NOT EXISTS ix_uc_tenant_user_created
  ON user_classes (user_classes_masjid_id, user_classes_user_id, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_uc_tenant_class_term_active
  ON user_classes (user_classes_masjid_id, user_classes_class_id, user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS ix_uc_tenant_status_created
  ON user_classes (user_classes_masjid_id, user_classes_status, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

-- Akses cepat by foreign keys (alive only)
CREATE INDEX IF NOT EXISTS idx_uc_user_alive
  ON user_classes(user_classes_user_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_class_alive
  ON user_classes(user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_term_alive
  ON user_classes(user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_alive
  ON user_classes(user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_student_alive
  ON user_classes(user_classes_masjid_student_id)
  WHERE user_classes_deleted_at IS NULL;

-- (Opsional, ringan) BRIN untuk range besar
CREATE INDEX IF NOT EXISTS brin_uc_created_at
  ON user_classes USING BRIN (user_classes_created_at);

-- =========================================================
-- TABEL: user_class_sections (penempatan siswa ke section) — NO STATUS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  user_class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment siswa
  user_class_sections_user_class_id UUID NOT NULL,

  -- section (kelas paralel)
  user_class_sections_section_id UUID NOT NULL,

  -- tenant (denormalized untuk filter cepat)
  user_class_sections_masjid_id UUID NOT NULL,

  user_class_sections_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  user_class_sections_unassigned_at DATE,

  CONSTRAINT chk_ucs_dates
  CHECK (user_class_sections_unassigned_at IS NULL
         OR user_class_sections_unassigned_at >= user_class_sections_assigned_at),

  user_class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_deleted_at TIMESTAMPTZ
);

-- Unik: hanya 1 placement aktif per enrolment (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_sections_active_per_user_class
  ON user_class_sections(user_class_sections_user_class_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- Indeks dasar
CREATE INDEX IF NOT EXISTS idx_user_class_sections_user_class
  ON user_class_sections(user_class_sections_user_class_id);
CREATE INDEX IF NOT EXISTS idx_user_class_sections_section
  ON user_class_sections(user_class_sections_section_id);
CREATE INDEX IF NOT EXISTS idx_user_class_sections_assigned_at
  ON user_class_sections(user_class_sections_assigned_at);
CREATE INDEX IF NOT EXISTS idx_user_class_sections_unassigned_at
  ON user_class_sections(user_class_sections_unassigned_at);

-- Indeks per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid
  ON user_class_sections(user_class_sections_masjid_id);

-- Partial: aktif per tenant (sering untuk “current placement”)
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid_active
  ON user_class_sections(user_class_sections_masjid_id,
                         user_class_sections_user_class_id,
                         user_class_sections_section_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- (Opsional, ringan) BRIN waktu
CREATE INDEX IF NOT EXISTS brin_ucs_created_at
  ON user_class_sections USING BRIN (user_class_sections_created_at);

-- =======================
-- Syarat pair unik di parent (kalau belum ada)
-- =======================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_masjid
      UNIQUE (class_sections_id, class_sections_masjid_id);
  END IF;
END$$;

-- =======================
-- FK komposit tenant-safe (tanpa trigger)
-- =======================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_user_class_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      ADD CONSTRAINT fk_ucs_user_class_masjid_pair
      FOREIGN KEY (user_class_sections_user_class_id, user_class_sections_masjid_id)
      REFERENCES user_classes (user_classes_id, user_classes_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_section_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      ADD CONSTRAINT fk_ucs_section_masjid_pair
      FOREIGN KEY (user_class_sections_section_id, user_class_sections_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

COMMIT;
