-- =========================================================
-- FRESH/SAFE INSTALL: Subjects, Class-Subjects, CSST (refer to class_subjects)
-- Idempotent & ready for use
-- =========================================================
BEGIN;

-- ---------- EXTENSIONS ----------
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =========================================================
-- SUBJECTS
-- =========================================================
CREATE TABLE IF NOT EXISTS subjects (
  subjects_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  subjects_code       VARCHAR(40)  NOT NULL,
  subjects_name       VARCHAR(120) NOT NULL,
  subjects_desc       TEXT,
  subjects_slug       VARCHAR(160),

  subjects_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  subjects_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_deleted_at TIMESTAMPTZ
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='subjects' AND column_name='subjects_slug'
  ) THEN
    ALTER TABLE subjects ADD COLUMN subjects_slug VARCHAR(160);
  END IF;
END$$;

-- Backfill slug unik per masjid (soft-delete aware)
WITH base AS (
  SELECT s.subjects_id, s.subjects_masjid_id,
         COALESCE(NULLIF(trim(s.subjects_slug), ''),
                  NULLIF(trim(s.subjects_name), ''),
                  trim(s.subjects_code)) AS raw_src
  FROM subjects s
  WHERE s.subjects_deleted_at IS NULL
),
cand AS (
  SELECT subjects_id, subjects_masjid_id,
         regexp_replace(regexp_replace(lower(raw_src), '[^a-z0-9]+', '-', 'g'), '(^-+|-+$)', '', 'g') AS slug0
  FROM base
),
norm AS (
  SELECT subjects_id, subjects_masjid_id,
         CASE WHEN slug0 IS NULL OR slug0 = ''
              THEN 'subject-' || substring(replace(subjects_id::text,'-','') for 8)
              ELSE slug0 END AS slug1
  FROM cand
),
ranked AS (
  SELECT n.*, ROW_NUMBER() OVER (PARTITION BY n.subjects_masjid_id, n.slug1 ORDER BY n.subjects_id) AS rn
  FROM norm n
),
final_slug AS (
  SELECT subjects_id,
         CASE WHEN rn=1 THEN slug1 ELSE slug1||'-'||rn::text END AS slug_final
  FROM ranked
)
UPDATE subjects s
SET subjects_slug = f.slug_final
FROM final_slug f
WHERE s.subjects_id = f.subjects_id
  AND (s.subjects_slug IS NULL OR trim(s.subjects_slug) = '');

DO $$
BEGIN
  BEGIN
    ALTER TABLE subjects ALTER COLUMN subjects_slug SET NOT NULL;
  EXCEPTION WHEN others THEN NULL;
  END;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_code_not_blank') THEN
    ALTER TABLE subjects ADD CONSTRAINT chk_subjects_code_not_blank CHECK (length(trim(subjects_code)) > 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_slug_not_blank') THEN
    ALTER TABLE subjects ADD CONSTRAINT chk_subjects_slug_not_blank CHECK (length(trim(subjects_slug)) > 0);
  END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_code_per_masjid
  ON subjects (subjects_masjid_id, lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_slug_per_masjid
  ON subjects (subjects_masjid_id, lower(subjects_slug))
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_active
  ON subjects(subjects_masjid_id)
  WHERE subjects_is_active = TRUE AND subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_subjects_name_trgm
  ON subjects USING gin (subjects_name gin_trgm_ops)
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_masjid_alive
  ON subjects(subjects_masjid_id)
  WHERE subjects_deleted_at IS NULL;

CREATE OR REPLACE FUNCTION fn_subjects_normalize()
RETURNS TRIGGER AS $$
DECLARE v_slug text;
BEGIN
  NEW.subjects_code := trim(NEW.subjects_code);
  NEW.subjects_name := trim(NEW.subjects_name);
  IF NEW.subjects_desc IS NOT NULL THEN NEW.subjects_desc := NULLIF(trim(NEW.subjects_desc), ''); END IF;
  IF NEW.subjects_slug IS NOT NULL THEN NEW.subjects_slug := NULLIF(trim(NEW.subjects_slug), ''); END IF;

  IF NEW.subjects_slug IS NULL OR NEW.subjects_slug = '' THEN
    v_slug := COALESCE(NEW.subjects_name, NEW.subjects_code);
    v_slug := lower(regexp_replace(v_slug, '[^a-z0-9]+', '-', 'g'));
    v_slug := regexp_replace(v_slug, '(^-+|-+$)', '', 'g');
    IF v_slug IS NULL OR v_slug = '' THEN
      v_slug := 'subject-' || substring(replace(NEW.subjects_id::text,'-','') for 8);
    END IF;
    NEW.subjects_slug := v_slug;
  END IF;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_normalize') THEN
    DROP TRIGGER trg_subjects_normalize ON subjects;
  END IF;
  CREATE TRIGGER trg_subjects_normalize
    BEFORE INSERT OR UPDATE OF subjects_name, subjects_code, subjects_slug, subjects_desc
    ON subjects FOR EACH ROW EXECUTE FUNCTION fn_subjects_normalize();
END$$;

-- =========================================================
-- CLASS_SUBJECTS
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subjects_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id)   ON DELETE CASCADE,
  class_subjects_class_id   UUID NOT NULL REFERENCES classes(class_id)     ON DELETE CASCADE,
  class_subjects_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,

  class_subjects_term_id UUID,

  class_subjects_order_index       INT,
  class_subjects_hours_per_week    INT,
  class_subjects_min_passing_score INT CHECK (class_subjects_min_passing_score BETWEEN 0 AND 100),
  class_subjects_weight_on_report  INT,
  class_subjects_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subjects_desc              TEXT,

  class_subjects_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_subjects_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_subjects_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_deleted_at  TIMESTAMPTZ
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

-- FK ke academic_terms (jika kolom ada)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_subjects' AND column_name='class_subjects_term_id'
  ) AND NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_cs_term'
  ) THEN
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

-- updated_at trigger
CREATE OR REPLACE FUNCTION trg_set_timestamptz_class_subjects()
RETURNS trigger AS $$
BEGIN
  NEW.class_subjects_updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamptz_class_subjects ON class_subjects;
CREATE TRIGGER set_timestamptz_class_subjects
BEFORE UPDATE ON class_subjects
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamptz_class_subjects();

-- (Opsional kuat) jadikan pair (id, masjid) unik utk validasi tenant downstream
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='uq_class_subjects_id_masjid') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT uq_class_subjects_id_masjid
      UNIQUE (class_subjects_id, class_subjects_masjid_id);
  END IF;
END$$;

-- =========================================================
-- CLASS SECTION SUBJECT TEACHERS (→ class_subjects, manual section_id)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_section_subject_teachers_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- dikirim manual di payload
  class_section_subject_teachers_section_id UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- kunci konteks (kelas+mapel[+term]) → class_subjects
  class_section_subject_teachers_class_subjects_id UUID NOT NULL
    REFERENCES class_subjects(class_subjects_id) ON UPDATE CASCADE ON DELETE CASCADE,

  -- guru → masjid_teachers
  class_section_subject_teachers_teacher_id UUID NOT NULL
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE RESTRICT,

  class_section_subject_teachers_is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_deleted_at  TIMESTAMPTZ NULL
);

-- FK komposit tenant-safe untuk SECTION (section_id + masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_csst_section_masjid') THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT fk_csst_section_masjid
      FOREIGN KEY (
        class_section_subject_teachers_section_id,
        class_section_subject_teachers_masjid_id
      )
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- Unique aktif: satu guru per class_subjects (aktif & belum terhapus)
DROP INDEX IF EXISTS uq_csst_active_unique;
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_active_by_cs
ON class_section_subject_teachers (
  class_section_subject_teachers_class_subjects_id,
  class_section_subject_teachers_teacher_id
)
WHERE class_section_subject_teachers_is_active = TRUE
  AND class_section_subject_teachers_deleted_at IS NULL;

-- Index umum
CREATE INDEX IF NOT EXISTS idx_csst_by_cs_alive
  ON class_section_subject_teachers (class_section_subject_teachers_class_subjects_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_by_section_alive
  ON class_section_subject_teachers (class_section_subject_teachers_section_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_by_teacher_alive
  ON class_section_subject_teachers (class_section_subject_teachers_teacher_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_by_masjid_alive
  ON class_section_subject_teachers (class_section_subject_teachers_masjid_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- Touch updated_at
CREATE OR REPLACE FUNCTION trg_set_timestamp_class_sec_subj_teachers()
RETURNS trigger AS $$
BEGIN
  NEW.class_section_subject_teachers_updated_at = NOW();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;
CREATE TRIGGER set_timestamp_class_sec_subj_teachers
BEFORE UPDATE ON class_section_subject_teachers
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_class_sec_subj_teachers();

-- Validasi konsistensi (bandingkan class_id vs class_id, bukan section_id)
CREATE OR REPLACE FUNCTION fn_csst_validate_consistency()
RETURNS TRIGGER AS $$
DECLARE
  v_cs_masjid    UUID;
  v_cs_class     UUID;
  v_sec_class    UUID;
  v_teacher_mjid UUID;
BEGIN
  -- ambil konteks dari class_subjects
  SELECT class_subjects_masjid_id, class_subjects_class_id
    INTO v_cs_masjid, v_cs_class
  FROM class_subjects
  WHERE class_subjects_id = NEW.class_section_subject_teachers_class_subjects_id
    AND class_subjects_deleted_at IS NULL;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'class_subjects % tidak ditemukan/terhapus',
      NEW.class_section_subject_teachers_class_subjects_id;
  END IF;

  -- ambil class_id dari section (section -> class_id)
  SELECT class_sections_class_id
    INTO v_sec_class
  FROM class_sections
  WHERE class_sections_id = NEW.class_section_subject_teachers_section_id
    AND class_sections_deleted_at IS NULL;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Section % tidak ditemukan/terhapus',
      NEW.class_section_subject_teachers_section_id;
  END IF;

  -- tenant harus sama (row vs class_subjects)
  IF v_cs_masjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: class_subjects(%) != row_masjid(%)',
      v_cs_masjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  -- kelas dari class_subjects harus sama dengan kelas dari section
  IF v_cs_class IS DISTINCT FROM v_sec_class THEN
    RAISE EXCEPTION 'Class mismatch: class_subjects.class_id(%) != class_sections.class_id(%)',
      v_cs_class, v_sec_class;
  END IF;

  -- guru harus milik masjid ini
  SELECT masjid_teacher_masjid_id INTO v_teacher_mjid
  FROM masjid_teachers
  WHERE masjid_teacher_id = NEW.class_section_subject_teachers_teacher_id;

  IF v_teacher_mjid IS NULL THEN
    RAISE EXCEPTION 'Teacher % tidak valid (masjid_teachers)',
      NEW.class_section_subject_teachers_teacher_id;
  END IF;

  IF v_teacher_mjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: teacher(%) != row_masjid(%)',
      v_teacher_mjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_csst_validate_consistency') THEN
    DROP TRIGGER trg_csst_validate_consistency ON class_section_subject_teachers;
  END IF;

  -- DEFERRABLE: aman dipakai saat insert dalam satu transaksi
  CREATE CONSTRAINT TRIGGER trg_csst_validate_consistency
  AFTER INSERT OR UPDATE OF
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_class_subjects_id,
    class_section_subject_teachers_teacher_id
  ON class_section_subject_teachers
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_csst_validate_consistency();
END$$;

-- cleanup artefak lama (jika ada)
DROP TRIGGER IF EXISTS trg_csst_sync_section_from_cs ON class_section_subject_teachers;
DROP FUNCTION IF EXISTS fn_csst_sync_section_from_cs();

COMMIT;
