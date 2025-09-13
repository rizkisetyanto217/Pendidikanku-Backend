BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- untuk GIN trigram

-- =========================================================
-- ENUM: STATUS KEPEGAWAIAN GURU (INDONESIA)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'teacher_employment_enum') THEN
    CREATE TYPE teacher_employment_enum AS ENUM (
      'tetap',        -- pegawai/guru tetap
      'kontrak',      -- kontrak
      'paruh_waktu',  -- part-time
      'magang',       -- intern
      'honorer',      -- honorer
      'relawan',      -- volunteer
      'tamu'          -- guest/kunjungan
    );
  END IF;
END$$;

-- =========================================================
-- TABEL: MASJID_TEACHERS
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teacher_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope/relasi
  masjid_teacher_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teacher_user_id   UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,

  -- Identitas/kepegawaian (level masjid)
  masjid_teacher_code        VARCHAR(50),              -- kode internal (unik per masjid; alive only)
  masjid_teacher_nip         VARCHAR(50),              -- NIP/NIM/NIK lokal (unik per masjid; alive only)
  masjid_teacher_title       VARCHAR(80),              -- jabatan/gelar (Ust., Ustdz., dsb.)
  masjid_teacher_employment  teacher_employment_enum,  -- status kepegawaian
  masjid_teacher_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  masjid_teacher_joined_at   DATE,
  masjid_teacher_left_at     DATE,

  -- Verifikasi internal
  masjid_teacher_is_verified BOOLEAN   NOT NULL DEFAULT FALSE,
  masjid_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  masjid_teacher_is_public   BOOLEAN   NOT NULL DEFAULT TRUE,
  masjid_teacher_notes       TEXT,

  -- Audit
  masjid_teacher_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_deleted_at  TIMESTAMPTZ,

  -- Validasi tanggal (left >= joined)
  CONSTRAINT mtj_left_after_join_chk CHECK (
    masjid_teacher_left_at IS NULL
    OR masjid_teacher_joined_at IS NULL
    OR masjid_teacher_left_at >= masjid_teacher_joined_at
  )
);

-- Pair unik (tenant-safe join ops)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_mtj_id_masjid') THEN
    ALTER TABLE masjid_teachers
      ADD CONSTRAINT uq_mtj_id_masjid UNIQUE (masjid_teacher_id, masjid_teacher_masjid_id);
  END IF;
END$$;

-- =======================
-- INDEX & CONSTRAINTS
-- =======================

-- Unik: 1 user per masjid (hanya baris hidup)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_masjid_user_alive
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Unik CODE per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_code_alive_ci
  ON masjid_teachers (masjid_teacher_masjid_id, LOWER(masjid_teacher_code))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_code IS NOT NULL;

-- Unik NIP per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_nip_alive_ci
  ON masjid_teachers (masjid_teacher_masjid_id, LOWER(masjid_teacher_nip))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_nip IS NOT NULL;

-- Lookups umum (per tenant), alive only
-- (masjid_id, filter…) + created_at untuk pagination/stable sort
CREATE INDEX IF NOT EXISTS ix_mtj_tenant_active_public_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_active, masjid_teacher_is_public, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_verified_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_verified, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_employment_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_employment, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Akses cepat by user (mis. ambil profil guru di satu masjid)
CREATE INDEX IF NOT EXISTS idx_mtj_user_alive
  ON masjid_teachers (masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_masjid_alive
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Teks: cari di notes/title pakai ILIKE → trigram (alive only)
CREATE INDEX IF NOT EXISTS gin_mtj_notes_trgm_alive
  ON masjid_teachers USING GIN (LOWER(masjid_teacher_notes) gin_trgm_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Range besar → BRIN tanggal (hemat RAM, cocok data bertumbuh waktu)
CREATE INDEX IF NOT EXISTS brin_mtj_joined_at
  ON masjid_teachers USING BRIN (masjid_teacher_joined_at);
CREATE INDEX IF NOT EXISTS brin_mtj_created_at
  ON masjid_teachers USING BRIN (masjid_teacher_created_at);

-- =========================================================
-- TABEL: MASJID_STUDENTS (ringkas, tanpa joined/left)
-- =========================================================
CREATE TABLE IF NOT EXISTS masjid_students (
  masjid_student_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_student_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  masjid_student_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  masjid_student_code VARCHAR(50),

  masjid_student_status TEXT NOT NULL DEFAULT 'active'
    CHECK (masjid_student_status IN ('active','inactive','alumni')),

  -- catatan umum santri
  masjid_student_note TEXT,

  masjid_student_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  masjid_student_deleted_at TIMESTAMPTZ
);


-- Pair unik (tenant-safe join ops)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_ms_id_masjid') THEN
    ALTER TABLE masjid_students
      ADD CONSTRAINT uq_ms_id_masjid UNIQUE (masjid_student_id, masjid_student_masjid_id);
  END IF;
END$$;

-- Unik: 1 user aktif per masjid (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ms_user_per_masjid_live
  ON masjid_students (masjid_student_masjid_id, masjid_student_user_id)
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_status = 'active';

-- (Opsional) Unik CODE per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_ms_code_alive_ci
  ON masjid_students (masjid_student_masjid_id, LOWER(masjid_student_code))
  WHERE masjid_student_deleted_at IS NULL
    AND masjid_student_code IS NOT NULL;

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

-- Teks: catatan santri, pakai ILIKE
CREATE INDEX IF NOT EXISTS gin_ms_note_trgm_alive
  ON masjid_students USING GIN (LOWER(masjid_student_note) gin_trgm_ops)
  WHERE masjid_student_deleted_at IS NULL;

-- BRIN untuk range waktu besar
CREATE INDEX IF NOT EXISTS brin_ms_created_at
  ON masjid_students USING BRIN (masjid_student_created_at);

COMMIT;
