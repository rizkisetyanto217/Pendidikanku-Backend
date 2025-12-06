-- +migrate Up
BEGIN;

-- =========================================================
-- GENERAL billings (tetap, untuk non-per-siswa/campaign)
-- =========================================================
CREATE TABLE IF NOT EXISTS general_billings (
  general_billing_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  general_billing_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  general_billing_kind_id    UUID NOT NULL
    REFERENCES general_billing_kinds(general_billing_kind_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  general_billing_code       VARCHAR(60),
  general_billing_title      TEXT NOT NULL,
  general_billing_desc       TEXT,

  -- cakupan akademik (opsional)
  general_billing_class_id   UUID REFERENCES classes(class_id) ON DELETE SET NULL,
  general_billing_section_id UUID REFERENCES class_sections(class_section_id) ON DELETE SET NULL,
  general_billing_term_id    UUID REFERENCES academic_terms(academic_term_id) ON DELETE SET NULL,

  general_billing_due_date   DATE,
  general_billing_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  general_billing_default_amount_idr INT CHECK (general_billing_default_amount_idr >= 0),

  -- snapshots (MINIMAL)
  general_billing_kind_snapshot    JSONB,  -- {id, code, name}
  general_billing_class_snapshot   JSONB,  -- {id, name, slug}
  general_billing_section_snapshot JSONB,  -- {id, name, code}
  general_billing_term_snapshot    JSONB,  -- {id, academic_year, name, slug}

  general_billing_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  general_billing_deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_general_billings_code_per_tenant_alive
  ON general_billings (general_billing_school_id, LOWER(general_billing_code))
  WHERE general_billing_deleted_at IS NULL AND general_billing_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_gb_tenant_kind_active_created
  ON general_billings (general_billing_school_id, general_billing_kind_id, general_billing_is_active, general_billing_created_at DESC)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_due_alive
  ON general_billings (general_billing_due_date)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_kind_alive
  ON general_billings (general_billing_kind_id)
  WHERE general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_gb_term_alive
  ON general_billings (general_billing_term_id)
  WHERE general_billing_deleted_at IS NULL;

-- =========================================================
-- USER general billings (assign/tagihan ke user/siswa untuk GB di atas)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_general_billings (
  user_general_billing_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_general_billing_school_id          UUID NOT NULL,

  -- relasi ke siswa (opsional) — composite FK (id, school_id)
  user_general_billing_school_student_id  UUID,
  CONSTRAINT fk_ugb_student_tenant FOREIGN KEY (user_general_billing_school_student_id, user_general_billing_school_id)
    REFERENCES school_students (school_student_id, school_student_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- payer (opsional)
  user_general_billing_payer_user_id      UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- referensi ke general_billing (wajib)
  user_general_billing_billing_id         UUID NOT NULL
    REFERENCES general_billings(general_billing_id) ON DELETE CASCADE,

  -- nilai & status
  user_general_billing_amount_idr         INT NOT NULL CHECK (user_general_billing_amount_idr >= 0),
  user_general_billing_status             VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK (user_general_billing_status IN ('unpaid','paid','canceled')),
  user_general_billing_paid_at            TIMESTAMPTZ,
  user_general_billing_note               TEXT,

  -- snapshots ringan
  user_general_billing_title_snapshot     TEXT,
  user_general_billing_kind_code_snapshot TEXT,
  user_general_billing_kind_name_snapshot TEXT,

  -- metadata fleksibel
  user_general_billing_meta               JSONB DEFAULT '{}'::jsonb,

  -- timestamps (soft delete)
  user_general_billing_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_general_billing_deleted_at         TIMESTAMPTZ
);

-- Unik per student untuk satu billing (abaikan baris yang soft-deleted)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ugb_per_student_alive
  ON user_general_billings (user_general_billing_billing_id, user_general_billing_school_student_id)
  WHERE user_general_billing_deleted_at IS NULL;

-- Unik per payer untuk satu billing (abaikan baris yang soft-deleted)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ugb_per_payer_alive
  ON user_general_billings (user_general_billing_billing_id, user_general_billing_payer_user_id)
  WHERE user_general_billing_deleted_at IS NULL;

-- Indeks bantu query
CREATE INDEX IF NOT EXISTS ix_ugb_school_alive
  ON user_general_billings (user_general_billing_school_id)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_billing_alive
  ON user_general_billings (user_general_billing_billing_id)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_status_alive
  ON user_general_billings (user_general_billing_status)
  WHERE user_general_billing_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ugb_created_at_alive
  ON user_general_billings (user_general_billing_created_at DESC)
  WHERE user_general_billing_deleted_at IS NULL;

-- (opsional) index GIN untuk meta bila sering filter by meta->key
-- CREATE INDEX IF NOT EXISTS ix_ugb_meta_gin_alive
--   ON user_general_billings USING GIN (user_general_billing_meta)
--   WHERE user_general_billing_deleted_at IS NULL;

COMMIT;
