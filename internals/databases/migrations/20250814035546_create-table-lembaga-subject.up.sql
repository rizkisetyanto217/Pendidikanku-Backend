-- =========================================================
-- FRESH INSTALL: Subjects, Class-Subjects, CSST, Attendance Sessions
-- (Idempotent: aman di-run berkali-kali)
-- =========================================================

BEGIN;

-- =========================================================
-- SUBJECTS (timestamp only, slug unik per masjid)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- ---------- TABLE (fresh install) ----------
CREATE TABLE IF NOT EXISTS subjects (
  subjects_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  subjects_code       VARCHAR(40)  NOT NULL,
  subjects_name       VARCHAR(120) NOT NULL,
  subjects_desc       TEXT,

  -- NEW: slug (URL-friendly)
  subjects_slug       VARCHAR(160),

  subjects_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  subjects_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subjects_deleted_at TIMESTAMPTZ
);

-- ---------- MIGRASI: tambah kolom slug jika belum ada ----------
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='subjects' AND column_name='subjects_slug'
  ) THEN
    ALTER TABLE subjects ADD COLUMN subjects_slug VARCHAR(160);
  END IF;
END$$;

-- ---------- BACKFILL SLUG (unik per masjid, soft-delete aware) ----------
WITH base AS (
  SELECT
    s.subjects_id,
    s.subjects_masjid_id,
    COALESCE(NULLIF(trim(s.subjects_slug), ''),
             NULLIF(trim(s.subjects_name), ''),
             trim(s.subjects_code)) AS raw_src
  FROM subjects s
  WHERE s.subjects_deleted_at IS NULL
),
cand AS (
  SELECT
    subjects_id,
    subjects_masjid_id,
    regexp_replace(
      regexp_replace(lower(raw_src), '[^a-z0-9]+', '-', 'g'),
      '(^-+|-+$)', '', 'g'
    ) AS slug0
  FROM base
),
norm AS (
  SELECT
    c.subjects_id,
    c.subjects_masjid_id,
    CASE
      WHEN c.slug0 IS NULL OR c.slug0 = '' THEN
        'subject-' || substring(replace(c.subjects_id::text,'-','') for 8)
      ELSE c.slug0
    END AS slug1
  FROM cand c
),
ranked AS (
  SELECT
    n.*,
    ROW_NUMBER() OVER (
      PARTITION BY n.subjects_masjid_id, n.slug1
      ORDER BY n.subjects_id
    ) AS rn
  FROM norm n
),
final_slug AS (
  SELECT
    subjects_id,
    CASE WHEN rn = 1 THEN slug1 ELSE slug1 || '-' || rn::text END AS slug_final
  FROM ranked
)
UPDATE subjects s
SET subjects_slug = f.slug_final
FROM final_slug f
WHERE s.subjects_id = f.subjects_id
  AND (s.subjects_slug IS NULL OR trim(s.subjects_slug) = '');

-- Jadikan NOT NULL setelah backfill (best-effort)
DO $$
BEGIN
  BEGIN
    ALTER TABLE subjects
      ALTER COLUMN subjects_slug SET NOT NULL;
  EXCEPTION WHEN others THEN
    NULL;
  END;
END$$;

-- ---------- CHECK ringan ----------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_code_not_blank') THEN
    ALTER TABLE subjects
      ADD CONSTRAINT chk_subjects_code_not_blank
      CHECK (length(trim(subjects_code)) > 0);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_slug_not_blank') THEN
    ALTER TABLE subjects
      ADD CONSTRAINT chk_subjects_slug_not_blank
      CHECK (length(trim(subjects_slug)) > 0);
  END IF;
END$$;

-- ---------- INDEXES ----------
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

CREATE INDEX IF NOT EXISTS idx_subjects_code_ci_alive
  ON subjects (lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_slug_ci_alive
  ON subjects (lower(subjects_slug))
  WHERE subjects_deleted_at IS NULL;

-- ---------- TRIGGERS ----------
CREATE OR REPLACE FUNCTION fn_subjects_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.subjects_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_touch_updated_at') THEN
    DROP TRIGGER trg_subjects_touch_updated_at ON subjects;
  END IF;

  CREATE TRIGGER trg_subjects_touch_updated_at
    BEFORE UPDATE ON subjects
    FOR EACH ROW
    EXECUTE FUNCTION fn_subjects_touch_updated_at();
END$$;

CREATE OR REPLACE FUNCTION fn_subjects_normalize()
RETURNS TRIGGER AS $$
DECLARE v_slug text;
BEGIN
  NEW.subjects_code := trim(NEW.subjects_code);
  NEW.subjects_name := trim(NEW.subjects_name);
  IF NEW.subjects_desc IS NOT NULL THEN
    NEW.subjects_desc := NULLIF(trim(NEW.subjects_desc), '');
  END IF;
  IF NEW.subjects_slug IS NOT NULL THEN
    NEW.subjects_slug := NULLIF(trim(NEW.subjects_slug), '');
  END IF;

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
    ON subjects
    FOR EACH ROW
    EXECUTE FUNCTION fn_subjects_normalize();
END$$;

-- ===== Cleanup jejak lama berbasis academic_year (jika ada) =====
DROP INDEX IF EXISTS idx_cs_masjid_class_year_active;
DROP INDEX IF EXISTS idx_cs_masjid_subject_year_active;

-- ===== Tabel class_subjects (fresh install friendly) =====
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subjects_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id)   ON DELETE CASCADE,
  class_subjects_class_id   UUID NOT NULL REFERENCES classes(class_id)     ON DELETE CASCADE,
  class_subjects_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,

  -- RELASI KE ACADEMIC TERMS (opsional per semester)
  class_subjects_term_id UUID,

  -- metadata kurikulum
  class_subjects_order_index       INT,
  class_subjects_hours_per_week    INT,
  class_subjects_min_passing_score INT CHECK (class_subjects_min_passing_score BETWEEN 0 AND 100),
  class_subjects_weight_on_report  INT,
  class_subjects_is_core           BOOLEAN NOT NULL DEFAULT FALSE,
  class_subjects_desc              TEXT,

  class_subjects_is_active   BOOLEAN   NOT NULL DEFAULT TRUE,
  class_subjects_created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_subjects_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subjects_deleted_at  TIMESTAMPTZ
);

-- Pastikan kolom academic_year sudah di-drop (kalau skema lama masih punya)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_subjects'
      AND column_name='class_subjects_academic_year'
  ) THEN
    ALTER TABLE class_subjects DROP COLUMN class_subjects_academic_year;
  END IF;
END$$;

-- ===== FK komposit (tenant-safe) ke classes (class_id, masjid_id) =====
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

-- ===== FK ke academic_terms pada kolom class_subjects_term_id (beri nama eksplisit) =====
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

-- ===== Tenant-check: term harus milik masjid yang sama =====
CREATE OR REPLACE FUNCTION fn_cs_term_tenant_check()
RETURNS TRIGGER AS $$
DECLARE v_term_masjid UUID;
BEGIN
  IF NEW.class_subjects_term_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT academic_terms_masjid_id
    INTO v_term_masjid
  FROM academic_terms
  WHERE academic_terms_id = NEW.class_subjects_term_id
    AND academic_terms_deleted_at IS NULL;

  IF v_term_masjid IS NULL THEN
    RAISE EXCEPTION 'academic_term % tidak ditemukan/terhapus', NEW.class_subjects_term_id;
  END IF;

  IF v_term_masjid <> NEW.class_subjects_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: term(%) != class_subjects(%)',
      v_term_masjid, NEW.class_subjects_masjid_id;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cs_term_tenant_check') THEN
    DROP TRIGGER trg_cs_term_tenant_check ON class_subjects;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_cs_term_tenant_check
  AFTER INSERT OR UPDATE OF class_subjects_masjid_id, class_subjects_term_id
  ON class_subjects
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_cs_term_tenant_check();
END$$;

-- ===== UNIQUE soft-delete aware (berbasis TERM saja) =====
DROP INDEX IF EXISTS uq_class_subjects_by_term;
DROP INDEX IF EXISTS uq_class_subjects;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subjects_by_term
ON class_subjects (
  class_subjects_masjid_id,
  class_subjects_class_id,
  class_subjects_subject_id,
  COALESCE(class_subjects_term_id::text,'')
)
WHERE class_subjects_deleted_at IS NULL;

-- ===== Indexes (alive / active) =====
CREATE INDEX IF NOT EXISTS idx_cs_masjid_class_term_active
  ON class_subjects (
    class_subjects_masjid_id,
    class_subjects_class_id,
    COALESCE(class_subjects_term_id::text,'')
  )
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cs_masjid_subject_term_active
  ON class_subjects (
    class_subjects_masjid_id,
    class_subjects_subject_id,
    COALESCE(class_subjects_term_id::text,'')
  )
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cs_term_alive
  ON class_subjects (class_subjects_term_id)
  WHERE class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cs_masjid_active
  ON class_subjects (class_subjects_masjid_id)
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cs_class_order
  ON class_subjects (class_subjects_class_id, class_subjects_order_index)
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cs_masjid_alive
  ON class_subjects (class_subjects_masjid_id)
  WHERE class_subjects_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS gin_cs_desc_trgm
  ON class_subjects USING gin (class_subjects_desc gin_trgm_ops)
  WHERE class_subjects_deleted_at IS NULL;

-- ===== Trigger updated_at =====
CREATE OR REPLACE FUNCTION trg_set_timestamptz_class_subjects()
RETURNS trigger AS $$
BEGIN
  NEW.class_subjects_updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamptz_class_subjects ON class_subjects;
CREATE TRIGGER set_timestamptz_class_subjects
BEFORE UPDATE ON class_subjects
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamptz_class_subjects();

-- =========================================================
-- CLASS SECTION SUBJECT TEACHERS (soft delete friendly)
-- =========================================================

-- 1) TABLE
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_section_subject_teachers_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_section_subject_teachers_section_id UUID NOT NULL REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,
  class_section_subject_teachers_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,
  class_section_subject_teachers_teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

  class_section_subject_teachers_is_active   BOOLEAN   NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_section_subject_teachers_deleted_at  TIMESTAMPTZ NULL
);

-- 1b) Tambah kolom deleted_at jika belum ada (idempotent)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='class_section_subject_teachers'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='class_section_subject_teachers'
      AND column_name='class_section_subject_teachers_deleted_at'
  ) THEN
    ALTER TABLE class_section_subject_teachers
      ADD COLUMN class_section_subject_teachers_deleted_at TIMESTAMPTZ;
  END IF;
END$$;

-- 2) TENANT-SAFE FK: (section_id, masjid_id) → class_sections(id, masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_csst_section_masjid'
  ) THEN
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

-- 2.9) PRECONDITION untuk FK ke masjid_teachers:
--      wajib UNIQUE penuh di (masjid_teacher_masjid_id, masjid_teacher_user_id)
DO $$
DECLARE
  _dup_exists boolean;
BEGIN
  -- cek duplikat pasangan di SELURUH baris (termasuk soft-deleted)
  SELECT EXISTS (
    SELECT 1
    FROM masjid_teachers
    GROUP BY masjid_teacher_masjid_id, masjid_teacher_user_id
    HAVING COUNT(*) > 1
  ) INTO _dup_exists;

  IF _dup_exists THEN
    RAISE EXCEPTION
      'Tidak bisa tambah UNIQUE (masjid_teacher_masjid_id, masjid_teacher_user_id) di masjid_teachers: ada duplikat. Bersihkan dulu (termasuk baris soft-deleted).';
  END IF;

  -- tambah UNIQUE penuh bila belum ada
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_masjid_teachers_membership'
  ) THEN
    ALTER TABLE masjid_teachers
      ADD CONSTRAINT uq_masjid_teachers_membership
      UNIQUE (masjid_teacher_masjid_id, masjid_teacher_user_id);
  END IF;
END$$;

-- 3) TENANT-SAFE MEMBERSHIP FK → masjid_teachers (kolom yang benar)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_csst_teacher_membership'
  ) THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT fk_csst_teacher_membership
      FOREIGN KEY (
        class_section_subject_teachers_masjid_id,
        class_section_subject_teachers_teacher_user_id
      )
      REFERENCES masjid_teachers (masjid_teacher_masjid_id, masjid_teacher_user_id)
      ON UPDATE CASCADE
      ON DELETE RESTRICT;
  END IF;
END$$;

-- 4) UNIQUE aktif: cegah duplikasi assignment aktif
CREATE UNIQUE INDEX IF NOT EXISTS uq_csst_active_unique
ON class_section_subject_teachers (
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_subject_id,
  class_section_subject_teachers_teacher_user_id
)
WHERE class_section_subject_teachers_is_active = TRUE
  AND class_section_subject_teachers_deleted_at IS NULL;

-- 5) INDEX umum (soft-delete aware)
CREATE INDEX IF NOT EXISTS idx_csst_teacher_alive
  ON class_section_subject_teachers (class_section_subject_teachers_teacher_user_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_masjid_alive
  ON class_section_subject_teachers (class_section_subject_teachers_masjid_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_csst_section_subject_active_alive
  ON class_section_subject_teachers (
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_subject_id
  )
  WHERE class_section_subject_teachers_is_active = TRUE
    AND class_section_subject_teachers_deleted_at IS NULL;

-- 6) TRIGGER updated_at
CREATE OR REPLACE FUNCTION trg_set_timestamp_class_sec_subj_teachers()
RETURNS trigger AS $$
BEGIN
  NEW.class_section_subject_teachers_updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;
CREATE TRIGGER set_timestamp_class_sec_subj_teachers
BEFORE UPDATE ON class_section_subject_teachers
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_class_sec_subj_teachers();

-- 7) VALIDASI TENANT
CREATE OR REPLACE FUNCTION fn_class_sec_subj_teachers_validate_tenant()
RETURNS TRIGGER AS $BODY$
DECLARE
  v_sec_masjid UUID;
  v_sub_masjid UUID;
  has_sec_deleted_at BOOLEAN := FALSE;
  has_sub_deleted_at BOOLEAN := FALSE;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='class_sections' AND column_name='class_sections_deleted_at'
  ) INTO has_sec_deleted_at;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='subjects' AND column_name='subjects_deleted_at'
  ) INTO has_sub_deleted_at;

  -- validasi section
  IF has_sec_deleted_at THEN
    SELECT class_sections_masjid_id INTO v_sec_masjid
    FROM class_sections
    WHERE class_sections_id = NEW.class_section_subject_teachers_section_id
      AND class_sections_deleted_at IS NULL;
  ELSE
    SELECT class_sections_masjid_id INTO v_sec_masjid
    FROM class_sections
    WHERE class_sections_id = NEW.class_section_subject_teachers_section_id;
  END IF;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Section % tidak ditemukan / sudah dihapus', NEW.class_section_subject_teachers_section_id;
  END IF;
  IF v_sec_masjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: section(%) != row_masjid(%)',
      v_sec_masjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  -- validasi subject
  IF has_sub_deleted_at THEN
    SELECT subjects_masjid_id INTO v_sub_masjid
    FROM subjects
    WHERE subjects_id = NEW.class_section_subject_teachers_subject_id
      AND subjects_deleted_at IS NULL;
  ELSE
    SELECT subjects_masjid_id INTO v_sub_masjid
    FROM subjects
    WHERE subjects_id = NEW.class_section_subject_teachers_subject_id;
  END IF;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Subject % tidak ditemukan / sudah dihapus', NEW.class_section_subject_teachers_subject_id;
  END IF;
  IF v_sub_masjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: subject(%) != row_masjid(%)',
      v_sub_masjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  RETURN NEW;
END
$BODY$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sec_subj_teachers_validate_tenant') THEN
    DROP TRIGGER trg_class_sec_subj_teachers_validate_tenant ON class_section_subject_teachers;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_class_sec_subj_teachers_validate_tenant
  AFTER INSERT OR UPDATE OF
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_subject_id
  ON class_section_subject_teachers
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_class_sec_subj_teachers_validate_tenant();
END$$;

COMMIT;
