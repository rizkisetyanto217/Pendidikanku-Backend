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

   -- URL gambar class (opsional)
  class_image_url TEXT,

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

  class_sections_teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,

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

CREATE INDEX IF NOT EXISTS idx_sections_teacher ON class_sections(class_sections_teacher_id);

-- =========================================================
-- user_classes (enrolment siswa -> class/level)
-- Pembayaran bulanan: COALESCE(user_classes_fee_override_monthly_idr, classes.class_fee_monthly_idr)
-- =========================================================

-- Pastikan pasangan (class_id, class_masjid_id) unik di classes
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'uq_classes_id_masjid'
  ) THEN
    ALTER TABLE classes
      ADD CONSTRAINT uq_classes_id_masjid UNIQUE (class_id, class_masjid_id);
  END IF;
END$$;


-- 
-- 
-- 

CREATE TABLE IF NOT EXISTS user_classes (
  user_classes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_classes_user_id  UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- tetap simpan class_id anak
  user_classes_class_id UUID NOT NULL,

  -- denormalized tenant (filter cepat per masjid)
  user_classes_masjid_id UUID
    REFERENCES masjids(masjid_id) ON DELETE SET NULL,

  -- status enrolment
  user_classes_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_classes_status IN ('active','inactive','ended')),

  user_classes_started_at DATE,
  user_classes_ended_at   DATE,

  -- ended_at tidak boleh sebelum started_at
  CHECK (user_classes_ended_at IS NULL OR user_classes_ended_at >= user_classes_started_at),

  -- override tarif per bulan untuk siswa ini (NULL = ikut class_fee_monthly_idr)
  user_classes_fee_override_monthly_idr INT
    CHECK (user_classes_fee_override_monthly_idr IS NULL OR user_classes_fee_override_monthly_idr >= 0),

  user_classes_notes TEXT,

  user_classes_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_classes_updated_at TIMESTAMP
);



-- FK komposit: (class_id, masjid_id) di user_classes harus cocok dengan classes
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'fk_uc_class_masjid_pair'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_class_masjid_pair
      FOREIGN KEY (user_classes_class_id, user_classes_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- =========================================================
-- Indexes
-- =========================================================

-- Unik: hanya 1 enrolment 'active' yg belum berakhir per (user, class)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_classes_active_per_user_class
  ON user_classes(user_classes_user_id, user_classes_class_id)
  WHERE user_classes_status = 'active' AND user_classes_ended_at IS NULL;

-- Index dasar
CREATE INDEX IF NOT EXISTS idx_user_classes_user
  ON user_classes(user_classes_user_id);

CREATE INDEX IF NOT EXISTS idx_user_classes_class
  ON user_classes(user_classes_class_id);

CREATE INDEX IF NOT EXISTS idx_user_classes_status
  ON user_classes(user_classes_status);

CREATE INDEX IF NOT EXISTS idx_user_classes_started_at
  ON user_classes(user_classes_started_at);

CREATE INDEX IF NOT EXISTS idx_user_classes_ended_at
  ON user_classes(user_classes_ended_at);

CREATE INDEX IF NOT EXISTS idx_user_classes_created_at
  ON user_classes(user_classes_created_at DESC);

-- Filter cepat per tenant (tanpa JOIN)
CREATE INDEX IF NOT EXISTS idx_user_classes_masjid
  ON user_classes(user_classes_masjid_id);

-- Partial indexes untuk query enrolment aktif
CREATE INDEX IF NOT EXISTS idx_user_classes_user_active
  ON user_classes(user_classes_user_id, user_classes_class_id)
  WHERE user_classes_status = 'active' AND user_classes_ended_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_classes_class_active
  ON user_classes(user_classes_class_id, user_classes_user_id)
  WHERE user_classes_status = 'active' AND user_classes_ended_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_classes_masjid_active
  ON user_classes(user_classes_masjid_id, user_classes_user_id, user_classes_class_id)
  WHERE user_classes_status = 'active' AND user_classes_ended_at IS NULL;



-- =========================================================
-- user_class_sections (penempatan siswa ke section) — NO STATUS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  user_class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment siswa
  user_class_sections_user_class_id UUID NOT NULL,

  -- section (kelas-paralel)
  user_class_sections_section_id UUID NOT NULL,

  -- tenant (denormalized untuk filter cepat)
  user_class_sections_masjid_id UUID NOT NULL,

  user_class_sections_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  user_class_sections_unassigned_at DATE,

  -- unassign tidak boleh sebelum assign
  CONSTRAINT chk_ucs_dates
  CHECK (user_class_sections_unassigned_at IS NULL
         OR user_class_sections_unassigned_at >= user_class_sections_assigned_at),

  user_class_sections_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_class_sections_updated_at TIMESTAMP
);

-- ========== Unique & Indexes ==========
-- Unik: hanya 1 placement aktif (unassigned_at NULL) per enrolment
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_sections_active_per_user_class
  ON user_class_sections(user_class_sections_user_class_id)
  WHERE user_class_sections_unassigned_at IS NULL;

-- indeks dasar
CREATE INDEX IF NOT EXISTS idx_user_class_sections_user_class
  ON user_class_sections(user_class_sections_user_class_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_section
  ON user_class_sections(user_class_sections_section_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_assigned_at
  ON user_class_sections(user_class_sections_assigned_at);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_unassigned_at
  ON user_class_sections(user_class_sections_unassigned_at);

-- indeks per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid
  ON user_class_sections(user_class_sections_masjid_id);

-- partial index: yang aktif per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid_active
  ON user_class_sections(user_class_sections_masjid_id,
                         user_class_sections_user_class_id,
                         user_class_sections_section_id)
  WHERE user_class_sections_unassigned_at IS NULL;

-- (Tetap) Syarat untuk FK komposit di parent (kalau belum ada)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_classes_id_masjid'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT uq_user_classes_id_masjid
      UNIQUE (user_classes_id, user_classes_masjid_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_class_sections_id_masjid'
  ) THEN
    ALTER TABLE class_sections
      ADD CONSTRAINT uq_class_sections_id_masjid
      UNIQUE (class_sections_id, class_sections_masjid_id);
  END IF;
END$$;

-- FK komposit (tenant-safe)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_user_class_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      ADD CONSTRAINT fk_ucs_user_class_masjid_pair
      FOREIGN KEY (user_class_sections_user_class_id, user_class_sections_masjid_id)
      REFERENCES user_classes (user_classes_id, user_classes_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucs_section_masjid_pair'
  ) THEN
    ALTER TABLE user_class_sections
      ADD CONSTRAINT fk_ucs_section_masjid_pair
      FOREIGN KEY (user_class_sections_section_id, user_class_sections_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;



-- =========================================================
-- user_class_invoices (invoice bulanan per enrolment)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_invoices (
  invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment (FK ke user_classes)
  invoice_user_class_id UUID NOT NULL
    REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,

  invoice_period_month INT NOT NULL
    CHECK (invoice_period_month BETWEEN 200001 AND 999912),

  invoice_amount_idr INT NOT NULL CHECK (invoice_amount_idr >= 0),

  invoice_status TEXT NOT NULL DEFAULT 'unpaid'
    CHECK (invoice_status IN ('unpaid','paid','void')),

  invoice_due_date DATE,

  invoice_paid_at TIMESTAMP,
  invoice_paid_amount_idr INT CHECK (invoice_paid_amount_idr IS NULL OR invoice_paid_amount_idr >= 0),
  invoice_payment_method TEXT,
  invoice_payment_ref TEXT,

  invoice_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  invoice_updated_at TIMESTAMP
);

-- 1 invoice per (enrolment, bulan)
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoice_user_class_period
  ON user_class_invoices(invoice_user_class_id, invoice_period_month);

CREATE INDEX IF NOT EXISTS idx_invoice_user_class
  ON user_class_invoices(invoice_user_class_id);

CREATE INDEX IF NOT EXISTS idx_invoice_status
  ON user_class_invoices(invoice_status);

CREATE INDEX IF NOT EXISTS idx_invoice_period
  ON user_class_invoices(invoice_period_month);

CREATE INDEX IF NOT EXISTS idx_invoice_unpaid_partial
  ON user_class_invoices(invoice_user_class_id, invoice_period_month)
  WHERE invoice_status = 'unpaid';
