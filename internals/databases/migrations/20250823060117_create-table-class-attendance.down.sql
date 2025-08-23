BEGIN;

-- =========================================================
-- DOWN: user_class_attendance_sessions (CHILD)
-- =========================================================
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='user_class_attendance_sessions'
  ) THEN
    -- Triggers
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_timestamp_ucas') THEN
      DROP TRIGGER set_timestamp_ucas ON user_class_attendance_sessions;
    END IF;

    -- Constraint (UNIQUE USING INDEX) + unique index
    IF EXISTS (
      SELECT 1 FROM pg_constraint
      WHERE conrelid='user_class_attendance_sessions'::regclass
        AND conname='uq_ucas_session_userclass'
    ) THEN
      ALTER TABLE user_class_attendance_sessions
        DROP CONSTRAINT uq_ucas_session_userclass;
    END IF;
    DROP INDEX IF EXISTS uidx_ucas_session_userclass;

    -- Indexes (aktif)
    DROP INDEX IF EXISTS idx_ucas_masjid_created_at;
    DROP INDEX IF EXISTS idx_ucas_session_status;
    DROP INDEX IF EXISTS idx_ucas_userclass_created_at;
    DROP INDEX IF EXISTS idx_ucas_masjid_session;
    DROP INDEX IF EXISTS brin_ucas_created_at;
    DROP INDEX IF EXISTS gin_ucas_search;
    DROP INDEX IF EXISTS idx_ucas_session_attended;
    DROP INDEX IF EXISTS idx_ucas_session_absent;

    -- Indexes legacy (jaga-jaga)
    DROP INDEX IF EXISTS idx_ucae_session_present_only;
    DROP INDEX IF EXISTS idx_ucas_session_present_only;
    DROP INDEX IF EXISTS idx_ucae_session_attended;
    DROP INDEX IF EXISTS idx_ucae_session_absent;
    DROP INDEX IF EXISTS idx_ucae_session_status;

    -- Table
    DROP TABLE user_class_attendance_sessions;
  END IF;
END$$;

-- Function helper untuk updated_at (khusus UCAS)
DROP FUNCTION IF EXISTS trg_set_timestamp_ucas();

-- =========================================================
-- DOWN: class_attendance_sessions (PARENT)
-- =========================================================
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='class_attendance_sessions'
  ) THEN
    -- Triggers
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_cas_validate_links') THEN
      DROP TRIGGER trg_cas_validate_links ON class_attendance_sessions;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_cas_touch_updated_at') THEN
      DROP TRIGGER trg_cas_touch_updated_at ON class_attendance_sessions;
    END IF;

    -- Foreign Keys
    IF EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname='fk_cas_section_masjid_pair'
    ) THEN
      ALTER TABLE class_attendance_sessions
        DROP CONSTRAINT fk_cas_section_masjid_pair;
    END IF;

    IF EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_subject'
    ) THEN
      ALTER TABLE class_attendance_sessions
        DROP CONSTRAINT fk_cas_class_subject;
    END IF;

    IF EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teacher'
    ) THEN
      ALTER TABLE class_attendance_sessions
        DROP CONSTRAINT fk_cas_class_section_subject_teacher;
    END IF;

    -- Indexes (termasuk unik partial)
    DROP INDEX IF EXISTS idx_cas_section;
    DROP INDEX IF EXISTS idx_cas_masjid;
    DROP INDEX IF EXISTS idx_cas_date;
    DROP INDEX IF EXISTS idx_cas_class_subject;
    DROP INDEX IF EXISTS idx_cas_csst;
    DROP INDEX IF EXISTS idx_cas_teacher_user;

    DROP INDEX IF EXISTS uq_cas_section_date_when_cs_null;
    DROP INDEX IF EXISTS uq_cas_section_cs_date_when_cs_not_null;

    -- Table
    DROP TABLE class_attendance_sessions;
  END IF;
END$$;

-- Functions yang dipakai trigger/constraint milik CAS
DROP FUNCTION IF EXISTS fn_cas_validate_links();
-- JANGAN drop fungsi global yang mungkin dipakai tabel lain:
-- DROP FUNCTION IF EXISTS fn_touch_updated_at();

-- Opsional: kalau kamu sudah migrasi UP ke fungsi khusus CAS,
-- kita bisa drop juga kalau ada:
DROP FUNCTION IF EXISTS fn_cas_touch_updated_at();

COMMIT;
