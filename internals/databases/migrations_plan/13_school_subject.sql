-- =======================================================================
-- UP MIGRATION — SUBJECTS, CLASS_SUBJECTS, CLASS_SECTION_SUBJECT_TEACHERS
-- (versi final lengkap: kolom + constraint + index + trigger)
-- =======================================================================
BEGIN;

-- ---------- EXTENSIONS (aman diulang) ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;     -- trigram (GIN)
CREATE EXTENSION IF NOT EXISTS btree_gist;  -- untuk EXCLUDE + rentang overlap

/* ======================================================================
   1) SUBJECTS — master mapel per tenant
   ====================================================================== */
DROP TABLE IF EXISTS subjects CASCADE;
CREATE TABLE IF NOT EXISTS subjects (
  subjects_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  subjects_code       VARCHAR(40)  NOT NULL,
  subjects_name       VARCHAR(120) NOT NULL,
  subjects_desc       TEXT,
  subjects_slug       VARCHAR(160) NOT NULL,

  -- Tambahan kolom (produksi)
  subjects_group        TEXT,           -- IPA/IPS, Kelompok A/B, dll
  subjects_stream       TEXT,           -- Cambridge/Nasional
  subjects_level_min    SMALLINT,
  subjects_level_max    SMALLINT,
  subjects_credit_units NUMERIC(4,2) CHECK (subjects_credit_units IS NULL OR subjects_credit_units >= 0),
  subjects_color_hex    VARCHAR(9),     -- #RRGGBB[AA]
  subjects_icon_url     TEXT,
  subjects_external_ref TEXT,

  -- Opsional kurikulum & integrasi
  subjects_syllabus_url     TEXT,

  subjects_is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
  subjects_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_subjects_code_not_blank CHECK (length(trim(subjects_code)) > 0),
  CONSTRAINT chk_subjects_slug_not_blank CHECK (length(trim(subjects_slug)) > 0),
  CONSTRAINT chk_subjects_level_range CHECK (
    subjects_level_min IS NULL OR subjects_level_max IS NULL OR subjects_level_max >= subjects_level_min
  )
);

-- Unik per tenant (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_code_per_masjid
  ON subjects (subjects_masjid_id, lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_slug_per_masjid
  ON subjects (subjects_masjid_id, lower(subjects_slug))
  WHERE subjects_deleted_at IS NULL;

-- Indeks pencarian & filter
CREATE INDEX IF NOT EXISTS gin_subjects_name_trgm
  ON subjects USING gin (subjects_name gin_trgm_ops)
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_masjid_alive
  ON subjects(subjects_masjid_id)
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_active
  ON subjects(subjects_masjid_id)
  WHERE subjects_is_active = TRUE AND subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_subjects_tags ON subjects USING GIN (subjects_tags);

-- Trigger auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_subjects() RETURNS trigger AS $$
BEGIN
  NEW.subjects_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_subjects_updated ON subjects;
CREATE TRIGGER trg_subjects_updated
BEFORE UPDATE ON subjects
FOR EACH ROW EXECUTE FUNCTION set_updated_at_subjects();


/* ======================================================================
   2) CLASS_SUBJECTS — binding kelas ↔️ mapel [+ term/periode]
   ====================================================================== */
DROP TABLE IF EXISTS class_subjects CASCADE;
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subjects_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_subjects_class_id   UUID NOT NULL,
  class_subjects_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,

  -- Periode akademik (opsional)
  class_subjects_term_id UUID,                -- FK → academic_terms (jika ada)
  class_subjects_academic_year_id UUID,
  class_subjects_term SMALLINT CHECK (class_subjects_term BETWEEN 1 AND 6),

  -- Konfigurasi & bobot
  class_subjects_order_index       INT,
  class_subjects_hours_per_week    INT CHECK (class_subjects_hours_per_week IS NULL OR class_subjects_hours_per_week >= 0),
  class_subjects_credit_units      NUMERIC(4,2) CHECK (class_subjects_credit_units IS NULL OR class_subjects_credit_units >= 0),
  class_subjects_min_passing_score INT  CHECK (class_subjects_min_passing_score BETWEEN 0 AND 100),
  class_subjects_weight_on_report  INT,
  class_subjects_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subjects_desc              TEXT,

  -- Kebijakan penilaian (granular)
  class_subjects_weight_assignment SMALLINT,
  class_subjects_weight_quiz       SMALLINT,
  class_subjects_weight_mid        SMALLINT,
  class_subjects_weight_final      SMALLINT,
  class_subjects_min_attendance_percent SMALLINT CHECK (
    class_subjects_min_attendance_percent IS NULL OR (class_subjects_min_attendance_percent BETWEEN 0 AND 100)
  ),


  -- Tambahan opsional (kapasitas & policy)
  class_subjects_exam_type TEXT CHECK (class_subjects_exam_type IN ('uas','un','tryout')),
  class_subjects_notes TEXT,
  class_subjects_materials_url TEXT,

  class_subjects_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_subjects_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_deleted_at  TIMESTAMPTZ,

  -- Guard total bobot <= 100 (atau kosong semua)
  CONSTRAINT chk_cs_weights_sum CHECK (
    (class_subjects_weight_assignment IS NULL
     AND class_subjects_weight_quiz   IS NULL
     AND class_subjects_weight_mid    IS NULL
     AND class_subjects_weight_final  IS NULL)
    OR (
      COALESCE(class_subjects_weight_assignment,0)
      + COALESCE(class_subjects_weight_quiz,0)
      + COALESCE(class_subjects_weight_mid,0)
      + COALESCE(class_subjects_weight_final,0)
    ) BETWEEN 0 AND 100
  )
);

-- FK komposit ke classes (tenant-safe)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cs_class_masjid_pair') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT fk_cs_class_masjid_pair
      FOREIGN KEY (class_subjects_class_id, class_subjects_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- FK ke academic_terms (opsional, hanya jika tabel ada)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='academic_terms')
     AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_term') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT fk_cs_term
      FOREIGN KEY (class_subjects_term_id)
      REFERENCES academic_terms(academic_terms_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- Unique soft-delete aware (by term)
DROP INDEX IF EXISTS uq_class_subjects_by_term;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subjects_by_term
ON class_subjects (
  class_subjects_masjid_id,
  class_subjects_class_id,
  class_subjects_subject_id,
  COALESCE(class_subjects_term_id::text,'')
)
WHERE class_subjects_deleted_at IS NULL;

-- Unik tambahan: periode tanpa term_id
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subjects_by_period
ON class_subjects (
  class_subjects_masjid_id,
  class_subjects_class_id,
  class_subjects_subject_id,
  COALESCE(class_subjects_academic_year_id::text,''),
  COALESCE(class_subjects_term::text,'')
)
WHERE class_subjects_deleted_at IS NULL AND class_subjects_term_id IS NULL;

-- Indeks umum & waktu
CREATE INDEX IF NOT EXISTS idx_cs_active
  ON class_subjects(class_subjects_masjid_id, class_subjects_class_id)
  WHERE class_subjects_is_active = TRUE AND class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_cs_created_at
  ON class_subjects USING BRIN (class_subjects_created_at);

CREATE INDEX IF NOT EXISTS idx_cs_effective_at
  ON class_subjects(class_subjects_effective_at);
CREATE INDEX IF NOT EXISTS idx_cs_effective_until
  ON class_subjects(class_subjects_effective_until);

CREATE INDEX IF NOT EXISTS idx_cs_term
  ON class_subjects(class_subjects_term_id);

CREATE INDEX IF NOT EXISTS idx_cs_academic_period
  ON class_subjects(class_subjects_academic_year_id, class_subjects_term);

CREATE INDEX IF NOT EXISTS idx_cs_subject_group
  ON class_subjects(class_subjects_subject_group);

CREATE INDEX IF NOT EXISTS gin_cs_tags ON class_subjects USING GIN (class_subjects_tags);

-- Overlap efektif (opsional tapi bagus untuk konsistensi)
CREATE INDEX IF NOT EXISTS ix_cs_overlap
ON class_subjects
USING GIST (
  class_subjects_class_id,
  class_subjects_subject_id,
  tstzrange(class_subjects_effective_at, COALESCE(class_subjects_effective_until,'infinity'::timestamptz))
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='ex_cs_no_overlap') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT ex_cs_no_overlap EXCLUDE USING GIST
      (
        class_subjects_class_id WITH =,
        class_subjects_subject_id WITH =,
        tstzrange(class_subjects_effective_at, COALESCE(class_subjects_effective_until,'infinity'::timestamptz)) WITH &&
      )
      WHERE (class_subjects_deleted_at IS NULL);
  END IF;
END$$;

-- Trigger auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_class_subjects() RETURNS trigger AS $$
BEGIN
  NEW.class_subjects_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cs_updated ON class_subjects;
CREATE TRIGGER trg_cs_updated
BEFORE UPDATE ON class_subjects
FOR EACH ROW EXECUTE FUNCTION set_updated_at_class_subjects();


/* ======================================================================
   3) CLASS_SECTION_SUBJECT_TEACHERS — penugasan guru per section+mapel
   ====================================================================== */
DROP TABLE IF EXISTS class_section_subject_teachers CASCADE;
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_section_subject_teachers_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  class_section_subject_teachers_section_id UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  class_section_subject_teachers_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON UPDATE CASCADE ON DELETE CASCADE,

  class_section_subject_teachers_teacher_id UUID NOT NULL
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE RESTRICT,

  -- Tambahan kolom penugasan
  class_section_subject_teachers_role TEXT
    CHECK (class_section_subject_teachers_role IN ('lead','assistant','co','guest')),
  class_section_subject_teachers_allocation_hours_per_week INT
    CHECK (class_section_subject_teachers_allocation_hours_per_week IS NULL OR class_section_subject_teachers_allocation_hours_per_week >= 0),
  class_section_subject_teachers_weight_percent SMALLINT
    CHECK (class_section_subject_teachers_weight_percent IS NULL OR (class_section_subject_teachers_weight_percent BETWEEN 0 AND 100)),
  class_section_subject_teachers_order_index INT,
  class_section_subject_teachers_notes TEXT,

  -- Denormalisasi cepat
  class_section_subject_teachers_class_id   UUID,
  class_section_subject_teachers_subject_id UUID,

  -- Periode penugasan
  class_section_subject_teachers_effective_at    TIMESTAMPTZ,
  class_section_subject_teachers_effective_until TIMESTAMPTZ,

  -- Audit & integrasi
  class_section_subject_teachers_created_by_user_id UUID,
  class_section_subject_teachers_updated_by_user_id UUID,
  class_section_subject_teachers_external_ref TEXT,
  class_section_subject_teachers_tags TEXT[],

  -- Tambahan opsional peran/komunikasi/evaluasi
  class_section_subject_teachers_is_homeroom BOOLEAN DEFAULT FALSE,
  class_section_subject_teachers_coordinator_id UUID,
  class_section_subject_teachers_contact_email TEXT,
  class_section_subject_teachers_contact_phone TEXT,
  class_section_subject_teachers_eval_score NUMERIC(5,2),
  class_section_subject_teachers_notes_admin TEXT,

  class_section_subject_teachers_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_deleted_at  TIMESTAMPTZ
);

-- Konsistensi section ↔️ tenant (komposit)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_csst_section_masjid_pair') THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT fk_csst_section_masjid_pair
      FOREIGN KEY (class_section_subject_teachers_section_id, class_section_subject_teachers_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- Unik aktif: hindari duplikasi penugasan aktif untuk (section, class_subjects, teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_active_section_subject_teacher
ON class_section_subject_teachers (
  class_section_subject_teachers_masjid_id,
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_class_subjects_id,
  class_section_subject_teachers_teacher_id
)
WHERE class_section_subject_teachers_deleted_at IS NULL
  AND class_section_subject_teachers_is_active = TRUE
  AND (
    class_section_subject_teachers_effective_until IS NULL
    OR class_section_subject_teachers_effective_until >= NOW()
  );

-- Opsional: hanya 1 "lead" aktif per (section, class_subjects)
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_active_lead_per_section_subject
ON class_section_subject_teachers (
  class_section_subject_teachers_masjid_id,
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_class_subjects_id
)
WHERE class_section_subject_teachers_deleted_at IS NULL
  AND class_section_subject_teachers_is_active = TRUE
  AND class_section_subject_teachers_role = 'lead'
  AND (
    class_section_subject_teachers_effective_until IS NULL
    OR class_section_subject_teachers_effective_until >= NOW()
  );

-- Kontrol overlap penugasan (per section+class_subjects+teacher)
CREATE INDEX IF NOT EXISTS ix_csst_overlap
ON class_section_subject_teachers
USING GIST (
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_class_subjects_id,
  class_section_subject_teachers_teacher_id,
  tstzrange(class_section_subject_teachers_effective_at, COALESCE(class_section_subject_teachers_effective_until,'infinity'::timestamptz))
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='ex_csst_no_overlap') THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT ex_csst_no_overlap EXCLUDE USING GIST
      (
        class_section_subject_teachers_section_id WITH =,
        class_section_subject_teachers_class_subjects_id WITH =,
        class_section_subject_teachers_teacher_id WITH =,
        tstzrange(class_section_subject_teachers_effective_at, COALESCE(class_section_subject_teachers_effective_until,'infinity'::timestamptz)) WITH &&
      )
      WHERE (class_section_subject_teachers_deleted_at IS NULL);
  END IF;
END$$;

-- Indeks umum
CREATE INDEX IF NOT EXISTS idx_csst_active
  ON class_section_subject_teachers(class_section_subject_teachers_masjid_id, class_section_subject_teachers_section_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL
    AND class_section_subject_teachers_is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_csst_teacher
  ON class_section_subject_teachers(class_section_subject_teachers_teacher_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_class_subjects
  ON class_section_subject_teachers(class_section_subject_teachers_class_subjects_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_effective_at
  ON class_section_subject_teachers(class_section_subject_teachers_effective_at);
CREATE INDEX IF NOT EXISTS idx_csst_effective_until
  ON class_section_subject_teachers(class_section_subject_teachers_effective_until);

CREATE INDEX IF NOT EXISTS gin_csst_tags ON class_section_subject_teachers USING GIN (class_section_subject_teachers_tags);

-- Trigger auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_csst() RETURNS trigger AS $$
BEGIN
  NEW.class_section_subject_teachers_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_csst_updated ON class_section_subject_teachers;
CREATE TRIGGER trg_csst_updated
BEFORE UPDATE ON class_section_subject_teachers
FOR EACH ROW EXECUTE FUNCTION set_updated_at_csst();

COMMIT;