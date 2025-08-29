-- =========================================================
-- DOWN (SAFE, IDEMPOTENT, DEPENDENCY-AWARE)
-- Target:
--   1) class_section_subject_teachers   (CSST)
--   2) class_subjects                   (CS)
--   3) subjects                         (SUB)
-- Catatan:
--  - Tidak ada BEGIN/COMMIT manual.
--  - Selalu putus FK dari tabel lain ke target sebelum DROP TABLE.
-- =========================================================

-- ========== UTIL: putus semua FK yang menunjuk ke sebuah tabel ==========
-- Gunakan blok ini sebelum DROP TABLE target (ganti 'public.<table_name>').
-- (Tidak perlu diubah; panggil berulang dengan target berbeda.)

-- Putus semua FK yang MEREFERENSIKAN class_section_subject_teachers
DO $$
DECLARE
  target regclass := to_regclass('public.class_section_subject_teachers');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname  AS src_schema,
             c.relname   AS src_table,
             con.conname AS constraint_name
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid        -- tabel sumber (yang punya FK)
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target                          -- menunjuk ke target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.src_schema, r.src_table, r.constraint_name);
    END LOOP;
  END IF;
END$$;

-- Putus semua FK yang MEREFERENSIKAN class_subjects
DO $$
DECLARE
  target regclass := to_regclass('public.class_subjects');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname, c.relname, con.conname
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.nspname, r.relname, r.conname);
    END LOOP;
  END IF;
END$$;

-- Putus semua FK yang MEREFERENSIKAN subjects
DO $$
DECLARE
  target regclass := to_regclass('public.subjects');
  r RECORD;
BEGIN
  IF target IS NOT NULL THEN
    FOR r IN
      SELECT ns.nspname, c.relname, con.conname
      FROM pg_constraint con
      JOIN pg_class      c  ON c.oid = con.conrelid
      JOIN pg_namespace  ns ON ns.oid = c.relnamespace
      WHERE con.contype = 'f'
        AND con.confrelid = target
    LOOP
      EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                     r.nspname, r.relname, r.conname);
    END LOOP;
  END IF;
END$$;

-- =========================================================
-- A) CLASS SECTION SUBJECT TEACHERS (CSST)
-- =========================================================
DO $$
DECLARE
  rel regclass := to_regclass('public.class_section_subject_teachers');
BEGIN
  IF rel IS NOT NULL THEN
    -- TRIGGERS
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_class_sec_subj_teachers_validate_tenant' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER trg_class_sec_subj_teachers_validate_tenant ON %s', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_timestamptz_class_sec_subj_teachers' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER set_timestamptz_class_sec_subj_teachers ON %s', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_timestamp_class_sec_subj_teachers' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER set_timestamp_class_sec_subj_teachers ON %s', rel);
    END IF;

    -- CONSTRAINTS (FK milik CSST)
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_csst_section_masjid' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT fk_csst_section_masjid', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_csst_teacher_membership' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT fk_csst_teacher_membership', rel);
    END IF;

    -- INDEXES (lepas)
    DROP INDEX IF EXISTS public.uq_csst_active_unique;
    DROP INDEX IF EXISTS public.idx_csst_teacher_alive;
    DROP INDEX IF EXISTS public.idx_csst_masjid_alive;
    DROP INDEX IF EXISTS public.idx_csst_section_subject_active_alive;

    -- TABLE
    EXECUTE format('DROP TABLE IF EXISTS %s', rel);
  END IF;
END$$;

-- FUNCTIONS milik CSST
DROP FUNCTION IF EXISTS public.fn_class_sec_subj_teachers_validate_tenant();
DROP FUNCTION IF EXISTS public.trg_set_timestamptz_class_sec_subj_teachers();
DROP FUNCTION IF EXISTS public.trg_set_timestamp_class_sec_subj_teachers();

-- =========================================================
-- B) CLASS SUBJECTS (CS)
-- =========================================================
DO $$
DECLARE
  rel regclass := to_regclass('public.class_subjects');
BEGIN
  IF rel IS NOT NULL THEN
    -- TRIGGERS
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_cs_term_tenant_check' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER trg_cs_term_tenant_check ON %s', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_timestamptz_class_subjects' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER set_timestamptz_class_subjects ON %s', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_timestamp_class_subjects' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER set_timestamp_class_subjects ON %s', rel);
    END IF;

    -- CONSTRAINTS (FK milik CS)
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_class_masjid_pair' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT fk_cs_class_masjid_pair', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cs_term' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT fk_cs_term', rel);
    END IF;

    -- INDEXES
    DROP INDEX IF EXISTS public.uq_class_subjects_by_term;
    DROP INDEX IF EXISTS public.uq_class_subjects;
    DROP INDEX IF EXISTS public.idx_cs_masjid_class_term_active;
    DROP INDEX IF EXISTS public.idx_cs_masjid_subject_term_active;
    DROP INDEX IF EXISTS public.idx_cs_term_alive;
    DROP INDEX IF EXISTS public.idx_cs_masjid_active;
    DROP INDEX IF EXISTS public.idx_cs_class_order;
    DROP INDEX IF EXISTS public.idx_cs_masjid_alive;
    DROP INDEX IF EXISTS public.gin_cs_desc_trgm;

    -- TABLE
    EXECUTE format('DROP TABLE IF EXISTS %s', rel);
  END IF;
END$$;

-- FUNCTIONS milik CS
DROP FUNCTION IF EXISTS public.fn_cs_term_tenant_check();
DROP FUNCTION IF EXISTS public.trg_set_timestamptz_class_subjects();

-- =========================================================
-- C) SUBJECTS (SUB)
-- =========================================================
DO $$
DECLARE
  rel regclass := to_regclass('public.subjects');
BEGIN
  IF rel IS NOT NULL THEN
    -- TRIGGERS
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_touch_updated_at' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER trg_subjects_touch_updated_at ON %s', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_subjects_normalize' AND tgrelid=rel) THEN
      EXECUTE format('DROP TRIGGER trg_subjects_normalize ON %s', rel);
    END IF;

    -- CONSTRAINTS (CHECK)
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_code_not_blank' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT chk_subjects_code_not_blank', rel);
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_subjects_slug_not_blank' AND conrelid=rel) THEN
      EXECUTE format('ALTER TABLE %s DROP CONSTRAINT chk_subjects_slug_not_blank', rel);
    END IF;

    -- INDEXES
    DROP INDEX IF EXISTS public.uq_subjects_code_per_masjid;
    DROP INDEX IF EXISTS public.uq_subjects_slug_per_masjid;
    DROP INDEX IF EXISTS public.idx_subjects_active;
    DROP INDEX IF EXISTS public.gin_subjects_name_trgm;
    DROP INDEX IF EXISTS public.idx_subjects_masjid_alive;
    DROP INDEX IF EXISTS public.idx_subjects_code_ci_alive;
    DROP INDEX IF EXISTS public.idx_subjects_slug_ci_alive;

    -- TABLE
    EXECUTE format('DROP TABLE IF EXISTS %s', rel);
  END IF;
END$$;

-- FUNCTIONS milik SUB
DROP FUNCTION IF EXISTS public.fn_subjects_touch_updated_at();
DROP FUNCTION IF EXISTS public.fn_subjects_normalize();
