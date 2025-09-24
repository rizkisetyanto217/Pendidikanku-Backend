BEGIN;

-- safe to repeat
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- TABEL: user_classes  (plural table, singular columns)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  user_class_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- identitas siswa pada tenant (WAJIB)
  user_class_masjid_student_id UUID NOT NULL,

  -- kelas & tenant
  user_class_class_id  UUID NOT NULL,
  user_class_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  -- lifecycle enrolment
  user_class_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_class_status IN ('active','inactive','completed')),

  -- outcome (hasil akhir) - diisi hanya kalau completed
  user_class_result TEXT
    CHECK (user_class_result IN ('passed','failed')),

  -- billing ringan
  user_class_register_paid_at TIMESTAMPTZ,
  user_class_paid_until       TIMESTAMPTZ,
  user_class_paid_grace_days  SMALLINT NOT NULL DEFAULT 0
    CHECK (user_class_paid_grace_days BETWEEN 0 AND 100),

  -- jejak waktu enrolment per kelas
  user_class_joined_at    TIMESTAMPTZ,
  user_class_left_at      TIMESTAMPTZ,
  user_class_completed_at TIMESTAMPTZ,

  -- audit
  user_class_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_user_class_dates CHECK (
    user_class_left_at IS NULL
    OR user_class_joined_at IS NULL
    OR user_class_left_at >= user_class_joined_at
  ),
  CONSTRAINT chk_user_class_result_only_when_completed CHECK (
    (user_class_status = 'completed' AND user_class_result IS NOT NULL)
    OR (user_class_status <> 'completed' AND user_class_result IS NULL)
  ),
  CONSTRAINT chk_user_class_completed_at_when_completed CHECK (
    user_class_status <> 'completed'
    OR user_class_completed_at IS NOT NULL
  ),

  -- FK tenant-safe (komposit) ke classes
  CONSTRAINT fk_user_class__class_masjid_pair
    FOREIGN KEY (user_class_class_id, user_class_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- FK ke masjid_students (single)
  CONSTRAINT fk_user_class__masjid_student
    FOREIGN KEY (user_class_masjid_student_id)
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- Pair unik untuk join multi-tenant aman
  CONSTRAINT uq_user_class_id_masjid
    UNIQUE (user_class_id, user_class_masjid_id)
);

-- INDEXES user_classes (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_active_per_student_class
  ON user_classes (user_class_masjid_student_id, user_class_class_id, user_class_masjid_id)
  WHERE user_class_deleted_at IS NULL
    AND user_class_status = 'active';

CREATE INDEX IF NOT EXISTS ix_user_class_tenant_student_created
  ON user_classes (user_class_masjid_id, user_class_masjid_student_id, user_class_created_at DESC)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_class_tenant_status_created
  ON user_classes (user_class_masjid_id, user_class_status, user_class_created_at DESC)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_class_class_alive
  ON user_classes (user_class_class_id)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_class_masjid_alive
  ON user_classes (user_class_masjid_id)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_class_masjid_student_alive
  ON user_classes (user_class_masjid_student_id)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_class_tenant_class_active
  ON user_classes (user_class_masjid_id, user_class_class_id)
  WHERE user_class_deleted_at IS NULL
    AND user_class_status = 'active';

CREATE INDEX IF NOT EXISTS ix_user_class_dunning_tenant_status_due
  ON user_classes (user_class_masjid_id, user_class_status, user_class_paid_until)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_class_class_joined_at_alive
  ON user_classes (user_class_class_id, user_class_joined_at)
  WHERE user_class_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_user_class_class_completed_at_completed
  ON user_classes (user_class_class_id, user_class_completed_at)
  WHERE user_class_deleted_at IS NULL
    AND user_class_status = 'completed';

CREATE INDEX IF NOT EXISTS brin_user_class_created_at
  ON user_classes USING BRIN (user_class_created_at);

CREATE INDEX IF NOT EXISTS brin_user_class_updated_at
  ON user_classes USING BRIN (user_class_updated_at);



-- =========================================================
-- TABEL: user_class_sections (histori penempatan section)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  user_class_section_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment siswa
  user_class_section_user_class_id UUID NOT NULL,

  -- section (kelas paralel)
  user_class_section_section_id UUID NOT NULL,

  -- tenant (denormalized untuk filter cepat)
  user_class_section_masjid_id UUID NOT NULL,

  user_class_section_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  user_class_section_unassigned_at DATE,

  CONSTRAINT chk_user_class_section_dates CHECK (
    user_class_section_unassigned_at IS NULL
    OR user_class_section_unassigned_at >= user_class_section_assigned_at
  ),

  user_class_section_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_section_deleted_at TIMESTAMPTZ,

  -- FK komposit tenant-safe ke user_classes
  CONSTRAINT fk_user_class_section__user_class_masjid_pair
    FOREIGN KEY (user_class_section_user_class_id, user_class_section_masjid_id)
    REFERENCES user_classes (user_class_id, user_class_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_user_class_section__section_masjid_pair
    FOREIGN KEY (user_class_section_section_id, user_class_section_masjid_id)
    REFERENCES class_sections (class_section_id, class_section_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- INDEXES user_class_sections
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_section_active_per_user_class
  ON user_class_sections(user_class_section_user_class_id)
  WHERE user_class_section_unassigned_at IS NULL
    AND user_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_class_section__user_class
  ON user_class_sections(user_class_section_user_class_id);

CREATE INDEX IF NOT EXISTS idx_user_class_section__section
  ON user_class_sections(user_class_section_section_id);

CREATE INDEX IF NOT EXISTS idx_user_class_section_assigned_at
  ON user_class_sections(user_class_section_assigned_at);

CREATE INDEX IF NOT EXISTS idx_user_class_section_unassigned_at
  ON user_class_sections(user_class_section_unassigned_at);

CREATE INDEX IF NOT EXISTS idx_user_class_section_masjid
  ON user_class_sections(user_class_section_masjid_id);

CREATE INDEX IF NOT EXISTS idx_user_class_section_masjid_active
  ON user_class_sections (user_class_section_masjid_id,
                          user_class_section_user_class_id,
                          user_class_section_section_id)
  WHERE user_class_section_unassigned_at IS NULL
    AND user_class_section_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_user_class_section_created_at
  ON user_class_sections USING BRIN (user_class_section_created_at);

COMMIT;