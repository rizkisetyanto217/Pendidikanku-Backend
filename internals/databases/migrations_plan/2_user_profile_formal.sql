-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (aman diulang)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index


-- 1) FRESH CREATE — users_profile_formal (TANPA kolom verifikasi dokumen)
-- =========================================================
CREATE TABLE IF NOT EXISTS users_profile_formal (
  users_profile_formal_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  users_profile_formal_user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Orang tua / wali
  users_profile_formal_father_name         VARCHAR(50),
  users_profile_formal_father_phone        VARCHAR(20),
  users_profile_formal_mother_name         VARCHAR(50),
  users_profile_formal_mother_phone        VARCHAR(20),
  users_profile_formal_guardian_name       VARCHAR(50),
  users_profile_formal_guardian_phone      VARCHAR(20),
  users_profile_formal_guardian_relation   VARCHAR(30),

  -- Kontak darurat
  users_profile_formal_emergency_contact_name      VARCHAR(100),
  users_profile_formal_emergency_contact_relation  VARCHAR(30),
  users_profile_formal_emergency_contact_phone     VARCHAR(20),

  -- Identitas pribadi
  users_profile_formal_nik                 VARCHAR(20),
  users_profile_formal_religion            VARCHAR(30),
  users_profile_formal_nationality         VARCHAR(50),

  -- Kesehatan ringkas
  users_profile_formal_medical_notes       TEXT,
  users_profile_formal_special_needs       TEXT,

  -- Audit
  users_profile_formal_created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_profile_formal_updated_at          TIMESTAMPTZ,
  users_profile_formal_deleted_at          TIMESTAMPTZ,

  -- Unik per user
  CONSTRAINT uq_users_profile_formal_user UNIQUE (users_profile_formal_user_id),

  -- Hygiene
  CONSTRAINT ck_users_profile_formal_postal_code CHECK (
    users_profile_formal_postal_code IS NULL OR users_profile_formal_postal_code ~ '^[0-9]{5,6}$'
  ),
  CONSTRAINT ck_users_profile_formal_nik CHECK (
    users_profile_formal_nik IS NULL OR users_profile_formal_nik ~ '^[0-9]{8,20}$'
  )
);

-- Index (alive only)
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_user_alive
  ON users_profile_formal (users_profile_formal_user_id)
  WHERE users_profile_formal_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upf_nik_alive
  ON users_profile_formal (users_profile_formal_nik)
  WHERE users_profile_formal_deleted_at IS NULL
    AND users_profile_formal_nik IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_upf_birth_date_alive
  ON users_profile_formal (users_profile_formal_birth_date)
  WHERE users_profile_formal_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upf_city_province_alive
  ON users_profile_formal (users_profile_formal_city, users_profile_formal_province)
  WHERE users_profile_formal_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upf_address_trgm
  ON users_profile_formal USING GIN (users_profile_formal_address_line gin_trgm_ops);

COMMIT;


/* =========================================================
   2) MASJID_STUDENTS (pivot user ↔ school, role fixed student)
   ========================================================= */
CREATE TABLE IF NOT EXISTS school_students (
  school_student_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  school_student_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  school_student_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- Identitas siswa di tenant
  school_student_code VARCHAR(50),   -- kartu/member code (opsional)
  school_student_nim  VARCHAR(50),   -- NIM/NIS/NISN lokal per school/sekolah

  -- Status keanggotaan
  school_student_status TEXT NOT NULL DEFAULT 'active'
    CHECK (school_student_status IN ('active','inactive','alumni')),

  -- Operasional (histori)
  school_student_joined_at TIMESTAMPTZ,
  school_student_left_at   TIMESTAMPTZ,

  -- Catatan umum santri
  school_student_note TEXT,

  -- Penempatan akademik (snapshot ringan)
  school_student_current_class_id UUID,           -- FK opsional ke class_sections/rooms

  -- Audit core
  school_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  school_student_deleted_at TIMESTAMPTZ,

  -- Validasi pola (opsional & ringan)
  CONSTRAINT ck_ms_nim_format CHECK (
    school_student_nim IS NULL OR school_student_nim ~ '^[A-Za-z0-9._-]{3,30}$'
  ),
);

-- Pair unik (tenant-safe join ops) — idempotent via DO
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_ms_id_school') THEN
    ALTER TABLE school_students
      ADD CONSTRAINT uq_ms_id_school UNIQUE (school_student_id, school_student_school_id);
  END IF;
END$$;

-- Unik: 1 user AKTIF per school (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_user_per_school_live
  ON school_students (school_student_school_id, school_student_user_id)
  WHERE school_student_deleted_at IS NULL
    AND school_student_status = 'active';

-- Unik CODE per school (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_code_alive_ci
  ON school_students (school_student_school_id, LOWER(school_student_code))
  WHERE school_student_deleted_at IS NULL
    AND school_student_code IS NOT NULL;

-- Unik NIM per school (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_nim_alive_ci
  ON school_students (school_student_school_id, LOWER(school_student_nim))
  WHERE school_student_deleted_at IS NULL
    AND school_student_nim IS NOT NULL;

-- Lookups umum per tenant (alive only) + created_at untuk pagination
CREATE INDEX IF NOT EXISTS ix_ms_tenant_status_created
  ON school_students (school_student_school_id, school_student_status, school_student_created_at DESC)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_school_alive
  ON school_students (school_student_school_id)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_user_alive
  ON school_students (school_student_user_id)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_school_status_alive
  ON school_students (school_student_school_id, school_student_status)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_user_status_alive
  ON school_students (school_student_user_id, school_student_status)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_joined_at_alive
  ON school_students (school_student_joined_at DESC)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_left_at_alive
  ON school_students (school_student_left_at)
  WHERE school_student_deleted_at IS NULL
    AND school_student_left_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ms_created_at_alive
  ON school_students (school_student_created_at DESC)
  WHERE school_student_deleted_at IS NULL;

-- Teks: catatan santri, pakai ILIKE
CREATE INDEX IF NOT EXISTS gin_ms_note_trgm_alive
  ON school_students USING GIN (LOWER(school_student_note) gin_trgm_ops)
  WHERE school_student_deleted_at IS NULL;

-- BRIN untuk range waktu besar (timeline ingestion)
CREATE INDEX IF NOT EXISTS brin_ms_created_at
  ON school_students USING BRIN (school_student_created_at);

-- Tambahan index untuk kolom rekomendasi (aktif bila kolom dipakai)
CREATE INDEX IF NOT EXISTS idx_ms_intake_year_alive
  ON school_students (school_student_intake_year)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_admission_batch_alive
  ON school_students (school_student_school_id, school_student_admission_batch)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_current_class_alive
  ON school_students (school_student_current_class_id)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_grade_alive
  ON school_students (school_student_current_grade)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_verified_alive
  ON school_students (school_student_is_verified, school_student_verified_at DESC)
  WHERE school_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_status_changed_alive
  ON school_students (school_student_status, school_student_status_changed_at DESC)
  WHERE school_student_deleted_at IS NULL;

COMMIT;
