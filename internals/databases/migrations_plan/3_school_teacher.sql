BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram & ilike cepat

-- =========================================================
-- ENUM: STATUS KEPEGAWAIAN GURU (INDONESIA)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'teacher_employment_enum') THEN
    CREATE TYPE teacher_employment_enum AS ENUM (
      'tetap','kontrak','paruh_waktu','magang','honorer','relawan','tamu'
    );
  END IF;
END$$;

-- =========================================================
-- TABEL: USER_TEACHERS (profil global per user)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_teachers (
  user_teacher_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_teacher_user_id              UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  -- Profil ringkas (GLOBAL)
  user_teacher_field                VARCHAR(80),
  user_teacher_short_bio            VARCHAR(300),
  user_teacher_greeting             TEXT,
  user_teacher_education            TEXT,
  user_teacher_activity             TEXT,
  user_teacher_experience_years     SMALLINT,

  -- Metadata fleksibel (GLOBAL)
  user_teacher_specialties          JSONB,  -- array: ["fiqih","tahfidz",...]
  user_teacher_certificates         JSONB,  -- array objek
  user_teacher_portfolio            JSONB,  -- array objek
  user_teacher_public_contacts      JSONB,  -- object: {whatsapp:"",...}

  -- Audit
  user_teacher_created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_teacher_updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_teacher_deleted_at           TIMESTAMPTZ,

  -- Guards
  CONSTRAINT ck_ut_exp_years_range CHECK (
    user_teacher_experience_years IS NULL
    OR user_teacher_experience_years BETWEEN 0 AND 80
  ),
  CONSTRAINT ck_ut_specialties_type CHECK (
    user_teacher_specialties IS NULL OR jsonb_typeof(user_teacher_specialties) = 'array'
  ),
  CONSTRAINT ck_ut_certificates_type CHECK (
    user_teacher_certificates IS NULL OR jsonb_typeof(user_teacher_certificates) = 'array'
  ),
  CONSTRAINT ck_ut_portfolio_type CHECK (
    user_teacher_portfolio IS NULL OR jsonb_typeof(user_teacher_portfolio) = 'array'
  ),
  CONSTRAINT ck_ut_public_contacts_type CHECK (
    user_teacher_public_contacts IS NULL OR jsonb_typeof(user_teacher_public_contacts) = 'object'
  )
);

-- Index utilitas
CREATE INDEX IF NOT EXISTS idx_ut_user ON user_teachers (user_teacher_user_id);
CREATE INDEX IF NOT EXISTS gin_ut_specialties ON user_teachers USING gin (user_teacher_specialties);
CREATE INDEX IF NOT EXISTS gin_ut_certificates ON user_teachers USING gin (user_teacher_certificates);
CREATE INDEX IF NOT EXISTS gin_ut_portfolio ON user_teachers USING gin (user_teacher_portfolio);
CREATE INDEX IF NOT EXISTS gin_ut_public_contacts ON user_teachers USING gin (user_teacher_public_contacts);

-- Trigger: lifecycle avatar + touch updated_at
CREATE OR REPLACE FUNCTION ut_on_avatar_change()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND NEW.user_teacher_avatar_url IS DISTINCT FROM OLD.user_teacher_avatar_url THEN
    IF OLD.user_teacher_avatar_url IS NOT NULL THEN
      NEW.user_teacher_avatar_deleted_url := OLD.user_teacher_avatar_url;
      NEW.user_teacher_avatar_delete_pending_until := now() + INTERVAL '7 days';
    END IF;
  END IF;
  NEW.user_teacher_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ut_avatar_change ON user_teachers;
CREATE TRIGGER trg_ut_avatar_change
BEFORE UPDATE ON user_teachers
FOR EACH ROW
EXECUTE FUNCTION ut_on_avatar_change();


-- =========================================================
-- TABEL: MASJID_TEACHERS (tanpa NIP)
-- =========================================================
CREATE TABLE IF NOT EXISTS school_teachers (
  school_teacher_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope/relasi
  school_teacher_school_id   UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
  school_teacher_user_id     UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,

  -- Identitas/kepegawaian (KHUSUS MASJID)
  school_teacher_code        VARCHAR(50),              -- unik per school (alive)
  school_teacher_title       VARCHAR(80),              -- Ust./Ustdz./dsb.
  school_teacher_employment  teacher_employment_enum,  -- status kepegawaian
  school_teacher_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  school_teacher_joined_at   DATE,
  school_teacher_left_at     DATE,

  -- Verifikasi internal
  school_teacher_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  school_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  school_teacher_is_public   BOOLEAN NOT NULL DEFAULT TRUE,
  school_teacher_notes       TEXT,

  -- Audit
  school_teacher_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_teacher_deleted_at  TIMESTAMPTZ,

  -- Validasi tanggal
  CONSTRAINT mtj_left_after_join_chk CHECK (
    school_teacher_left_at IS NULL
    OR school_teacher_joined_at IS NULL
    OR school_teacher_left_at >= school_teacher_joined_at
  )
);

-- Pair unik (tenant-safe join ops)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_mtj_id_school') THEN
    ALTER TABLE school_teachers
      ADD CONSTRAINT uq_mtj_id_school UNIQUE (school_teacher_id, school_teacher_school_id);
  END IF;
END$$;

-- INDEX & CONSTRAINTS (tanpa duplikasi profil)
-- Unik: 1 user per school (alive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_school_user_alive
  ON school_teachers (school_teacher_school_id, school_teacher_user_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Unik CODE per school (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_code_alive_ci
  ON school_teachers (school_teacher_school_id, LOWER(school_teacher_code))
  WHERE school_teacher_deleted_at IS NULL
    AND school_teacher_code IS NOT NULL;

-- (NIP constraint dihapus)

-- Lookups umum (per tenant), alive only
CREATE INDEX IF NOT EXISTS ix_mtj_tenant_active_public_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_active, school_teacher_is_public, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_verified_created
  ON school_teachers (school_teacher_school_id, school_teacher_is_verified, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_employment_created
  ON school_teachers (school_teacher_school_id, school_teacher_employment, school_teacher_created_at DESC)
  WHERE school_teacher_deleted_at IS NULL;

-- Akses cepat by user/school (alive only)
CREATE INDEX IF NOT EXISTS idx_mtj_user_alive
  ON school_teachers (school_teacher_user_id)
  WHERE school_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_school_alive
  ON school_teachers (school_teacher_school_id)
  WHERE school_teacher_deleted_at IS NULL;

-- Teks: cari di notes pakai ILIKE → trigram (alive only)
CREATE INDEX IF NOT EXISTS gin_mtj_notes_trgm_alive
  ON school_teachers USING GIN (LOWER(school_teacher_notes) gin_trgm_ops)
  WHERE school_teacher_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_mtj_joined_at
  ON school_teachers USING BRIN (school_teacher_joined_at);
CREATE INDEX IF NOT EXISTS brin_mtj_created_at
  ON school_teachers USING BRIN (school_teacher_created_at);

-- Trigger: touch updated_at pada update
CREATE OR REPLACE FUNCTION mtj_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.school_teacher_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mtj_touch ON school_teachers;
CREATE TRIGGER trg_mtj_touch
BEFORE UPDATE ON school_teachers
FOR EACH ROW
EXECUTE FUNCTION mtj_touch_updated_at();

-- =========================================================
-- VIEW: v_school_teachers_enriched (tanpa NIP)
-- =========================================================
CREATE OR REPLACE VIEW v_school_teachers_enriched AS
SELECT
  mt.school_teacher_id,
  mt.school_teacher_school_id,
  mt.school_teacher_user_id,

  -- status kepegawaian per school
  mt.school_teacher_code,
  mt.school_teacher_title,
  mt.school_teacher_employment,
  mt.school_teacher_is_active,
  mt.school_teacher_joined_at,
  mt.school_teacher_left_at,
  mt.school_teacher_is_verified,
  mt.school_teacher_verified_at,
  mt.school_teacher_is_public,
  mt.school_teacher_notes,
  mt.school_teacher_created_at,
  mt.school_teacher_updated_at,

  -- data user (ringkas)
  u.user_name,
  u.full_name,
  u.email,

  -- profil guru GLOBAL
  ut.user_teacher_id,
  ut.user_teacher_avatar_url,
  ut.user_teacher_field,
  ut.user_teacher_short_bio,
  ut.user_teacher_greeting,
  ut.user_teacher_education,
  ut.user_teacher_activity,
  ut.user_teacher_experience_years,
  ut.user_teacher_specialties,
  ut.user_teacher_certificates,
  ut.user_teacher_portfolio,
  ut.user_teacher_public_contacts
FROM school_teachers mt
JOIN users u ON u.id = mt.school_teacher_user_id
LEFT JOIN user_teachers ut
  ON ut.user_teacher_user_id = mt.school_teacher_user_id
WHERE mt.school_teacher_deleted_at IS NULL;

COMMIT;
