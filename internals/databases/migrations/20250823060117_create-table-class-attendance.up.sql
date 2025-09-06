BEGIN;

-- =========================
-- Prasyarat
-- =========================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

/* =========================================================
   CLASS_ATTENDANCE_SESSIONS (lite; + CSST; tanpa created_at/updated_at)
   ========================================================= */
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_attendance_sessions_section_id       UUID NOT NULL,
  class_attendance_sessions_masjid_id        UUID NOT NULL,

  -- Mapel Wajib
  class_attendance_sessions_class_subject_id UUID NOT NULL,

  -- (Baru) Optional linkage ke assignment (CSST)
  class_attendance_sessions_csst_id UUID, -- → class_section_subject_teachers

  -- Optional room
  class_attendance_sessions_class_room_id    UUID, -- FK → class_rooms

  class_attendance_sessions_date  DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note  TEXT,

  -- Guru yang mengajar (tetap/pengganti) → masjid_teachers
  class_attendance_sessions_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- soft delete
  class_attendance_sessions_deleted_at TIMESTAMPTZ
);

-- =========================
-- FOREIGN KEYS (tenant-safe)
-- =========================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_section_masjid_pair') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_section_masjid_pair
      FOREIGN KEY (class_attendance_sessions_section_id, class_attendance_sessions_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_class_subject') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_subject
      FOREIGN KEY (class_attendance_sessions_class_subject_id)
      REFERENCES class_subjects(class_subjects_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_class_room') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_room
      FOREIGN KEY (class_attendance_sessions_class_room_id)
      REFERENCES class_rooms(class_room_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;

  -- (Baru) FK ke CSST
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_csst') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_csst
      FOREIGN KEY (class_attendance_sessions_csst_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- =========================
-- INDEXES (akses cepat)
-- =========================
CREATE INDEX IF NOT EXISTS idx_cas_section
  ON class_attendance_sessions(class_attendance_sessions_section_id);

CREATE INDEX IF NOT EXISTS idx_cas_masjid
  ON class_attendance_sessions(class_attendance_sessions_masjid_id);

CREATE INDEX IF NOT EXISTS idx_cas_date
  ON class_attendance_sessions(class_attendance_sessions_date DESC);

CREATE INDEX IF NOT EXISTS idx_cas_class_subject
  ON class_attendance_sessions(class_attendance_sessions_class_subject_id);

-- (Baru) CSST reference
CREATE INDEX IF NOT EXISTS idx_cas_csst_alive
  ON class_attendance_sessions (class_attendance_sessions_csst_id)
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- (Baru) Query by masjid + CSST + date (alive)
CREATE INDEX IF NOT EXISTS idx_cas_masjid_csst_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_csst_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_class_room
  ON class_attendance_sessions(class_attendance_sessions_class_room_id);

CREATE INDEX IF NOT EXISTS idx_cas_teacher_id
  ON class_attendance_sessions(class_attendance_sessions_teacher_id);

-- Join umum (masjid, section, date) & hanya alive
CREATE INDEX IF NOT EXISTS idx_cas_masjid_section_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- (Opsional) Query by subject per masjid per date (alive)
CREATE INDEX IF NOT EXISTS idx_cas_masjid_subject_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_class_subject_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- (Opsional) Query by teacher per masjid per date (alive)
CREATE INDEX IF NOT EXISTS idx_cas_masjid_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_teacher_id,
    class_attendance_sessions_date DESC
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- Unik: satu sesi per (masjid, section, class_subject, date) saat belum soft-deleted
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_masjid_section_subject_date
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_class_subject_id,
    class_attendance_sessions_date
  )
  WHERE class_attendance_sessions_deleted_at IS NULL;

-- =========================
-- CLEANUP validator lama (jika ada) — tidak pakai trigger validasi
-- =========================
DROP TRIGGER IF EXISTS trg_cas_validate_links ON class_attendance_sessions;
DROP FUNCTION IF EXISTS fn_cas_validate_links();



/* =========================================================
   TABLE: class_attendance_session_url (multi URL per sesi)
   ========================================================= */
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

-- Hapus kolom lama jika masih ada
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

-- =========================
-- Indexes URL (tanpa trigger tenant guard)
-- =========================
-- Unik per session untuk href (case-insensitive), hanya alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_casu_href_per_session_alive
  ON class_attendance_session_url (
    class_attendance_session_url_session_id,
    lower(class_attendance_session_url_href)
  )
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Akses cepat per session (alive)
CREATE INDEX IF NOT EXISTS idx_casu_session_alive
  ON class_attendance_session_url (class_attendance_session_url_session_id)
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Akses cepat per masjid + created_at (alive)
CREATE INDEX IF NOT EXISTS idx_casu_masjid_created_alive
  ON class_attendance_session_url (class_attendance_session_url_masjid_id, class_attendance_session_url_created_at DESC)
  WHERE class_attendance_session_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casu_created_at
  ON class_attendance_session_url (class_attendance_session_url_created_at DESC);

-- Bersihkan trigger lama tenant guard (jika masih ada)
DROP TRIGGER IF EXISTS trg_casu_tenant_guard ON class_attendance_session_url;
DROP FUNCTION IF EXISTS fn_casu_tenant_guard();

-- =========================
-- Backfill dari kolom lama di sessions (jika ada)
-- =========================
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

COMMIT;
