-- =========================================================
-- UP MIGRATION
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- =========================================================
-- 1) TABEL class_attendance_sessions (dengan perubahan relasi ke masjid_teachers)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_attendance_sessions_section_id UUID NOT NULL,
  class_attendance_sessions_masjid_id  UUID NOT NULL,

  -- Kurikulum (opsional)
  class_attendance_sessions_class_subject_id UUID,

  -- Penugasan guru (opsional)
  class_attendance_sessions_class_section_subject_teacher_id UUID,

  class_attendance_sessions_date  DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note  TEXT,

  -- GANTI: refer ke masjid_teachers, bukan users
  class_attendance_sessions_teacher_id UUID REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  class_attendance_sessions_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_sessions_deleted_at TIMESTAMPTZ
);

-- =========================================================
-- 2) FOREIGN KEYS (idempotent)
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

-- (b) Kurikulum: FK ke class_subjects
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
-- 3) INDEXES (soft-delete aware)
-- =========================================================
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
  ON class_attendance_sessions(class_attendance_sessions_teacher_id);

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
-- =========================================================
CREATE OR REPLACE FUNCTION fn_cas_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_sec_masjid UUID;
  v_sec_class  UUID;
  v_cs_masjid  UUID;
  v_cs_class   UUID;
  v_cs_subject UUID;
  v_css_masjid UUID;
  v_css_sec    UUID;
  v_css_subj   UUID;
  v_css_teacher UUID;
BEGIN
  -- Section
  SELECT class_sections_masjid_id, class_sections_class_id
    INTO v_sec_masjid, v_sec_class
  FROM class_sections
  WHERE class_sections_id = NEW.class_attendance_sessions_section_id
    AND class_sections_deleted_at IS NULL;

  IF v_sec_masjid IS NULL THEN
    RAISE EXCEPTION 'Section invalid/terhapus';
  END IF;

  IF NEW.class_attendance_sessions_masjid_id <> v_sec_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: session(%) vs section(%)',
      NEW.class_attendance_sessions_masjid_id, v_sec_masjid;
  END IF;

  -- Class_subject (opsional)
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

    IF v_sec_class IS NOT NULL AND v_cs_class IS NOT NULL AND v_sec_class <> v_cs_class THEN
      RAISE EXCEPTION 'class_subject.class_id berbeda dengan section.class_id';
    END IF;
  END IF;

  -- CSS Teacher (opsional)
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

    IF NEW.class_attendance_sessions_class_subject_id IS NOT NULL THEN
      IF v_cs_subject IS NULL THEN
        SELECT class_subjects_subject_id INTO v_cs_subject
        FROM class_subjects
        WHERE class_subjects_id = NEW.class_attendance_sessions_class_subject_id;
      END IF;

      IF v_css_subj <> v_cs_subject THEN
        RAISE EXCEPTION 'Subject CSS(%) != class_subject(%)', v_css_subj, v_cs_subject;
      END IF;
    END IF;

    IF NEW.class_attendance_sessions_teacher_user_id IS NULL THEN
      NEW.class_attendance_sessions_teacher_user_id := v_css_teacher;
    END IF;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

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
-- 5) Touch updated_at otomatis
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_class_attendance_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_attendance_sessions_updated_at := CURRENT_TIMESTAMP;
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

-- =========================================================
-- 6) TABLE: class_attendance_session_url (multi URL per sesi) — no is_primary/order
-- =========================================================
CREATE TABLE IF NOT EXISTS class_attendance_session_url (
  class_attendance_session_url_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  class_attendance_session_url_masjid_id  UUID NOT NULL,
  class_attendance_session_url_session_id UUID NOT NULL,

  class_attendance_session_url_label      VARCHAR(120),

  class_attendance_session_url_href       TEXT NOT NULL,
  class_attendance_session_url_trash_url  TEXT,
  class_attendance_session_url_delete_pending_until TIMESTAMPTZ,

  class_attendance_session_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_deleted_at TIMESTAMPTZ,

  CONSTRAINT fk_casu_session
    FOREIGN KEY (class_attendance_session_url_session_id)
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_casu_masjid
    FOREIGN KEY (class_attendance_session_url_masjid_id)
    REFERENCES masjids(masjid_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- Jika sebelumnya tabel sudah ada dan berisi kolom is_primary/order → hapus
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_attendance_session_url'
      AND column_name='class_attendance_session_url_is_primary'
  ) THEN
    EXECUTE 'ALTER TABLE class_attendance_session_url DROP COLUMN class_attendance_session_url_is_primary';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_attendance_session_url'
      AND column_name='class_attendance_session_url_order'
  ) THEN
    EXECUTE 'ALTER TABLE class_attendance_session_url DROP COLUMN class_attendance_session_url_order';
  END IF;
END$$;

-- Drop index lama yang bergantung pada is_primary (jika ada)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_casu_primary_per_session_alive') THEN
    EXECUTE 'DROP INDEX uq_casu_primary_per_session_alive';
  END IF;
END$$;

-- Tenant guard
CREATE OR REPLACE FUNCTION fn_casu_tenant_guard()
RETURNS TRIGGER AS $$
DECLARE v_mid UUID;
BEGIN
  SELECT class_attendance_sessions_masjid_id
    INTO v_mid
  FROM class_attendance_sessions
  WHERE class_attendance_sessions_id = NEW.class_attendance_session_url_session_id
    AND class_attendance_sessions_deleted_at IS NULL;

  IF v_mid IS NULL THEN
    RAISE EXCEPTION 'Session tidak valid/terhapus';
  END IF;

  IF NEW.class_attendance_session_url_masjid_id IS DISTINCT FROM v_mid THEN
    RAISE EXCEPTION 'Masjid mismatch: url(%) vs session(%)',
      NEW.class_attendance_session_url_masjid_id, v_mid;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_casu_tenant_guard') THEN
    DROP TRIGGER trg_casu_tenant_guard ON class_attendance_session_url;
  END IF;

  CREATE TRIGGER trg_casu_tenant_guard
  BEFORE INSERT OR UPDATE ON class_attendance_session_url
  FOR EACH ROW EXECUTE FUNCTION fn_casu_tenant_guard();
END$$;

-- Indexes & unique (tanpa primary-per-session)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casu_href_per_session_alive
  ON class_attendance_session_url (
    class_attendance_session_url_session_id,
    lower(class_attendance_session_url_href)
  )
  WHERE class_attendance_session_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casu_session_alive
  ON class_attendance_session_url (class_attendance_session_url_session_id)
  WHERE class_attendance_session_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casu_created_at
  ON class_attendance_session_url (class_attendance_session_url_created_at DESC);

-- Touch updated_at
CREATE OR REPLACE FUNCTION fn_touch_casu_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.class_attendance_session_url_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_casu_updated_at') THEN
    DROP TRIGGER trg_touch_casu_updated_at ON class_attendance_session_url;
  END IF;

  CREATE TRIGGER trg_touch_casu_updated_at
  BEFORE UPDATE ON class_attendance_session_url
  FOR EACH ROW EXECUTE FUNCTION fn_touch_casu_updated_at();
END$$;

-- =========================================================
-- 7) BACKFILL dari kolom lama di sessions (jika ada), lalu DROP kolom lama
-- =========================================================
DO $$
DECLARE
  has_img   boolean;
  has_trash boolean;
  has_due   boolean;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_image_url'
  ) INTO has_img;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_image_trash_url'
  ) INTO has_trash;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_image_delete_pending_until'
  ) INTO has_due;

  IF has_img THEN
    EXECUTE $ins$
      INSERT INTO class_attendance_session_url (
        class_attendance_session_url_masjid_id,
        class_attendance_session_url_session_id,
        class_attendance_session_url_label,
        class_attendance_session_url_href,
        class_attendance_session_url_trash_url,
        class_attendance_session_url_delete_pending_until
      )
      SELECT
        s.class_attendance_sessions_masjid_id,
        s.class_attendance_sessions_id,
        'Cover',
        s.class_attendance_sessions_image_url,
        CASE WHEN $1 THEN s.class_attendance_sessions_image_trash_url ELSE NULL END,
        CASE WHEN $2 THEN s.class_attendance_sessions_image_delete_pending_until ELSE NULL END
      FROM class_attendance_sessions s
      WHERE s.class_attendance_sessions_image_url IS NOT NULL
        AND btrim(s.class_attendance_sessions_image_url) <> ''
        AND NOT EXISTS (
          SELECT 1 FROM class_attendance_session_url u
          WHERE u.class_attendance_session_url_session_id = s.class_attendance_sessions_id
            AND u.class_attendance_session_url_deleted_at IS NULL
            AND lower(u.class_attendance_session_url_href) = lower(s.class_attendance_sessions_image_url)
        )
    $ins$ USING has_trash, has_due;
  END IF;
END$$;
