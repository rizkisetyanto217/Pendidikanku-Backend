-- =========================================================
-- DOWN MIGRATION (Revert UP: restore CSST column; drop URL table; drop new FKs/idx/triggers)
-- =========================================================
BEGIN;

-- -----------------------------------------
-- A. class_attendance_sessions: TRIGGERS/FUNCTIONS (baru) → drop
-- -----------------------------------------

-- 1) Drop constraint trigger & function validator baru
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_validate_links') THEN
    DROP TRIGGER trg_cas_validate_links ON class_attendance_sessions;
  END IF;
END$$;

-- Fungsi bisa dipakai juga oleh deployment lain. Jika aman, drop:
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_cas_validate_links') THEN
    DROP FUNCTION fn_cas_validate_links() CASCADE;
  END IF;
END$$;

-- 2) Trigger touch updated_at: UP tadi create/replace & recreate trigger.
--    Jika kamu ingin mengembalikan ke keadaan sebelum UP, hapus trigger/func ini:
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_touch_updated_at') THEN
    DROP TRIGGER trg_cas_touch_updated_at ON class_attendance_sessions;
  END IF;
END$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_touch_class_attendance_sessions_updated_at') THEN
    DROP FUNCTION fn_touch_class_attendance_sessions_updated_at() CASCADE;
  END IF;
END$$;

-- -----------------------------------------
-- B. class_attendance_sessions: INDEXES/FKs (baru) → drop
-- -----------------------------------------

-- Unique yang baru
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_cas_masjid_section_subject_date') THEN
    EXECUTE 'DROP INDEX uq_cas_masjid_section_subject_date';
  END IF;
END$$;

-- Index biasa yang baru
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_section') THEN
    EXECUTE 'DROP INDEX idx_cas_section';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_masjid') THEN
    EXECUTE 'DROP INDEX idx_cas_masjid';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_date') THEN
    EXECUTE 'DROP INDEX idx_cas_date';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_class_subject') THEN
    EXECUTE 'DROP INDEX idx_cas_class_subject';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_teacher_id') THEN
    EXECUTE 'DROP INDEX idx_cas_teacher_id';
  END IF;
END$$;

-- FKs baru
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_section_masjid_pair') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_section_masjid_pair;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_subject') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_class_subject;
  END IF;
END$$;

-- -----------------------------------------
-- C. class_attendance_sessions: restore CSST column + FK/idx lama
-- -----------------------------------------

-- 1) Tambah kembali kolom CSST (nullable)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_class_section_subject_teacher_id'
  ) THEN
    ALTER TABLE class_attendance_sessions
      ADD COLUMN class_attendance_sessions_class_section_subject_teacher_id UUID;
  END IF;
END$$;

-- 2) Tambah kembali FK ke class_section_subject_teachers (nama constraint disesuaikan)
DO $$
BEGIN
  -- Bersihkan sisa constraint jika ada dengan nama lama
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_csst_old') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_csst_old;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teacher') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_class_section_subject_teacher;
  END IF;

  -- Tambahkan constraint standar
  BEGIN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_section_subject_teacher
      FOREIGN KEY (class_attendance_sessions_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  EXCEPTION WHEN undefined_table THEN
    -- Jika tabel referensi belum ada di environment, biarkan tanpa FK.
    RAISE NOTICE 'Table class_section_subject_teachers tidak ditemukan; FK CSST dilewati.';
  END;
END$$;

-- 3) Index lama untuk CSST
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_cas_csst') THEN
    EXECUTE 'CREATE INDEX idx_cas_csst
             ON class_attendance_sessions(class_attendance_sessions_class_section_subject_teacher_id)';
  END IF;
END$$;

-- 4) (Opsional) Kembalikan unique lama 2-varian (jika memang digunakan sebelumnya)
--    Aktifkan kalau perlu; biarkan non-aktif jika kamu tidak menggunakannya lagi.
-- DO $$
-- BEGIN
--   IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_cas_section_date_when_cs_null') THEN
--     EXECUTE 'CREATE UNIQUE INDEX uq_cas_section_date_when_cs_null
--              ON class_attendance_sessions(class_attendance_sessions_section_id,
--                                           class_attendance_sessions_date)
--              WHERE class_attendance_sessions_deleted_at IS NULL
--                AND class_attendance_sessions_class_subject_id IS NULL';
--   END IF;
--   IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_cas_section_cs_date_when_cs_not_null') THEN
--     EXECUTE '' ||
--       'CREATE UNIQUE INDEX uq_cas_section_cs_date_when_cs_not_null ' ||
--       'ON class_attendance_sessions(class_attendance_sessions_section_id, ' ||
--       '                               class_attendance_sessions_class_subject_id, ' ||
--       '                               class_attendance_sessions_date) ' ||
--       'WHERE class_attendance_sessions_deleted_at IS NULL ' ||
--       '  AND class_attendance_sessions_class_subject_id IS NOT NULL';
--   END IF;
-- END$$;

-- 5) (Opsional) Jika sebelumnya ada trigger validator lama yang mengandalkan CSST,
--    silakan re-create di sini. Karena definisi historisnya bervariasi, bagian ini dikosongkan.
--    Kamu bisa menambahkan kembali sesuai versi lama:
--    CREATE FUNCTION fn_cas_validate_links_legacy() RETURNS TRIGGER AS $$ ... $$ LANGUAGE plpgsql;
--    CREATE CONSTRAINT TRIGGER trg_cas_validate_links AFTER INSERT OR UPDATE OF ... EXECUTE FUNCTION fn_cas_validate_links_legacy();

-- -----------------------------------------
-- D. class_attendance_session_url: drop table + triggers/func/indexes
-- -----------------------------------------

-- 1) Putus trigger-tenant guard & touch
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_casu_tenant_guard') THEN
    DROP TRIGGER trg_casu_tenant_guard ON class_attendance_session_url;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_casu_updated_at') THEN
    DROP TRIGGER trg_touch_casu_updated_at ON class_attendance_session_url;
  END IF;
END$$;

-- 2) Drop functions terkait URL
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_casu_tenant_guard') THEN
    DROP FUNCTION fn_casu_tenant_guard() CASCADE;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_touch_casu_updated_at') THEN
    DROP FUNCTION fn_touch_casu_updated_at() CASCADE;
  END IF;
END$$;

-- 3) Drop indexes URL
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='uq_casu_href_per_session_alive') THEN
    EXECUTE 'DROP INDEX uq_casu_href_per_session_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_casu_session_alive') THEN
    EXECUTE 'DROP INDEX idx_casu_session_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='idx_casu_created_at') THEN
    EXECUTE 'DROP INDEX idx_casu_created_at';
  END IF;
END$$;

-- 4) Drop table URL
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='class_attendance_session_url') THEN
    DROP TABLE class_attendance_session_url;
  END IF;
END$$;

COMMIT;
