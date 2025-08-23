BEGIN;

-- =========================================================
-- DOWN: lembaga_stats
--   - drop trigger & function updated_at
--   - drop check non-neg
--   - drop index updated_at
-- =========================================================

-- 1) Triggers & functions
DROP TRIGGER  IF EXISTS trg_lembaga_stats_touch_updated_at ON lembaga_stats;
DROP FUNCTION IF EXISTS fn_lembaga_stats_touch_updated_at();

-- 2) Check constraint
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_lembaga_stats_nonneg') THEN
    ALTER TABLE lembaga_stats DROP CONSTRAINT chk_lembaga_stats_nonneg;
  END IF;
END$$;

-- 3) Indexes
DROP INDEX IF EXISTS idx_lembaga_stats_updated_at;



-- =========================================================
-- DOWN: user_class_attendance_semester_stats
--   - drop constraint triggers & functions
--   - drop unique & supporting indexes
--   - drop FK ke academic_terms (nama auto) lalu drop kolom term_id
--   - drop trigger & function updated_at
-- =========================================================

-- 1) Constraint trigger & function (validasi tenant/term)
DROP TRIGGER  IF EXISTS trg_ucass_validate_links ON user_class_attendance_semester_stats;
DROP FUNCTION IF EXISTS fn_ucass_validate_links();

-- 2) Uniques & supporting indexes
DROP INDEX IF EXISTS uq_ucass_tenant_userclass_section_term;
DROP INDEX IF EXISTS uq_ucass_tenant_userclass_section_period;
DROP INDEX IF EXISTS ix_ucass_masjid_term;
DROP INDEX IF EXISTS ix_ucass_term;
DROP INDEX IF EXISTS ix_ucass_masjid_section_period;
DROP INDEX IF EXISTS ix_ucass_userclass;

-- 3) Hapus FK ke academic_terms (nama constraint auto → cari & drop)
DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN
    SELECT conname
    FROM pg_constraint
    WHERE conrelid = 'user_class_attendance_semester_stats'::regclass
      AND contype = 'f'
      AND pg_get_constraintdef(oid) ILIKE '%(user_class_attendance_semester_stats_term_id)%'
  LOOP
    EXECUTE format(
      'ALTER TABLE user_class_attendance_semester_stats DROP CONSTRAINT %I',
      r.conname
    );
  END LOOP;
END$$;

-- 4) (Opsional) drop kolom term_id bila ingin benar-benar kembali ke desain lama
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='user_class_attendance_semester_stats'
      AND column_name='user_class_attendance_semester_stats_term_id'
  ) THEN
    ALTER TABLE user_class_attendance_semester_stats
      DROP COLUMN user_class_attendance_semester_stats_term_id;
  END IF;
END$$;

-- 5) Trigger & function updated_at
DROP TRIGGER  IF EXISTS trg_ucass_touch_updated_at ON user_class_attendance_semester_stats;
DROP FUNCTION IF EXISTS fn_ucass_touch_updated_at();



-- =========================================================
-- (Opsional) HARD DROP TABLES — kalau memang ingin bersih total
--   *Non-aktif secara default; hapus komentar jika dibutuhkan*
-- =========================================================
DROP TABLE IF EXISTS user_class_attendance_semester_stats CASCADE;
DROP TABLE IF EXISTS lembaga_stats CASCADE;




COMMIT;
