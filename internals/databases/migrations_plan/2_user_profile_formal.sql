-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (aman diulang)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index


/* =========================================================
   1) USERS_PROFILE_FORMAL
   ========================================================= */
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

  -- Alamat domisili
  users_profile_formal_address_line        TEXT,
  users_profile_formal_subdistrict         VARCHAR(100),
  users_profile_formal_city                VARCHAR(100),
  users_profile_formal_province            VARCHAR(100),
  users_profile_formal_postal_code         VARCHAR(10),

  -- Kontak darurat
  users_profile_formal_emergency_contact_name      VARCHAR(100),
  users_profile_formal_emergency_contact_relation  VARCHAR(30),
  users_profile_formal_emergency_contact_phone     VARCHAR(20),

  -- Identitas pribadi
  users_profile_formal_birth_place         VARCHAR(100),
  users_profile_formal_birth_date          DATE,
  users_profile_formal_nik                 VARCHAR(20),
  users_profile_formal_religion            VARCHAR(30),
  users_profile_formal_nationality         VARCHAR(50),

  -- Kesehatan ringkas
  users_profile_formal_medical_notes       TEXT,
  users_profile_formal_special_needs       TEXT,

  -- Verifikasi dokumen
  users_profile_formal_document_verification_status VARCHAR(20),
  users_profile_formal_document_verification_notes  TEXT,
  users_profile_formal_document_verified_by         UUID,
  users_profile_formal_document_verified_at         TIMESTAMPTZ,

  -- Audit
  users_profile_formal_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_profile_formal_updated_at        TIMESTAMPTZ,
  users_profile_formal_deleted_at        TIMESTAMPTZ,

  -- Unik per user
  CONSTRAINT uq_users_profile_formal_user UNIQUE (users_profile_formal_user_id),

  -- Hygiene sesuai kolom eksisting
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

CREATE INDEX IF NOT EXISTS idx_upf_verif_status_alive
  ON users_profile_formal (
    users_profile_formal_document_verification_status,
    users_profile_formal_updated_at DESC
  )
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



/* =========================================================
   2) MASJID_STUDENTS (pivot user ↔ masjid, role fixed student)
   ========================================================= */
CREATE TABLE IF NOT EXISTS masjid_students (
  masjid_student_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_student_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  masjid_student_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- Identitas siswa di tenant
  masjid_student_code VARCHAR(50),   -- kartu/member code (opsional)
  masjid_student_nim  VARCHAR(50),   -- NIM/NIS/NISN lokal per masjid/sekolah

  -- Status keanggotaan
  masjid_student_status TEXT NOT NULL DEFAULT 'active'
    CHECK (masjid_student_status IN ('active','inactive','alumni')),

  -- Operasional (histori)
  masjid_student_joined_at TIMESTAMPTZ,
  masjid_student_left_at   TIMESTAMPTZ,

  -- Catatan umum santri
  masjid_student_note TEXT,

  -- Penempatan akademik (snapshot ringan)
  masjid_student_current_class_id UUID,           -- FK opsional ke class_sections/rooms

  -- Audit core
  masjid_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_deleted_at TIMESTAMPTZ,

  -- Validasi pola (opsional & ringan)
  CONSTRAINT ck_ms_nim_format CHECK (
    masjid_student_nim IS NULL OR masjid_student_nim ~ '^[A-Za-z0-9._-]{3,30}$'
  ),
);

-- Pair unik (tenant-safe join ops) — idempotent via DO
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_ms_id_masjid') THEN
    ALTER TABLE masjid_students
      ADD CONSTRAINT uq_ms_id_masjid UNIQUE (masjid_student_id, masjid_student_masjid_id);
  END IF;
END$$;

-- Unik: 1 user AKTIF per masjid (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_user_per_masjid_live
  ON masjid_students (masjid_student_masjid_id, masjid_student_user_id)
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_status = 'active';

-- Unik CODE per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_code_alive_ci
  ON masjid_students (masjid_student_masjid_id, LOWER(masjid_student_code))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_code IS NOT NULL;

-- Unik NIM per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_nim_alive_ci
  ON masjid_students (masjid_student_masjid_id, LOWER(masjid_student_nim))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_nim IS NOT NULL;

-- Lookups umum per tenant (alive only) + created_at untuk pagination
CREATE INDEX IF NOT EXISTS ix_ms_tenant_status_created
  ON masjid_students (masjid_student_masjid_id, masjid_student_status, masjid_student_created_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_masjid_alive
  ON masjid_students (masjid_student_masjid_id)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_user_alive
  ON masjid_students (masjid_student_user_id)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_masjid_status_alive
  ON masjid_students (masjid_student_masjid_id, masjid_student_status)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_user_status_alive
  ON masjid_students (masjid_student_user_id, masjid_student_status)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_joined_at_alive
  ON masjid_students (masjid_student_joined_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_left_at_alive
  ON masjid_students (masjid_student_left_at)
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_left_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ms_created_at_alive
  ON masjid_students (masjid_student_created_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

-- Teks: catatan santri, pakai ILIKE
CREATE INDEX IF NOT EXISTS gin_ms_note_trgm_alive
  ON masjid_students USING GIN (LOWER(masjid_student_note) gin_trgm_ops)
  WHERE masjid_student_deleted_at IS NULL;

-- BRIN untuk range waktu besar (timeline ingestion)
CREATE INDEX IF NOT EXISTS brin_ms_created_at
  ON masjid_students USING BRIN (masjid_student_created_at);

-- Tambahan index untuk kolom rekomendasi (aktif bila kolom dipakai)
CREATE INDEX IF NOT EXISTS idx_ms_intake_year_alive
  ON masjid_students (masjid_student_intake_year)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_admission_batch_alive
  ON masjid_students (masjid_student_masjid_id, masjid_student_admission_batch)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_current_class_alive
  ON masjid_students (masjid_student_current_class_id)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_grade_alive
  ON masjid_students (masjid_student_current_grade)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_verified_alive
  ON masjid_students (masjid_student_is_verified, masjid_student_verified_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ms_status_changed_alive
  ON masjid_students (masjid_student_status, masjid_student_status_changed_at DESC)
  WHERE masjid_student_deleted_at IS NULL;

COMMIT;
