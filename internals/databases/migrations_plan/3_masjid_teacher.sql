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
CREATE TABLE IF NOT EXISTS masjid_teachers (
  masjid_teacher_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Scope/relasi
  masjid_teacher_masjid_id   UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  masjid_teacher_user_id     UUID NOT NULL REFERENCES users(id)          ON DELETE CASCADE,

  -- Identitas/kepegawaian (KHUSUS MASJID)
  masjid_teacher_code        VARCHAR(50),              -- unik per masjid (alive)
  masjid_teacher_title       VARCHAR(80),              -- Ust./Ustdz./dsb.
  masjid_teacher_employment  teacher_employment_enum,  -- status kepegawaian
  masjid_teacher_is_active   BOOLEAN NOT NULL DEFAULT TRUE,

  -- Periode kerja
  masjid_teacher_joined_at   DATE,
  masjid_teacher_left_at     DATE,

  -- Verifikasi internal
  masjid_teacher_is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  masjid_teacher_verified_at TIMESTAMPTZ,

  -- Visibilitas & catatan
  masjid_teacher_is_public   BOOLEAN NOT NULL DEFAULT TRUE,
  masjid_teacher_notes       TEXT,

  -- Audit
  masjid_teacher_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_teacher_deleted_at  TIMESTAMPTZ,

  -- Validasi tanggal
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

-- INDEX & CONSTRAINTS (tanpa duplikasi profil)
-- Unik: 1 user per masjid (alive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_masjid_user_alive
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Unik CODE per masjid (CI; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mtj_code_alive_ci
  ON masjid_teachers (masjid_teacher_masjid_id, LOWER(masjid_teacher_code))
  WHERE masjid_teacher_deleted_at IS NULL
    AND masjid_teacher_code IS NOT NULL;

-- (NIP constraint dihapus)

-- Lookups umum (per tenant), alive only
CREATE INDEX IF NOT EXISTS ix_mtj_tenant_active_public_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_active, masjid_teacher_is_public, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_verified_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_is_verified, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mtj_tenant_employment_created
  ON masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_employment, masjid_teacher_created_at DESC)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Akses cepat by user/masjid (alive only)
CREATE INDEX IF NOT EXISTS idx_mtj_user_alive
  ON masjid_teachers (masjid_teacher_user_id)
  WHERE masjid_teacher_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mtj_masjid_alive
  ON masjid_teachers (masjid_teacher_masjid_id)
  WHERE masjid_teacher_deleted_at IS NULL;

-- Teks: cari di notes pakai ILIKE → trigram (alive only)
CREATE INDEX IF NOT EXISTS gin_mtj_notes_trgm_alive
  ON masjid_teachers USING GIN (LOWER(masjid_teacher_notes) gin_trgm_ops)
  WHERE masjid_teacher_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_mtj_joined_at
  ON masjid_teachers USING BRIN (masjid_teacher_joined_at);
CREATE INDEX IF NOT EXISTS brin_mtj_created_at
  ON masjid_teachers USING BRIN (masjid_teacher_created_at);

-- Trigger: touch updated_at pada update
CREATE OR REPLACE FUNCTION mtj_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.masjid_teacher_updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mtj_touch ON masjid_teachers;
CREATE TRIGGER trg_mtj_touch
BEFORE UPDATE ON masjid_teachers
FOR EACH ROW
EXECUTE FUNCTION mtj_touch_updated_at();

-- =========================================================
-- VIEW: v_masjid_teachers_enriched (tanpa NIP)
-- =========================================================
CREATE OR REPLACE VIEW v_masjid_teachers_enriched AS
SELECT
  mt.masjid_teacher_id,
  mt.masjid_teacher_masjid_id,
  mt.masjid_teacher_user_id,

  -- status kepegawaian per masjid
  mt.masjid_teacher_code,
  mt.masjid_teacher_title,
  mt.masjid_teacher_employment,
  mt.masjid_teacher_is_active,
  mt.masjid_teacher_joined_at,
  mt.masjid_teacher_left_at,
  mt.masjid_teacher_is_verified,
  mt.masjid_teacher_verified_at,
  mt.masjid_teacher_is_public,
  mt.masjid_teacher_notes,
  mt.masjid_teacher_created_at,
  mt.masjid_teacher_updated_at,

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
FROM masjid_teachers mt
JOIN users u ON u.id = mt.masjid_teacher_user_id
LEFT JOIN user_teachers ut
  ON ut.user_teacher_user_id = mt.masjid_teacher_user_id
WHERE mt.masjid_teacher_deleted_at IS NULL;

COMMIT;
