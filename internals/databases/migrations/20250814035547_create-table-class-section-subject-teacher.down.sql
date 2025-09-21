-- +migrate Down

-- 1) (Opsional, known refs) Lepas FK di tabel lain yang mungkin menunjuk ke CSST
ALTER TABLE IF EXISTS assessments
  DROP CONSTRAINT IF EXISTS fk_assessments_csst,
  DROP CONSTRAINT IF EXISTS fk_assessments_csst_masjid_tenant_safe;

-- 2) (Umum) Lepas semua FK yang mereferensikan tabel yang akan di-drop
DO $$
DECLARE
  r record;
BEGIN
  FOR r IN
    SELECT
      quote_ident(nsp.nspname) || '.' || quote_ident(cls.relname) AS relname,
      conname
    FROM pg_constraint c
    JOIN pg_class     cls ON cls.oid = c.conrelid
    JOIN pg_namespace nsp ON nsp.oid = cls.relnamespace
    WHERE c.contype = 'f'
      AND c.confrelid IN (
        'class_section_subject_teachers'::regclass,
        'user_class_section_subject_teachers'::regclass
      )
  LOOP
    EXECUTE format('ALTER TABLE %s DROP CONSTRAINT IF EXISTS %I', r.relname, r.conname);
  END LOOP;
END$$;

-- 3) Drop tables (index2 ikut terhapus otomatis)
DROP TABLE IF EXISTS user_class_section_subject_teachers;
DROP TABLE IF EXISTS class_section_subject_teachers;
