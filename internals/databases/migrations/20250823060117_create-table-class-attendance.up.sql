
-- =========================================================
-- 1) TABLE
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_attendance_sessions_section_id UUID NOT NULL,
  class_attendance_sessions_masjid_id  UUID NOT NULL,

  -- Kurikulum: refer ke class_subjects (boleh NULL saat awal input)
  class_attendance_sessions_class_subject_id UUID,

  -- Penugasan guru per section+subject (opsional)
  class_attendance_sessions_class_section_subject_teacher_id UUID,

  class_attendance_sessions_date  DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note  TEXT,

  class_attendance_sessions_teacher_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  class_attendance_sessions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- 2) FOREIGN KEYS
-- =========================================================

-- (a) Tenant-safe: composite FK ke class_sections(id, masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_section_masjid_pair'
  ) THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_section_masjid_pair
      FOREIGN KEY (class_attendance_sessions_section_id, class_attendance_sessions_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- (b) Kurikulum: FK ke class_subjects (bukan subjects langsung)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_subject') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_subject;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_class_subject') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_subject
      FOREIGN KEY (class_attendance_sessions_class_subject_id)
      REFERENCES class_subjects(class_subjects_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- (c) Penugasan guru: FK ke class_section_subject_teachers
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_csst_old') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_csst_old;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teacher'
  ) THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_section_subject_teacher
      FOREIGN KEY (class_attendance_sessions_class_section_subject_teacher_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================================================
-- 3) INDEXES (termasuk unik soft-delete aware)
-- =========================================================

-- Query umum
CREATE INDEX IF NOT EXISTS idx_cas_section
  ON class_attendance_sessions(class_attendance_sessions_section_id);

CREATE INDEX IF NOT EXISTS idx_cas_masjid
  ON class_attendance_sessions(class_attendance_sessions_masjid_id);

CREATE INDEX IF NOT EXISTS idx_cas_date
  ON class_attendance_sessions(class_attendance_sessions_date DESC);

CREATE INDEX IF NOT EXISTS idx_cas_class_subject
  ON class_attendance_sessions(class_attendance_sessions_class_subject_id);

CREATE INDEX IF NOT EXISTS idx_cas_csst
  ON class_attendance_sessions(class_attendance_sessions_class_section_subject_teacher_id);

CREATE INDEX IF NOT EXISTS idx_cas_teacher_user
  ON class_attendance_sessions(class_attendance_sessions_teacher_user_id);

-- Unik: jika class_subject_id IS NULL → unik per (masjid, section, date)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_section_date_when_cs_null
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_class_subject_id IS NULL
    AND class_attendance_sessions_deleted_at IS NULL;

-- Unik: jika class_subject_id NOT NULL → unik per (masjid, section, class_subject, date)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_section_cs_date_when_cs_not_null
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_class_subject_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_class_subject_id IS NOT NULL
    AND class_attendance_sessions_deleted_at IS NULL;

-- =========================================================
-- 4) TRIGGERS: validasi konsistensi relasi (DEFERRABLE)
--    - pastikan masjid session = masjid section = masjid class_subject
--    - pastikan class_subject.class_id sama dengan section.class_id (jika kolom class_id ada)
--    - pastikan CSS teacher cocok: masjid & section sama, dan subject-nya match class_subject
--    - auto-isi teacher_user_id dari CSS teacher jika NULL
-- =========================================================

CREATE OR REPLACE FUNCTION fn_cas_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_sec_masjid UUID;
  v_sec_class  UUID;        -- boleh NULL jika tidak ada kolom class_id
  v_cs_masjid  UUID;
  v_cs_class   UUID;
  v_cs_subject UUID;
  v_css_masjid UUID;
  v_css_sec    UUID;
  v_css_subj   UUID;
  v_css_teacher UUID;
BEGIN
  -- 1) Section → masjid & (opsional) class_id
  SELECT class_sections_masjid_id, class_sections_class_id
    INTO v_sec_masjid, v_sec_class
  FROM class_sections
  WHERE class_sections_id = NEW.class_attendance_sessions_section_id
    AND class_sections_deleted_at IS NULL;

  IF v_sec_masjid IS NULL THEN
    RAISE EXCEPTION 'Section invalid/terhapus';
  END IF;

  -- 2) Cocokkan masjid session vs section
  IF NEW.class_attendance_sessions_masjid_id <> v_sec_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: session(%) vs section(%)',
      NEW.class_attendance_sessions_masjid_id, v_sec_masjid;
  END IF;

  -- 3) Class_subject (opsional): cek masjid & (opsional) class_id
  IF NEW.class_attendance_sessions_class_subject_id IS NOT NULL THEN
    SELECT class_subjects_masjid_id, class_subjects_class_id, class_subjects_subject_id
      INTO v_cs_masjid, v_cs_class, v_cs_subject
    FROM class_subjects
    WHERE class_subjects_id = NEW.class_attendance_sessions_class_subject_id
      AND class_subjects_deleted_at IS NULL;

    IF v_cs_masjid IS NULL THEN
      RAISE EXCEPTION 'Class subject invalid/terhapus';
    END IF;

    IF v_cs_masjid <> NEW.class_attendance_sessions_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: class_subject(%) vs session(%)',
        v_cs_masjid, NEW.class_attendance_sessions_masjid_id;
    END IF;

    -- Jika kedua sisi punya class_id, pastikan sama
    IF v_sec_class IS NOT NULL AND v_cs_class IS NOT NULL AND v_sec_class <> v_cs_class THEN
      RAISE EXCEPTION 'class_subject.class_id berbeda dengan section.class_id';
    END IF;
  END IF;

  -- 4) CSS Teacher (opsional): cek masjid, section, subject harus match
  IF NEW.class_attendance_sessions_class_section_subject_teacher_id IS NOT NULL THEN
    SELECT
      class_section_subject_teachers_masjid_id,
      class_section_subject_teachers_section_id,
      class_section_subject_teachers_subject_id,
      class_section_subject_teachers_teacher_user_id
    INTO v_css_masjid, v_css_sec, v_css_subj, v_css_teacher
    FROM class_section_subject_teachers
    WHERE class_section_subject_teachers_id = NEW.class_attendance_sessions_class_section_subject_teacher_id
      AND class_section_subject_teachers_deleted_at IS NULL;

    IF v_css_masjid IS NULL THEN
      RAISE EXCEPTION 'CSS teacher invalid/terhapus';
    END IF;

    IF v_css_masjid <> NEW.class_attendance_sessions_masjid_id THEN
      RAISE EXCEPTION 'Masjid CSS(%) != session(%)', v_css_masjid, NEW.class_attendance_sessions_masjid_id;
    END IF;

    IF v_css_sec <> NEW.class_attendance_sessions_section_id THEN
      RAISE EXCEPTION 'Section CSS(%) != session(%)', v_css_sec, NEW.class_attendance_sessions_section_id;
    END IF;

    -- Jika class_subject diisi, pastikan subject CSS sama dengan subject milik class_subject
    IF NEW.class_attendance_sessions_class_subject_id IS NOT NULL THEN
      IF v_cs_subject IS NULL THEN
        -- ambil ulang jika belum diisi dari langkah (3)
        SELECT class_subjects_subject_id INTO v_cs_subject
        FROM class_subjects
        WHERE class_subjects_id = NEW.class_attendance_sessions_class_subject_id;
      END IF;

      IF v_css_subj <> v_cs_subject THEN
        RAISE EXCEPTION 'Subject CSS(%) != class_subject(%)', v_css_subj, v_cs_subject;
      END IF;
    END IF;

    -- Auto-isi teacher_user_id dari CSS Teacher bila belum diset
    IF NEW.class_attendance_sessions_teacher_user_id IS NULL THEN
      NEW.class_attendance_sessions_teacher_user_id := v_css_teacher;
    END IF;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Buat constraint trigger (DEFERRABLE)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_validate_links') THEN
    DROP TRIGGER trg_cas_validate_links ON class_attendance_sessions;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_cas_validate_links
    AFTER INSERT OR UPDATE OF
      class_attendance_sessions_masjid_id,
      class_attendance_sessions_section_id,
      class_attendance_sessions_class_subject_id,
      class_attendance_sessions_class_section_subject_teacher_id,
      class_attendance_sessions_teacher_user_id
    ON class_attendance_sessions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_cas_validate_links();
END$$;

-- =========================================================
-- 5) (Opsional) updated_at auto
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_class_attendance_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_attendance_sessions_updated_at := CURRENT_TIMESTAMPTZ;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_touch_updated_at') THEN
    DROP TRIGGER trg_cas_touch_updated_at ON class_attendance_sessions;
  END IF;

  CREATE TRIGGER trg_cas_touch_updated_at
    BEFORE UPDATE ON class_attendance_sessions
    FOR EACH ROW
    EXECUTE FUNCTION fn_touch_class_attendance_sessions_updated_at();
END$$;
