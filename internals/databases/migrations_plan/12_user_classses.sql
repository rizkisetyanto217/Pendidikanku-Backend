
-- =========================================================
-- UP MIGRATION — USER_CLASSES & USER_CLASS_SECTIONS (FINAL+ADDONS)
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS (aman diulang) ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;     -- trigram (opsional)
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- utk EXCLUDE & overlap rentang waktu

-- =========================================================
-- TABEL: user_classes
-- =========================================================
CREATE TABLE IF NOT EXISTS user_classes (
  user_classes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- identitas siswa pada tenant (WAJIB)
  user_classes_masjid_student_id UUID NOT NULL,

  -- kelas & tenant
  user_classes_class_id  UUID NOT NULL,
  user_classes_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE RESTRICT,

  -- lifecycle enrolment
  user_classes_status TEXT NOT NULL DEFAULT 'active'
    CHECK (user_classes_status IN ('active','inactive','completed')),

  -- outcome (hasil akhir) - diisi hanya kalau completed
  user_classes_result TEXT
    CHECK (user_classes_result IN ('passed','failed')),

  -- jejak waktu enrolment per kelas
  user_classes_joined_at    TIMESTAMPTZ,
  user_classes_left_at      TIMESTAMPTZ,
  user_classes_completed_at TIMESTAMPTZ,

  -- audit waktu
  user_classes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_classes_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_classes_deleted_at TIMESTAMPTZ,


  -- sumber & kanal pendaftaran
  user_classes_enrollment_note TEXT,


  -- engagement & metrik cache
  user_classes_progress_percent SMALLINT
    CHECK (user_classes_progress_percent BETWEEN 0 AND 100),
  user_classes_attendance_count INT DEFAULT 0
    CHECK (user_classes_attendance_count >= 0),
  user_classes_absence_count INT DEFAULT 0
    CHECK (user_classes_absence_count >= 0),
  user_classes_grade_average NUMERIC(5,2),
  user_classes_first_attended_at TIMESTAMPTZ,
  user_classes_last_attended_at  TIMESTAMPTZ,
  user_classes_last_activity_at  TIMESTAMPTZ,
  user_classes_total_hours NUMERIC(6,2)
    CHECK (user_classes_total_hours IS NULL OR user_classes_total_hours >= 0),

  -- flags tambahan
  user_classes_is_repeated BOOLEAN DEFAULT FALSE,
  user_classes_dropout_reason TEXT,

  -- billing ringan
  user_classes_is_paid BOOLEAN,
  user_classes_price_cents INT CHECK (user_classes_price_cents >= 0),
  user_classes_currency CHAR(3),
  user_classes_billing_plan_id UUID,

  -- reward/sertifikasi
  user_classes_honor_points INT DEFAULT 0 CHECK (user_classes_honor_points >= 0),
  user_classes_certificate_url TEXT,

  -- integrasi eksternal
  user_classes_external_ref TEXT,
  user_classes_tags TEXT[],

  -- Guard tanggal (left >= joined)
  CONSTRAINT chk_uc_dates CHECK (
    user_classes_left_at IS NULL
    OR user_classes_joined_at IS NULL
    OR user_classes_left_at >= user_classes_joined_at
  ),

  -- Outcome hanya saat completed
  CONSTRAINT chk_uc_result_only_when_completed CHECK (
    (user_classes_status = 'completed' AND user_classes_result IS NOT NULL)
    OR (user_classes_status <> 'completed' AND user_classes_result IS NULL)
  ),

  -- Kalau completed, wajib ada completed_at
  CONSTRAINT chk_uc_completed_at_when_completed CHECK (
    user_classes_status <> 'completed'
    OR user_classes_completed_at IS NOT NULL
  ),

  -- FK tenant-safe (komposit) ke classes
  CONSTRAINT fk_uc_class_masjid_pair
    FOREIGN KEY (user_classes_class_id, user_classes_masjid_id)
    REFERENCES classes (class_id, class_masjid_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- FK ke masjid_students
  CONSTRAINT fk_uc_masjid_student
    FOREIGN KEY (user_classes_masjid_student_id)
    REFERENCES masjid_students(masjid_student_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- Guard unik multi-tenant
  CONSTRAINT uq_user_classes_id_masjid
    UNIQUE (user_classes_id, user_classes_masjid_id)
);

-- =======================
-- INDEKS user_classes
-- =======================
CREATE UNIQUE INDEX IF NOT EXISTS uq_uc_active_per_student_class
  ON user_classes (user_classes_masjid_student_id, user_classes_class_id, user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS ix_uc_tenant_student_created
  ON user_classes (user_classes_masjid_id, user_classes_masjid_student_id, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_uc_tenant_status_created
  ON user_classes (user_classes_masjid_id, user_classes_status, user_classes_created_at DESC)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_class_alive
  ON user_classes(user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_alive
  ON user_classes(user_classes_masjid_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_masjid_student_alive
  ON user_classes(user_classes_masjid_student_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_uc_tenant_class_active
  ON user_classes (user_classes_masjid_id, user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'active';

CREATE INDEX IF NOT EXISTS brin_uc_created_at
  ON user_classes USING BRIN (user_classes_created_at);

CREATE INDEX IF NOT EXISTS idx_uc_created_by     ON user_classes(user_classes_created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_uc_last_attended  ON user_classes(user_classes_last_attended_at);
CREATE INDEX IF NOT EXISTS idx_uc_last_activity  ON user_classes(user_classes_last_activity_at);

CREATE INDEX IF NOT EXISTS ix_uc_tenant_academic_term
  ON user_classes (user_classes_masjid_id, user_classes_academic_year_id, user_classes_term)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_teacher
  ON user_classes (user_classes_teacher_id)
  WHERE user_classes_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uc_total_hours
  ON user_classes (user_classes_total_hours);

-- GIN index utk tags
CREATE INDEX IF NOT EXISTS gin_uc_tags ON user_classes USING GIN (user_classes_tags);


-- =========================================================
-- TABEL: user_class_sections (histori penempatan section)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_sections (
  user_class_sections_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- enrolment siswa
  user_class_sections_user_class_id UUID NOT NULL,

  -- section (kelas paralel)
  user_class_sections_section_id UUID NOT NULL,

  -- tenant (denormalized)
  user_class_sections_masjid_id UUID NOT NULL,

  -- timeline penempatan
  user_class_sections_assigned_at   DATE NOT NULL DEFAULT CURRENT_DATE,
  user_class_sections_unassigned_at DATE,

  -- audit waktu
  user_class_sections_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_sections_deleted_at TIMESTAMPTZ,

  -- audit aktor & alasan
  user_class_sections_assigned_by_user_id   UUID,
  user_class_sections_unassigned_by_user_id UUID,
  user_class_sections_reason TEXT,
  user_class_sections_assignment_source TEXT
    CHECK (user_class_sections_assignment_source IN ('manual','import','promotion','api')),

  -- meta akademik & posisi
  user_class_sections_subject_group TEXT,  -- ex: IPA/IPS, Kelompok A/B
  user_class_sections_stream        TEXT,  -- ex: Cambridge/Nasional
  user_class_sections_roll_number   INT,
  user_class_sections_seat_number   INT CHECK (user_class_sections_seat_number IS NULL OR user_class_sections_seat_number > 0),
  user_class_sections_is_primary    BOOLEAN DEFAULT TRUE,

  -- integrasi & tag
  user_class_sections_external_ref TEXT,
  user_class_sections_tags TEXT[],

  -- guard tanggal
  CONSTRAINT chk_ucs_dates CHECK (
    user_class_sections_unassigned_at IS NULL
    OR user_class_sections_unassigned_at >= user_class_sections_assigned_at
  ),

  -- FK komposit tenant-safe ke user_classes
  CONSTRAINT fk_ucs_user_class_masjid_pair
    FOREIGN KEY (user_class_sections_user_class_id, user_class_sections_masjid_id)
    REFERENCES user_classes (user_classes_id, user_classes_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- FK komposit tenant-safe ke class_sections
  CONSTRAINT fk_ucs_section_masjid_pair
    FOREIGN KEY (user_class_sections_section_id, user_class_sections_masjid_id)
    REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- =======================
-- INDEKS user_class_sections
-- =======================
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_class_sections_active_per_user_class
  ON user_class_sections(user_class_sections_user_class_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- Cegah dobel section aktif pada kelas induk yg sama
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucs_active_per_user_class_parent
  ON user_class_sections (user_class_sections_user_class_id, user_class_sections_class_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_class_sections_user_class
  ON user_class_sections(user_class_sections_user_class_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_section
  ON user_class_sections(user_class_sections_section_id);

CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid
  ON user_class_sections(user_class_sections_masjid_id);

-- Timeline & masa berlaku
CREATE INDEX IF NOT EXISTS idx_user_class_sections_assigned_at
  ON user_class_sections(user_class_sections_assigned_at);
CREATE INDEX IF NOT EXISTS idx_user_class_sections_unassigned_at
  ON user_class_sections(user_class_sections_unassigned_at);
CREATE INDEX IF NOT EXISTS idx_ucs_effective_at
  ON user_class_sections(user_class_sections_effective_at);
CREATE INDEX IF NOT EXISTS idx_ucs_effective_until
  ON user_class_sections(user_class_sections_effective_until);

-- Aktif per tenant
CREATE INDEX IF NOT EXISTS idx_user_class_sections_masjid_active
  ON user_class_sections(user_class_sections_masjid_id,
                         user_class_sections_user_class_id,
                         user_class_sections_section_id)
  WHERE user_class_sections_unassigned_at IS NULL
    AND user_class_sections_deleted_at IS NULL;

-- BRIN waktu
CREATE INDEX IF NOT EXISTS brin_ucs_created_at
  ON user_class_sections USING BRIN (user_class_sections_created_at);

-- Akademik & meta
CREATE INDEX IF NOT EXISTS idx_ucs_class
  ON user_class_sections(user_class_sections_class_id);
CREATE INDEX IF NOT EXISTS idx_ucs_academic_term
  ON user_class_sections(user_class_sections_academic_year_id, user_class_sections_term);
CREATE INDEX IF NOT EXISTS idx_ucs_subject_group
  ON user_class_sections(user_class_sections_subject_group);
CREATE INDEX IF NOT EXISTS idx_ucs_stream
  ON user_class_sections(user_class_sections_stream);

-- GIN index utk tags
CREATE INDEX IF NOT EXISTS gin_ucs_tags ON user_class_sections USING GIN (user_class_sections_tags);


-- =========================================================
-- ADD-ONS #1 — Auto-update updated_at (triggers)
-- =========================================================
CREATE OR REPLACE FUNCTION set_updated_at_user_classes() RETURNS trigger AS $$
BEGIN
  NEW.user_classes_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_uc_updated ON user_classes;
CREATE TRIGGER trg_uc_updated
BEFORE UPDATE ON user_classes
FOR EACH ROW EXECUTE FUNCTION set_updated_at_user_classes();

CREATE OR REPLACE FUNCTION set_updated_at_user_class_sections() RETURNS trigger AS $$
BEGIN
  NEW.user_class_sections_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ucs_updated ON user_class_sections;
CREATE TRIGGER trg_ucs_updated
BEFORE UPDATE ON user_class_sections
FOR EACH ROW EXECUTE FUNCTION set_updated_at_user_class_sections();


-- =========================================================
-- ADD-ONS #2 — Larang overlap penempatan (EXCLUDE)
-- =========================================================
-- Index bantu overlap (GiST dgn btree_gist utk operator '=' di UUID)
CREATE INDEX IF NOT EXISTS ix_ucs_overlap
  ON user_class_sections
USING GIST (
  user_class_sections_user_class_id,
  tstzrange(user_class_sections_effective_at, COALESCE(user_class_sections_effective_until,'infinity'::timestamptz))
);

-- Exclusion constraint idempotent via DO-block
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'ex_ucs_no_overlap'
  ) THEN
    ALTER TABLE user_class_sections
      ADD CONSTRAINT ex_ucs_no_overlap EXCLUDE USING GIST
      (
        user_class_sections_user_class_id WITH =,
        tstzrange(user_class_sections_effective_at, COALESCE(user_class_sections_effective_until,'infinity'::timestamptz)) WITH &&
      )
      WHERE (user_class_sections_deleted_at IS NULL);
  END IF;
END$$;


-- =========================================================
-- ADD-ONS #3 — Guard billing konsisten
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chk_uc_billing_consistency'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT chk_uc_billing_consistency CHECK (
        (user_classes_is_paid IS TRUE  AND user_classes_price_cents IS NOT NULL AND user_classes_currency IS NOT NULL) OR
        (user_classes_is_paid IS NOT TRUE AND user_classes_price_cents IS NULL   AND user_classes_currency IS NULL)
      );
  END IF;
END$$;


-- =========================================================
-- ADD-ONS #4 — Validasi completed window (harus di dalam joined-left)
-- =========================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chk_uc_completed_window'
  ) THEN
    ALTER TABLE user_classes
      ADD CONSTRAINT chk_uc_completed_window CHECK (
        user_classes_completed_at IS NULL
        OR (
          (user_classes_joined_at IS NULL OR user_classes_completed_at >= user_classes_joined_at)
          AND
          (user_classes_left_at   IS NULL OR user_classes_completed_at <= user_classes_left_at)
        )
      );
  END IF;
END$$;


-- =========================================================
-- ADD-ONS #5 — (sudah dibuat) GIN index utk tags
-- (gin_uc_tags & gin_ucs_tags di atas)
-- =========================================================


-- =========================================================
-- ADD-ONS #6 — Partial index stateful (unpaid/active & completed recent)
-- =========================================================
CREATE INDEX IF NOT EXISTS ix_uc_unpaid_active
  ON user_classes(user_classes_masjid_id, user_classes_class_id)
  WHERE user_classes_deleted_at IS NULL
    AND COALESCE(user_classes_is_paid, FALSE) = FALSE
    AND user_classes_status IN ('active','inactive');

CREATE INDEX IF NOT EXISTS ix_uc_completed_recent
  ON user_classes(user_classes_masjid_id, user_classes_completed_at DESC)
  WHERE user_classes_deleted_at IS NULL
    AND user_classes_status = 'completed';

COMMIT;