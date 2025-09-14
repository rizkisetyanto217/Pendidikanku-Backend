-- +migrate Up
BEGIN;

-- =========================================================
-- EXTENSIONS (aman diulang)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- case-insensitive text
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram index

/* =========================================================
   1) USERS_PROFILE_FORMAL (global: orang tua, wali, alamat, identitas, kesehatan)
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

  -- Kondisi keluarga & ekonomi
  users_profile_formal_parent_marital_status VARCHAR(20), -- married/divorced/widowed/separated/other
  users_profile_formal_household_income    VARCHAR(50),
  users_profile_formal_number_of_siblings  SMALLINT,
  users_profile_formal_transport_mode      VARCHAR(30),   -- walk/bicycle/motorcycle/car/bus/train/other
  users_profile_formal_residence_status    VARCHAR(30),   -- own/rent/dorm/other
  users_profile_formal_language_at_home    VARCHAR(50),
  users_profile_formal_distance_to_school_km NUMERIC(6,2),

  -- Kesehatan ringkas
  users_profile_formal_medical_notes       TEXT,
  users_profile_formal_allergies           TEXT,
  users_profile_formal_special_needs       TEXT,
  users_profile_formal_disability_status   VARCHAR(20),   -- none/low/medium/high
  users_profile_formal_immunization_notes  TEXT,
  users_profile_formal_bpjs_number         VARCHAR(30),
  users_profile_formal_emergency_priority  SMALLINT,

  -- Dokumen dasar
  users_profile_formal_doc_birth_certificate_url  TEXT,
  users_profile_formal_doc_family_card_url        TEXT,
  users_profile_formal_doc_parent_id_url          TEXT,

  -- Verifikasi dokumen
  users_profile_formal_document_verification_status VARCHAR(20), -- pending/verified/rejected
  users_profile_formal_document_verification_notes  TEXT,
  users_profile_formal_document_verified_by         UUID,
  users_profile_formal_document_verified_at         TIMESTAMPTZ,

  -- Ekstensibilitas
  users_profile_formal_home_geo_lat      NUMERIC(9,6),
  users_profile_formal_home_geo_lng      NUMERIC(9,6),
  users_profile_formal_meta              JSONB,

  -- Audit
  users_profile_formal_created_by        UUID,
  users_profile_formal_updated_by        UUID,
  users_profile_formal_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  users_profile_formal_updated_at        TIMESTAMPTZ,
  users_profile_formal_deleted_at        TIMESTAMPTZ,

  -- Unik per user
  CONSTRAINT uq_users_profile_formal_user UNIQUE (users_profile_formal_user_id),

  -- Hygiene sederhana
  CONSTRAINT ck_users_profile_formal_parent_marital_status CHECK (
    users_profile_formal_parent_marital_status IS NULL OR
    users_profile_formal_parent_marital_status IN ('married','divorced','widowed','separated','other')
  ),
  CONSTRAINT ck_users_profile_formal_transport_mode CHECK (
    users_profile_formal_transport_mode IS NULL OR
    users_profile_formal_transport_mode IN ('walk','bicycle','motorcycle','car','bus','train','other')
  ),
  CONSTRAINT ck_users_profile_formal_disability_status CHECK (
    users_profile_formal_disability_status IS NULL OR
    users_profile_formal_disability_status IN ('none','low','medium','high')
  ),
  CONSTRAINT ck_users_profile_formal_postal_code CHECK (
    users_profile_formal_postal_code IS NULL OR users_profile_formal_postal_code ~ '^[0-9]{5,6}$'
  ),
  CONSTRAINT ck_users_profile_formal_nik CHECK (
    users_profile_formal_nik IS NULL OR users_profile_formal_nik ~ '^[0-9]{8,20}$'
  )
);

-- Index bantu (alive only)
CREATE INDEX IF NOT EXISTS idx_users_profile_formal_user_alive
  ON users_profile_formal (users_profile_formal_user_id)
  WHERE users_profile_formal_deleted_at IS NULL;


/* =========================================================
   2) USER_MASJID_MEMBERSHIPS (pivot user ↔ masjid)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_masjid_memberships (
  user_masjid_membership_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_masjid_membership_masjid_id     UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_masjid_membership_user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  user_masjid_membership_code          VARCHAR(50),
  user_masjid_membership_status        TEXT NOT NULL DEFAULT 'active' CHECK (
    user_masjid_membership_status IN ('active','inactive','alumni')
  ),

  -- catatan & meta
  user_masjid_membership_note          TEXT,
  user_masjid_membership_notes_internal TEXT,

  -- tambahan operasional
  user_masjid_membership_joined_at     TIMESTAMPTZ,
  user_masjid_membership_left_at       TIMESTAMPTZ,
  user_masjid_membership_role          TEXT,  -- student/teacher/staff/parent/volunteer/other
  user_masjid_membership_registration_channel TEXT, -- online/offline/referral/import/other
  user_masjid_membership_source        TEXT,  -- import/api/form/etc

  -- audit by
  user_masjid_membership_created_by    UUID,
  user_masjid_membership_updated_by    UUID,
  user_masjid_membership_deactivated_reason TEXT,
  user_masjid_membership_deactivated_by UUID,
  user_masjid_membership_deactivated_at TIMESTAMPTZ,
  user_masjid_membership_reactivated_by UUID,
  user_masjid_membership_reactivated_at TIMESTAMPTZ,

  -- audit timestamps
  user_masjid_membership_created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_masjid_membership_updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_masjid_membership_deleted_at    TIMESTAMPTZ,

  -- hygiene
  CONSTRAINT ck_user_masjid_membership_role CHECK (
    user_masjid_membership_role IS NULL OR
    user_masjid_membership_role IN ('student','teacher','staff','parent','volunteer','other')
  ),
  CONSTRAINT ck_user_masjid_membership_registration_channel CHECK (
    user_masjid_membership_registration_channel IS NULL OR
    user_masjid_membership_registration_channel IN ('online','offline','referral','import','other')
  )
);

-- Unik: 1 user aktif per masjid (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_umm_user_per_masjid_live
  ON user_masjid_memberships (user_masjid_membership_masjid_id, user_masjid_membership_user_id)
  WHERE user_masjid_membership_deleted_at IS NULL
    AND user_masjid_membership_status = 'active';

-- Code unik per masjid (case-insensitive; alive only)
CREATE UNIQUE INDEX IF NOT EXISTS ux_umm_code_alive_ci
  ON user_masjid_memberships (user_masjid_membership_masjid_id, LOWER(user_masjid_membership_code))
  WHERE user_masjid_membership_deleted_at IS NULL
    AND user_masjid_membership_code IS NOT NULL;

-- Lookups umum
CREATE INDEX IF NOT EXISTS idx_umm_masjid_alive
  ON user_masjid_memberships (user_masjid_membership_masjid_id)
  WHERE user_masjid_membership_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_umm_user_alive
  ON user_masjid_memberships (user_masjid_membership_user_id)
  WHERE user_masjid_membership_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_umm_status_created
  ON user_masjid_memberships (user_masjid_membership_status, user_masjid_membership_created_at DESC)
  WHERE user_masjid_membership_deleted_at IS NULL;


/* =========================================================
   3) USER_MASJID_MEMBERSHIP_ACADEMICS (akademik per membership/tahun)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_masjid_membership_academics (
  user_masjid_membership_academics_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_masjid_membership_id                       UUID NOT NULL REFERENCES user_masjid_memberships(user_masjid_membership_id) ON DELETE CASCADE,

  -- Identitas akademik
  user_masjid_membership_academics_student_local_id   VARCHAR(50),
  user_masjid_membership_academics_nisn               VARCHAR(20),
  user_masjid_membership_academics_education_level    VARCHAR(10) CHECK (
    user_masjid_membership_academics_education_level IN ('tk','sd','smp','sma','smk','ma','pt')
  ),
  user_masjid_membership_academics_grade_level        SMALLINT,
  user_masjid_membership_academics_class_name         VARCHAR(20),
  user_masjid_membership_academics_academic_year      VARCHAR(9),   -- "YYYY/YYYY"
  user_masjid_membership_academics_semester           SMALLINT,     -- 1/2
  user_masjid_membership_academics_enrollment_status  VARCHAR(12) DEFAULT 'active' CHECK (
    user_masjid_membership_academics_enrollment_status IN ('active','graduated','moved','inactive','suspended')
  ),
  user_masjid_membership_academics_admission_date     DATE,
  user_masjid_membership_academics_graduation_date    DATE,
  user_masjid_membership_academics_major              VARCHAR(50),
  user_masjid_membership_academics_curriculum         VARCHAR(50),
  user_masjid_membership_academics_homeroom_teacher   VARCHAR(100),

  -- Nilai & catatan
  user_masjid_membership_academics_gpa                   NUMERIC(4,2),     -- bebas skala, validasi di app
  user_masjid_membership_academics_scholarship_status    BOOLEAN,
  user_masjid_membership_academics_attendance_percentage NUMERIC(5,2),    -- 0..100
  user_masjid_membership_academics_behavior_score        SMALLINT,
  user_masjid_membership_academics_warning_level         SMALLINT,        -- 0..3
  user_masjid_membership_academics_notes                 TEXT,

  -- Dokumen
  user_masjid_membership_academics_doc_report_card_url   TEXT,
  user_masjid_membership_academics_certificate_url       TEXT,

  -- Enrichment
  user_masjid_membership_academics_stream                VARCHAR(30),
  user_masjid_membership_academics_advisor_name          VARCHAR(100),
  user_masjid_membership_academics_credit_total          SMALLINT,
  user_masjid_membership_academics_special_program       VARCHAR(50),
  user_masjid_membership_academics_remedial_required     BOOLEAN,
  user_masjid_membership_academics_promoted              BOOLEAN,
  user_masjid_membership_academics_promoted_to_grade     SMALLINT,

  -- Audit
  user_masjid_membership_academics_created_by            UUID,
  user_masjid_membership_academics_updated_by            UUID,
  user_masjid_membership_academics_created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_masjid_membership_academics_updated_at            TIMESTAMPTZ,
  user_masjid_membership_academics_deleted_at            TIMESTAMPTZ,

  -- Hygiene
  CONSTRAINT ck_umma_academic_year_format CHECK (
    user_masjid_membership_academics_academic_year IS NULL OR
    user_masjid_membership_academics_academic_year ~ '^[0-9]{4}/[0-9]{4}$'
  ),
  CONSTRAINT ck_umma_semester CHECK (
    user_masjid_membership_academics_semester IS NULL OR
    user_masjid_membership_academics_semester IN (1,2)
  ),
  CONSTRAINT ck_umma_attendance_pct CHECK (
    user_masjid_membership_academics_attendance_percentage IS NULL OR
    (user_masjid_membership_academics_attendance_percentage >= 0 AND user_masjid_membership_academics_attendance_percentage <= 100)
  )
);

-- Unik per membership & tahun ajaran (alive only) — kalau mau simpan 1 baris per TA
CREATE UNIQUE INDEX IF NOT EXISTS uq_umma_membership_year_alive
  ON user_masjid_membership_academics (user_masjid_membership_id, user_masjid_membership_academics_academic_year)
  WHERE user_masjid_membership_academics_deleted_at IS NULL;

-- Lookups umum
CREATE INDEX IF NOT EXISTS idx_umma_level_class_year_alive
  ON user_masjid_membership_academics (
    user_masjid_membership_academics_education_level,
    user_masjid_membership_academics_grade_level,
    user_masjid_membership_academics_class_name,
    user_masjid_membership_academics_academic_year
  )
  WHERE user_masjid_membership_academics_deleted_at IS NULL;


/* =========================================================
   4) USER_MASJID_MEMBERSHIP_HISTORY (opsional: riwayat status)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_masjid_membership_history (
  user_masjid_membership_history_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_masjid_membership_id           UUID NOT NULL REFERENCES user_masjid_memberships(user_masjid_membership_id) ON DELETE CASCADE,
  user_masjid_membership_history_prev_status TEXT,
  user_masjid_membership_history_new_status  TEXT,
  user_masjid_membership_history_reason      TEXT,
  user_masjid_membership_history_changed_by  UUID,
  user_masjid_membership_history_changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ummh_membership_changed_at
  ON user_masjid_membership_history (user_masjid_membership_id, user_masjid_membership_history_changed_at DESC);


/* =========================================================
   5) USER_CONSENT_LOGS (opsional: log persetujuan)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_consent_logs (
  user_consent_log_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_consent_log_type     TEXT,          -- photo/trip/data_processing/messaging/other
  user_consent_log_granted  BOOLEAN,
  user_consent_log_context  JSONB,
  user_consent_log_ip       INET,
  user_consent_log_user_agent TEXT,
  user_consent_log_created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_consent_logs_user_created
  ON user_consent_logs (user_id, user_consent_log_created_at DESC);


/* =========================================================
   6) USER_MASJID_MEMBERSHIP_FINANCE (opsional: administrasi biaya)
   ========================================================= */
CREATE TABLE IF NOT EXISTS user_masjid_membership_finance (
  user_masjid_membership_finance_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_masjid_membership_id           UUID NOT NULL REFERENCES user_masjid_memberships(user_masjid_membership_id) ON DELETE CASCADE,

  user_masjid_membership_finance_fee_plan         VARCHAR(50),
  user_masjid_membership_finance_fee_amount       NUMERIC(12,2),
  user_masjid_membership_finance_fee_currency     VARCHAR(10),
  user_masjid_membership_finance_scholarship_amount NUMERIC(12,2),
  user_masjid_membership_finance_payment_status   VARCHAR(20),  -- current/overdue/cleared
  user_masjid_membership_finance_last_payment_at  TIMESTAMPTZ,
  user_masjid_membership_finance_notes            TEXT,

  user_masjid_membership_finance_created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_masjid_membership_finance_updated_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ummf_membership_status
  ON user_masjid_membership_finance (user_masjid_membership_id, user_masjid_membership_finance_payment_status);

COMMIT;
