-- =========================================
-- UP Migration — Certificates (simple, backend-driven)
-- =========================================
BEGIN;

-- =========================================================
-- 1) CERTIFICATE TEMPLATES (per school)
-- =========================================================
CREATE TABLE IF NOT EXISTS certificate_templates (
  certificate_templates_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  certificate_templates_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  certificate_templates_key  VARCHAR(48)  NOT NULL,        -- unik per school (mis: rapor, kelulusan, tahfidz)
  certificate_templates_name VARCHAR(160) NOT NULL,        -- nama human-friendly
  certificate_templates_desc TEXT,                         -- deskripsi opsional

  -- payload layout di-backend: field dinamis (background, posisi teks, font, dsb)
  certificate_templates_layout JSONB,                      -- backend-driven (bebas)

  certificate_templates_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  certificate_templates_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificate_templates_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificate_templates_deleted_at TIMESTAMPTZ
);

-- unik per school + key (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_certificate_templates_school_key_alive
  ON certificate_templates (certificate_templates_school_id, lower(certificate_templates_key))
  WHERE certificate_templates_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_certificate_templates_active
  ON certificate_templates (certificate_templates_school_id)
  WHERE certificate_templates_is_active = TRUE AND certificate_templates_deleted_at IS NULL;


-- =========================================================
-- 2) CERTIFICATES (terbitan sertifikat per siswa)
-- =========================================================
CREATE TABLE IF NOT EXISTS certificates (
  certificates_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  certificates_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- relasi konteks
  certificates_template_id UUID
    REFERENCES certificate_templates(certificate_templates_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  certificates_school_student_id UUID NOT NULL
    REFERENCES school_students(school_student_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  certificates_class_subject_id UUID
    REFERENCES class_subjects(class_subject_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- metadata umum
  certificates_title       VARCHAR(180) NOT NULL,        -- judul sertifikat (mis: "Sertifikat Kelulusan Fiqih")
  certificates_description TEXT,                         -- catatan opsional

  -- serial & status
  certificates_serial VARCHAR(64),                       -- nomor seri unik per school (opsional)
  certificates_status VARCHAR(16) NOT NULL DEFAULT 'issued'
    CHECK (certificates_status IN ('draft','issued','revoked')),

  certificates_issue_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- payload dinamis (backend-driven): mis. nama penerima, skor akhir, ranking, signers, dsb
  certificates_metadata JSONB,

  certificates_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificates_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  certificates_deleted_at TIMESTAMPTZ
);


-- Serial unik per school (jika diisi)
CREATE UNIQUE INDEX IF NOT EXISTS uq_certificates_school_serial_alive
  ON certificates (certificates_school_id, lower(certificates_serial))
  WHERE certificates_serial IS NOT NULL AND certificates_deleted_at IS NULL;

-- Indeks bantu
CREATE INDEX IF NOT EXISTS idx_certificates_school_created_at
  ON certificates (certificates_school_id, certificates_created_at DESC)
  WHERE certificates_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_certificates_student_alive
  ON certificates (certificates_school_student_id)
  WHERE certificates_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_certificates_status_alive
  ON certificates (certificates_status)
  WHERE certificates_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_certificates_created_at
  ON certificates USING BRIN (certificates_created_at);


COMMIT;
