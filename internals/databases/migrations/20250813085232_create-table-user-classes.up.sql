BEGIN;


-- Pastikan classes punya UNIQUE (class_id, class_masjid_id) untuk FK komposit
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_classes_id_masjid'
  ) THEN
    ALTER TABLE classes
      ADD CONSTRAINT uq_classes_id_masjid
      UNIQUE (class_id, class_masjid_id);
  END IF;
END$$;

-- Pastikan academic_terms punya UNIQUE (academic_terms_id, academic_terms_masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_academic_terms_id_masjid'
  ) THEN
    ALTER TABLE academic_terms
      ADD CONSTRAINT uq_academic_terms_id_masjid
      UNIQUE (academic_terms_id, academic_terms_masjid_id);
  END IF;
END$$;

-- =========================================================
-- Tabel: user_classes
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  user_classes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- siswa
  user_classes_user_id  UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- kelas yang diikuti pada term tsb (1 baris per user+class+term)
  user_classes_class_id UUID NOT NULL,

  -- tenant (wajib, agar FK komposit & filter cepat)
  user_classes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  -- term yang diikuti (wajib)
  user_classes_term_id UUID NOT NULL,

  -- (opsional) jejak opening untuk trace kuota/harga per term
  user_classes_opening_id UUID,

  -- status enrolment
  user_classes_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_classes_status IN ('active','inactive','ended')),

  -- snapshot biaya per siswa (NULL = ikut default/override lain)
  user_classes_fee_override_monthly_idr INT
    CHECK (user_classes_fee_override_monthly_idr IS NULL OR user_classes_fee_override_monthly_idr >= 0),

  user_classes_notes TEXT,

  user_classes_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_classes_updated_at TIMESTAMP,
  user_classes_deleted_at TIMESTAMP
);

-- =========================================================
-- Foreign Keys (tenant-safe)
-- =========================================================

-- FK komposit: (class_id, masjid_id) harus cocok dengan classes
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_class_masjid_pair'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_class_masjid_pair
      FOREIGN KEY (user_classes_class_id, user_classes_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- FK komposit: (term_id, masjid_id) harus cocok dengan academic_terms
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_term_masjid_pair'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_term_masjid_pair
      FOREIGN KEY (user_classes_term_id, user_classes_masjid_id)
      REFERENCES academic_terms (academic_terms_id, academic_terms_masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- (Opsional) FK: opening_id → class_term_openings
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_uc_opening'
  ) THEN
    -- Catatan: tidak komposit; asumsi sudah divalidasi di layer aplikasi.
    ALTER TABLE user_classes
      ADD CONSTRAINT fk_uc_opening
      FOREIGN KEY (user_classes_opening_id)
      REFERENCES class_term_openings (class_term_openings_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- Trigger: auto touch updated_at
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_user_classes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_classes_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_touch_user_classes ON user_classes;
CREATE TRIGGER trg_touch_user_classes
BEFORE UPDATE ON user_classes
FOR EACH ROW EXECUTE FUNCTION fn_touch_user_classes_updated_at();

-- =========================================================
-- Indexes & Partial Unique
-- =========================================================

-- Hindari duplikasi enrolment AKTIF pada kombinasi (user, class, term, masjid)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uc_active_per_user_class_term
  ON user_classes (user_classes_user_id, user_classes_class_id, user_classes_term_id, user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_uc_user
  ON user_classes (user_classes_user_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_class
  ON user_classes (user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid
  ON user_classes (user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_term
  ON user_classes (user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_created_at
  ON user_classes (user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

-- Indeks partial untuk query enrolment aktif
CREATE INDEX IF NOT EXISTS idx_uc_user_active
  ON user_classes (user_classes_user_id, user_classes_class_id, user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS idx_uc_class_active
  ON user_classes (user_classes_class_id, user_classes_user_id, user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS idx_uc_masjid_active
  ON user_classes (user_classes_masjid_id, user_classes_user_id, user_classes_class_id, user_classes_term_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

-- (Opsional) percepat lookup by opening
CREATE INDEX IF NOT EXISTS idx_uc_opening
  ON user_classes (user_classes_opening_id)
  WHERE user_classes_deleted_at IS NULL;


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


COMMIT;