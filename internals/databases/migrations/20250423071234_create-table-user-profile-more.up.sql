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



COMMIT;