-- =========================================================
-- Supabase/Postgres setup (jika belum)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- untuk gen_random_uuid()

-- =========================================
-- classes (FIX: CHECK & penamaan kolom)
-- =========================================
CREATE TABLE IF NOT EXISTS classes (
  class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,

  class_name VARCHAR(120) NOT NULL,
  class_slug VARCHAR(160) UNIQUE NOT NULL,
  class_description TEXT,
  class_level TEXT, -- "TK A", "TK B", "Tahfidz", dst

  -- NULL = gratis; >= 0 = tarif per bulan (IDR)
  class_fee_monthly_idr INT
    CHECK (class_fee_monthly_idr IS NULL OR class_fee_monthly_idr >= 0),

  class_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  class_updated_at TIMESTAMP,
  class_deleted_at TIMESTAMP
);

-- Index umum
CREATE INDEX IF NOT EXISTS idx_classes_masjid
  ON classes(class_masjid_id);

CREATE INDEX IF NOT EXISTS idx_classes_active
  ON classes(class_is_active);

CREATE INDEX IF NOT EXISTS idx_classes_created_at
  ON classes(class_created_at DESC);

-- Index slug
CREATE INDEX IF NOT EXISTS idx_classes_slug
  ON classes(class_slug);

-- Opsional: case-insensitive slug lookup
CREATE INDEX IF NOT EXISTS idx_classes_slug_lower
  ON classes(LOWER(class_slug))
  WHERE class_deleted_at IS NULL;



-- ====================================================
-- class_sections
-- ====================================================
CREATE TABLE IF NOT EXISTS class_sections (
  class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- FK benar ke kolom class_id
  class_sections_class_id UUID NOT NULL
    REFERENCES classes(class_id) ON DELETE CASCADE,

  -- masjid_id di section (dibawa turun dari class)
  class_sections_masjid_id UUID
    REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  class_sections_slug VARCHAR(160) UNIQUE NOT NULL,

  class_sections_name VARCHAR(100) NOT NULL,  -- "A", "B", "Pagi"
  class_sections_code VARCHAR(50),
  class_sections_capacity INT
    CHECK (class_sections_capacity IS NULL OR class_sections_capacity >= 0),
  class_sections_schedule JSONB,

  class_sections_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  class_sections_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  class_sections_updated_at TIMESTAMP,
  class_sections_deleted_at TIMESTAMP
);

-- Unik: nama section per class (abaikan yang terhapus)
CREATE UNIQUE INDEX IF NOT EXISTS uq_sections_class_name
  ON class_sections(class_sections_class_id, class_sections_name)
  WHERE class_sections_deleted_at IS NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_sections_class
  ON class_sections(class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_sections_active
  ON class_sections(class_sections_is_active);
CREATE INDEX IF NOT EXISTS idx_sections_masjid
  ON class_sections(class_sections_masjid_id);
CREATE INDEX IF NOT EXISTS idx_sections_created_at
  ON class_sections(class_sections_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sections_slug 
  ON class_sections(class_sections_slug);


-- =========================================================
-- user_classes (enrolment siswa -> class/level)
-- Pembayaran bulanan: coalesce(uc_fee_override_monthly_idr, classes.class_fee_monthly_idr)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  uc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  uc_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uc_class_id UUID NOT NULL REFERENCES classes(class_id) ON DELETE CASCADE,

  -- status enrolment
  uc_status TEXT NOT NULL DEFAULT 'active'
    CHECK (uc_status IN ('active','inactive','ended')),

  uc_started_at DATE NOT NULL DEFAULT CURRENT_DATE,
  uc_ended_at   DATE,

  -- override tarif per bulan untuk siswa ini (NULL = ikut class_fee_monthly_idr)
  uc_fee_override_monthly_idr INT
    CHECK (uc_fee_override_monthly_idr IS NULL OR uc_fee_override_monthly_idr >= 0),

  uc_notes TEXT,

  uc_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  uc_updated_at TIMESTAMP
);

-- Satu enrolment 'active' per (user, class)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uc_active_per_user_class
  ON user_classes(uc_user_id, uc_class_id)
  WHERE uc_status = 'active' AND uc_ended_at IS NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_uc_user
  ON user_classes(uc_user_id);

CREATE INDEX IF NOT EXISTS idx_uc_class
  ON user_classes(uc_class_id);

CREATE INDEX IF NOT EXISTS idx_uc_status
  ON user_classes(uc_status);

-- (Opsional) bantu query historis/aktif berdasar tanggal mulai/akhir
CREATE INDEX IF NOT EXISTS idx_uc_started_at
  ON user_classes(uc_started_at);

CREATE INDEX IF NOT EXISTS idx_uc_ended_at
  ON user_classes(uc_ended_at);



-- =========================================================
-- user_class_sections (penempatan siswa ke section)
--    Satu siswa (per enrolment) hanya punya 1 section 'active'
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  ucs_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment siswa
  ucs_uc_id     UUID NOT NULL
    REFERENCES user_classes(uc_id) ON DELETE CASCADE,

  -- section (kelas-paralel)
  ucs_section_id UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  ucs_status TEXT NOT NULL DEFAULT 'active'
    CHECK (ucs_status IN ('active','inactive','ended')),

  ucs_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  ucs_unassigned_at DATE,

  ucs_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ucs_updated_at TIMESTAMP
);

-- Pastikan hanya 1 penempatan aktif per enrolment
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucs_active_per_uc
  ON user_class_sections(ucs_uc_id)
  WHERE ucs_status = 'active' AND ucs_unassigned_at IS NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_ucs_uc
  ON user_class_sections(ucs_uc_id);

CREATE INDEX IF NOT EXISTS idx_ucs_section
  ON user_class_sections(ucs_section_id);

CREATE INDEX IF NOT EXISTS idx_ucs_section_status
  ON user_class_sections(ucs_section_id, ucs_status);

-- Opsional: bantu query historis/aktif berdasar tanggal
CREATE INDEX IF NOT EXISTS idx_ucs_assigned_at
  ON user_class_sections(ucs_assigned_at);

CREATE INDEX IF NOT EXISTS idx_ucs_unassigned_at
  ON user_class_sections(ucs_unassigned_at);



-- =========================================================
-- 5) user_tpa_class_invoices (invoice bulanan per enrolment)
--    Aplikasi saat INSERT mengisi invoice_amount_idr:
--    COALESCE(utc_fee_override_monthly_idr, class_fee_monthly_idr, 0)
-- =========================================================
-- =========================================================
-- user_class_invoices (invoice bulanan per enrolment)
--    Saat INSERT, isi invoice_amount_idr:
--    COALESCE(uc_fee_override_monthly_idr, class_fee_monthly_idr, 0)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_invoices (
  invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment (FK ke user_classes)
  invoice_uc_id UUID NOT NULL REFERENCES user_classes(uc_id) ON DELETE CASCADE,

  -- periode bulan: format YYYYMM, contoh 202508
  invoice_period_month INT NOT NULL
    CHECK (invoice_period_month BETWEEN 200001 AND 999912),

  -- nominal bulan tersebut (salin tarif efektif saat generate)
  invoice_amount_idr INT NOT NULL CHECK (invoice_amount_idr >= 0),

  -- status sederhana
  invoice_status TEXT NOT NULL DEFAULT 'unpaid'
    CHECK (invoice_status IN ('unpaid','paid','void')),

  invoice_due_date DATE,

  -- bukti bayar (opsional, untuk pembayaran lunas)
  invoice_paid_at TIMESTAMP,
  invoice_paid_amount_idr INT CHECK (invoice_paid_amount_idr IS NULL OR invoice_paid_amount_idr >= 0),
  invoice_payment_method TEXT,  -- 'cash','transfer','qris', dll
  invoice_payment_ref TEXT,

  invoice_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  invoice_updated_at TIMESTAMP
);

-- 1 invoice per (enrolment, bulan)
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoice_uc_period
  ON user_class_invoices(invoice_uc_id, invoice_period_month);

-- Index umum
CREATE INDEX IF NOT EXISTS idx_invoice_uc
  ON user_class_invoices(invoice_uc_id);

CREATE INDEX IF NOT EXISTS idx_invoice_status
  ON user_class_invoices(invoice_status);

CREATE INDEX IF NOT EXISTS idx_invoice_period
  ON user_class_invoices(invoice_period_month);

-- Optional: cepat ambil yang belum lunas
CREATE INDEX IF NOT EXISTS idx_invoice_unpaid_partial
  ON user_class_invoices(invoice_uc_id, invoice_period_month)
  WHERE invoice_status = 'unpaid';