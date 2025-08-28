BEGIN;

-- =========================================================
-- ========== DOWN: user_class_attendance_sessions =========
-- =========================================================

-- 1) Drop trigger set_timestamp (jika ada & menempel ke tabel ini)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_trigger t
    WHERE t.tgname = 'set_timestamptz_ucas'
      AND t.tgrelid = 'public.user_class_attendance_sessions'::regclass
  ) THEN
    EXECUTE 'DROP TRIGGER set_timestamptz_ucas ON public.user_class_attendance_sessions';
  END IF;
END$$;

-- 2) Drop UNIQUE constraint + backing index (jika ada)
DO $$
BEGIN
  IF to_regclass('public.user_class_attendance_sessions') IS NOT NULL THEN
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conrelid = 'public.user_class_attendance_sessions'::regclass
        AND conname  = 'uq_ucas_session_userclass'
    ) THEN
      EXECUTE 'ALTER TABLE public.user_class_attendance_sessions DROP CONSTRAINT uq_ucas_session_userclass';
    END IF;
  END IF;
END$$;

DROP INDEX IF EXISTS public.uidx_ucas_session_userclass;

-- 3) Drop indexes (aktif & legacy)
DROP INDEX IF EXISTS public.idx_ucas_masjid_created_at;
DROP INDEX IF EXISTS public.idx_ucas_session_status;
DROP INDEX IF EXISTS public.idx_ucas_userclass_created_at;
DROP INDEX IF EXISTS public.idx_ucas_masjid_session;
DROP INDEX IF EXISTS public.brin_ucas_created_at;
DROP INDEX IF EXISTS public.gin_ucas_search;
DROP INDEX IF EXISTS public.idx_ucas_session_attended;
DROP INDEX IF EXISTS public.idx_ucas_session_absent;

-- Legacy (jaga-jaga nama lama)
DROP INDEX IF EXISTS public.idx_ucae_session_present_only;
DROP INDEX IF EXISTS public.idx_ucas_session_present_only;
DROP INDEX IF EXISTS public.idx_ucae_session_attended;
DROP INDEX IF EXISTS public.idx_ucae_session_absent;
DROP INDEX IF EXISTS public.idx_ucae_session_status;

-- 4) Drop table (akan ikut drop constraint yg menempel)
DROP TABLE IF EXISTS public.user_class_attendance_sessions;

-- 5) Drop function trigger milik UCAS
DROP FUNCTION IF EXISTS public.trg_set_timestamptz_ucas();


-- =========================================================
-- ========== DOWN: class_attendance_sessions ==============
-- =========================================================

-- 1) Drop triggers (validate links + touch updated_at) bila menempel di CAS
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_trigger t
    WHERE t.tgname = 'trg_cas_validate_links'
      AND t.tgrelid = 'public.class_attendance_sessions'::regclass
  ) THEN
    EXECUTE 'DROP TRIGGER trg_cas_validate_links ON public.class_attendance_sessions';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_trigger t
    WHERE t.tgname = 'trg_cas_touch_updated_at'
      AND t.tgrelid = 'public.class_attendance_sessions'::regclass
  ) THEN
    EXECUTE 'DROP TRIGGER trg_cas_touch_updated_at ON public.class_attendance_sessions';
  END IF;
END$$;

-- 2) Drop FKs (idempotent)
DO $$
BEGIN
  IF to_regclass('public.class_attendance_sessions') IS NOT NULL THEN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_section_masjid_pair') THEN
      EXECUTE 'ALTER TABLE public.class_attendance_sessions DROP CONSTRAINT fk_cas_section_masjid_pair';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_subject') THEN
      EXECUTE 'ALTER TABLE public.class_attendance_sessions DROP CONSTRAINT fk_cas_class_subject';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teacher') THEN
      EXECUTE 'ALTER TABLE public.class_attendance_sessions DROP CONSTRAINT fk_cas_class_section_subject_teacher';
    END IF;
  END IF;
END$$;

-- 3) Drop indexes (termasuk partial unique)
DROP INDEX IF EXISTS public.idx_cas_section;
DROP INDEX IF EXISTS public.idx_cas_masjid;
DROP INDEX IF EXISTS public.idx_cas_date;
DROP INDEX IF EXISTS public.idx_cas_class_subject;
DROP INDEX IF EXISTS public.idx_cas_csst;
DROP INDEX IF EXISTS public.idx_cas_teacher_user;

DROP INDEX IF EXISTS public.uq_cas_section_date_when_cs_null;
DROP INDEX IF EXISTS public.uq_cas_section_cs_date_when_cs_not_null;

-- 4) Drop table CAS
DROP TABLE IF EXISTS public.class_attendance_sessions;

-- 5) Drop functions milik CAS
DROP FUNCTION IF EXISTS public.fn_cas_validate_links();
DROP FUNCTION IF EXISTS public.fn_cas_touch_updated_at();
-- ⚠️ SENGAJA: TIDAK drop fn_lectures_touch_updated_at / fungsi lain di luar CAS


COMMIT;
