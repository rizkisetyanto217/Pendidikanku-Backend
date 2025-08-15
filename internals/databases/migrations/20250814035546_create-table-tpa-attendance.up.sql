-- =========================================================
-- Class Attendance - Rename kolom + Setup FK & Indexes
-- Idempotent & aman untuk DB existing maupun fresh install
-- =========================================================

-- ---------- EXTENSIONS (sekali saja) ----------
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- ---------- 0) Fresh install: buat tabel dgn kolom baru ----------
DO $$
BEGIN
  IF to_regclass('public.class_attendance_sessions') IS NULL THEN
    CREATE TABLE class_attendance_sessions (
      class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

      class_attendance_sessions_section_id UUID NOT NULL,
      class_attendance_sessions_masjid_id  UUID NOT NULL,

      class_attendance_sessions_date  DATE NOT NULL DEFAULT CURRENT_DATE,
      class_attendance_sessions_title TEXT,
      class_attendance_sessions_general_info TEXT NOT NULL,
      class_attendance_sessions_note  TEXT,
      class_attendance_sessions_teacher_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

      class_attendance_sessions_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      class_attendance_sessions_updated_at TIMESTAMP
    );
  END IF;
END$$;

-- ---------- 1) RENAME kolom lama session_* -> class_attendance_sessions_* ----------
DO $$
BEGIN
  PERFORM 1;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_id')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_id TO class_attendance_sessions_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_section_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_section_id')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_section_id TO class_attendance_sessions_section_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_masjid_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_masjid_id')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_masjid_id TO class_attendance_sessions_masjid_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_date')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_date')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_date TO class_attendance_sessions_date;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_title')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_title')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_title TO class_attendance_sessions_title;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_general_info')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_general_info')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_general_info TO class_attendance_sessions_general_info;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_note')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_note')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_note TO class_attendance_sessions_note;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_teacher_user_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_teacher_user_id')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_teacher_user_id TO class_attendance_sessions_teacher_user_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_created_at')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_created_at')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_created_at TO class_attendance_sessions_created_at;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name='class_attendance_sessions' AND column_name='session_updated_at')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                     WHERE table_name='class_attendance_sessions' AND column_name='class_attendance_sessions_updated_at')
  THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN session_updated_at TO class_attendance_sessions_updated_at;
  END IF;
END$$;

-- ---------- 2) INDEX & UNIQUE ----------
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_section_date
  ON class_attendance_sessions(class_attendance_sessions_section_id, class_attendance_sessions_date);

CREATE INDEX IF NOT EXISTS idx_cas_section
  ON class_attendance_sessions(class_attendance_sessions_section_id);

CREATE INDEX IF NOT EXISTS idx_cas_masjid
  ON class_attendance_sessions(class_attendance_sessions_masjid_id);

CREATE INDEX IF NOT EXISTS idx_cas_date
  ON class_attendance_sessions(class_attendance_sessions_date DESC);

CREATE INDEX IF NOT EXISTS idx_cas_masjid_section_date
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_date DESC
  );

-- ---------- 3) FK komposit ke class_sections: (section_id, masjid_id) ----------
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'fk_cas_section_masjid_pair'
  ) THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_section_masjid_pair
      FOREIGN KEY (class_attendance_sessions_section_id, class_attendance_sessions_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;


-- ---------- 1) CREATE TABLE (fresh install bila belum ada) ----------
CREATE TABLE IF NOT EXISTS user_class_attendance_sessions (
  user_class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_attendance_sessions_session_id    UUID NOT NULL,
  user_class_attendance_sessions_user_class_id UUID NOT NULL,
  user_class_attendance_sessions_masjid_id     UUID NOT NULL,

  user_class_attendance_sessions_attendance_status TEXT NOT NULL
    CHECK (user_class_attendance_sessions_attendance_status IN ('present','sick','leave','absent')),

  user_class_attendance_sessions_score INT
    CHECK (
      user_class_attendance_sessions_score IS NULL
      OR (user_class_attendance_sessions_score BETWEEN 0 AND 100)
    ),

  user_class_attendance_sessions_grade_passed BOOLEAN,

  user_class_attendance_sessions_material_personal TEXT,
  user_class_attendance_sessions_personal_note     TEXT,
  user_class_attendance_sessions_memorization      TEXT,
  user_class_attendance_sessions_homework          TEXT,

  user_class_attendance_sessions_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_class_attendance_sessions_material_personal,'')), 'B') ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_sessions_personal_note,'')), 'B')     ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_sessions_memorization,'')), 'C')      ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_sessions_homework,'')), 'C')
  ) STORED,

  user_class_attendance_sessions_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_class_attendance_sessions_updated_at TIMESTAMP
);

-- ---------- 2) FK (idempotent) ----------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucas_session') THEN
    ALTER TABLE user_class_attendance_sessions
      ADD CONSTRAINT fk_ucas_session
      FOREIGN KEY (user_class_attendance_sessions_session_id)
      REFERENCES class_attendance_sessions(class_attendance_sessions_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucas_user_class') THEN
    ALTER TABLE user_class_attendance_sessions
      ADD CONSTRAINT fk_ucas_user_class
      FOREIGN KEY (user_class_attendance_sessions_user_class_id)
      REFERENCES user_classes(user_classes_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucas_masjid') THEN
    ALTER TABLE user_class_attendance_sessions
      ADD CONSTRAINT fk_ucas_masjid
      FOREIGN KEY (user_class_attendance_sessions_masjid_id)
      REFERENCES masjids(masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- ---------- 3) MIGRASI STATUS: SMALLINT -> TEXT (bila perlu) ----------
DO $$
DECLARE
  v_type TEXT;
BEGIN
  SELECT data_type INTO v_type
  FROM information_schema.columns
  WHERE table_schema='public'
    AND table_name='user_class_attendance_sessions'
    AND column_name='user_class_attendance_sessions_attendance_status';

  IF v_type = 'smallint' THEN
    -- drop index lama yang mungkin ada
    DROP INDEX IF EXISTS idx_ucae_session_present_only;
    DROP INDEX IF EXISTS idx_ucas_session_present_only;

    -- drop CHECK lama yg menempel di kolom
    PERFORM 1 FROM pg_constraint
     WHERE conrelid='user_class_attendance_sessions'::regclass
       AND contype='c'
       AND pg_get_constraintdef(oid) ILIKE '%user_class_attendance_sessions_attendance_status%';
    IF FOUND THEN
      DO $inner$
      DECLARE r RECORD;
      BEGIN
        FOR r IN
          SELECT conname
          FROM pg_constraint
          WHERE conrelid='user_class_attendance_sessions'::regclass
            AND contype='c'
            AND pg_get_constraintdef(oid) ILIKE '%user_class_attendance_sessions_attendance_status%'
        LOOP
          EXECUTE format('ALTER TABLE user_class_attendance_sessions DROP CONSTRAINT %I', r.conname);
        END LOOP;
      END;
      $inner$;
    END IF;

    -- mapping angka -> string
    ALTER TABLE user_class_attendance_sessions
      ALTER COLUMN user_class_attendance_sessions_attendance_status
      TYPE TEXT
      USING (
        CASE user_class_attendance_sessions_attendance_status
          WHEN 0 THEN 'present'
          WHEN 1 THEN 'sick'
          WHEN 2 THEN 'leave'
          WHEN 3 THEN 'absent'
          ELSE NULL
        END
      );
  END IF;

  -- normalisasi ke lower-trim
  IF v_type = 'text' THEN
    UPDATE user_class_attendance_sessions
       SET user_class_attendance_sessions_attendance_status =
           lower(trim(user_class_attendance_sessions_attendance_status))
     WHERE user_class_attendance_sessions_attendance_status IS NOT NULL;
  END IF;

  -- tambahkan CHECK final jika belum ada
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid='user_class_attendance_sessions'::regclass
      AND conname='chk_ucas_status_text'
  ) THEN
    ALTER TABLE user_class_attendance_sessions
      ADD CONSTRAINT chk_ucas_status_text
      CHECK (user_class_attendance_sessions_attendance_status
             IN ('present','sick','leave','absent'));
  END IF;
END$$;

-- ---------- 4) UNIQUE guard (per (session_id, user_class_id)) ----------
CREATE UNIQUE INDEX IF NOT EXISTS uidx_ucas_session_userclass
  ON user_class_attendance_sessions (
    user_class_attendance_sessions_session_id,
    user_class_attendance_sessions_user_class_id
  );

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname='uq_ucas_session_userclass'
      AND conrelid='user_class_attendance_sessions'::regclass
  ) THEN
    ALTER TABLE user_class_attendance_sessions
      ADD CONSTRAINT uq_ucas_session_userclass
      UNIQUE USING INDEX uidx_ucas_session_userclass;
  END IF;
END$$;

-- ---------- 5) TRIGGER updated_at ----------
CREATE OR REPLACE FUNCTION trg_set_timestamp_ucas()
RETURNS trigger AS $$
BEGIN
  NEW.user_class_attendance_sessions_updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_ucas ON user_class_attendance_sessions;
CREATE TRIGGER set_timestamp_ucas
BEFORE UPDATE ON user_class_attendance_sessions
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_ucas();

-- ---------- 6) INDEXES ----------
-- Timeline/aggregasi per masjid
CREATE INDEX IF NOT EXISTS idx_ucas_masjid_created_at
  ON user_class_attendance_sessions (
    user_class_attendance_sessions_masjid_id,
    user_class_attendance_sessions_created_at DESC
  );

-- Rekap per sesi + status (TEXT)
DROP INDEX IF EXISTS idx_ucae_session_status;
CREATE INDEX IF NOT EXISTS idx_ucas_session_status
  ON user_class_attendance_sessions (
    user_class_attendance_sessions_session_id,
    user_class_attendance_sessions_attendance_status
  );

-- Timeline progres per user_class
CREATE INDEX IF NOT EXISTS idx_ucas_userclass_created_at
  ON user_class_attendance_sessions (
    user_class_attendance_sessions_user_class_id,
    user_class_attendance_sessions_created_at DESC
  );

-- Kombinasi tenant + sesi
CREATE INDEX IF NOT EXISTS idx_ucas_masjid_session
  ON user_class_attendance_sessions (
    user_class_attendance_sessions_masjid_id,
    user_class_attendance_sessions_session_id
  );

-- BRIN per waktu
CREATE INDEX IF NOT EXISTS brin_ucas_created_at
  ON user_class_attendance_sessions
  USING brin (user_class_attendance_sessions_created_at);

-- Full-text search
CREATE INDEX IF NOT EXISTS gin_ucas_search
  ON user_class_attendance_sessions
  USING gin (user_class_attendance_sessions_search);

-- Partial index: attended (present/sick/leave)
DROP INDEX IF EXISTS idx_ucae_session_attended;
CREATE INDEX IF NOT EXISTS idx_ucas_session_attended
  ON user_class_attendance_sessions (user_class_attendance_sessions_session_id)
  WHERE user_class_attendance_sessions_attendance_status IN ('present','sick','leave');

-- Partial index: absent saja
DROP INDEX IF EXISTS idx_ucae_session_absent;
CREATE INDEX IF NOT EXISTS idx_ucas_session_absent
  ON user_class_attendance_sessions (user_class_attendance_sessions_session_id)
  WHERE user_class_attendance_sessions_attendance_status = 'absent';