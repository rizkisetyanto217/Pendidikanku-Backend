BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- opsional untuk pencarian

-- =========================================================
-- TABEL: user_classes (FINAL; enrolment ke masjid_students)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  user_classes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- identitas siswa pada tenant (WAJIB)
  user_classes_masjid_student_id UUID NOT NULL,

  -- kelas & tenant
  user_classes_class_id  UUID NOT NULL,
  user_classes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

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
  ),

  -- FK tenant-safe (komposit) ke classes
  CONSTRAINT fk_uc_class_masjid_pair
    FOREIGN KEY (user_classes_class_id, user_classes_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- FK ke masjid_students (single)
  CONSTRAINT fk_uc_masjid_student
    FOREIGN KEY (user_classes_masjid_student_id)
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- Guard unik untuk join aman multi-tenant (opsional)
  CONSTRAINT uq_user_classes_id_masjid
    UNIQUE (user_classes_id, user_classes_masjid_id)
);

-- =======================
-- Indeks & Partial Unique (soft-delete aware)
-- =======================

-- ❗ Cegah enrol aktif ganda per (masjid_student, class, masjid)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uc_active_per_student_class
  ON user_classes (user_classes_masjid_student_id, user_classes_class_id, user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

-- Lookups umum (tenant-first) + pagination
CREATE INDEX IF NOT EXISTS ix_uc_tenant_student_created
  ON user_classes (user_classes_masjid_id, user_classes_masjid_student_id, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_uc_tenant_status_created
  ON user_classes (user_classes_masjid_id, user_classes_status, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

-- Akses cepat (alive only)
CREATE INDEX IF NOT EXISTS idx_uc_class_alive
  ON user_classes(user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_alive
  ON user_classes(user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_student_alive
  ON user_classes(user_classes_masjid_student_id)
  WHERE user_classes_deleted_at IS NULL;

-- (Opsional) enrol aktif per tenant+class
CREATE INDEX IF NOT EXISTS ix_uc_tenant_class_active
  ON user_classes (user_classes_masjid_id, user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

-- (Opsional, ringan) BRIN untuk range besar
CREATE INDEX IF NOT EXISTS brin_uc_created_at
  ON user_classes USING BRIN (user_classes_created_at);



-- =========================================================
-- TABEL: user_class_sections (histori penempatan section) — TANPA TRIGGER
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

  CONSTRAINT chk_ucs_dates CHECK (
    user_class_sections_unassigned_at IS NULL
    OR user_class_sections_unassigned_at >= user_class_sections_assigned_at
  ),

  user_class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_deleted_at TIMESTAMPTZ,

  -- FK komposit tenant-safe ke user_classes
  CONSTRAINT fk_ucs_user_class_masjid_pair
    FOREIGN KEY (user_class_sections_user_class_id, user_class_sections_masjid_id)
    REFERENCES user_classes (user_classes_id, user_classes_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_ucs_section_masjid_pair
    FOREIGN KEY (user_class_sections_section_id, user_class_sections_masjid_id)
    REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- Unik: hanya 1 placement aktif per enrolment (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_sections_active_per_user_class
  ON user_class_sections(user_class_sections_user_class_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- Indeks dasar & per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_user_class
  ON user_class_sections(user_class_sections_user_class_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_section
  ON user_class_sections(user_class_sections_section_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_assigned_at
  ON user_class_sections(user_class_sections_assigned_at);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_unassigned_at
  ON user_class_sections(user_class_sections_unassigned_at);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid
  ON user_class_sections(user_class_sections_masjid_id);

-- Partial: current placement per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid_active
  ON user_class_sections(user_class_sections_masjid_id,
                         user_class_sections_user_class_id,
                         user_class_sections_section_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- (Opsional) BRIN waktu
CREATE INDEX IF NOT EXISTS brin_ucs_created_at
  ON user_class_sections USING BRIN (user_class_sections_created_at);

COMMIT;
