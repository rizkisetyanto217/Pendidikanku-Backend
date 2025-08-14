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


-- =========================================================
-- USER CLASS ATTENDANCE ENTRIES (fresh create, int status)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_attendance_entries (
  user_class_attendance_entries_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_attendance_entries_session_id    UUID NOT NULL,
  user_class_attendance_entries_user_class_id UUID NOT NULL,
  user_class_attendance_entries_masjid_id     UUID NOT NULL,

  -- 0=present,1=sick,2=leave,3=absent
  user_class_attendance_entries_attendance_status SMALLINT NOT NULL
    CHECK (user_class_attendance_entries_attendance_status IN (0,1,2,3)),
  -- 0 tanpa kehadiran, 1 hadir, 2 sakit, 3 izin

  user_class_attendance_entries_score INT
    CHECK (
      user_class_attendance_entries_score IS NULL
      OR (user_class_attendance_entries_score BETWEEN 0 AND 100)
    ),

  -- lulus/tidak lulus
  user_class_attendance_entries_grade_passed BOOLEAN,

  user_class_attendance_entries_material_personal TEXT,
  user_class_attendance_entries_personal_note     TEXT,
  user_class_attendance_entries_memorization      TEXT,
  user_class_attendance_entries_homework          TEXT,

  user_class_attendance_entries_search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(user_class_attendance_entries_material_personal,'')), 'B') ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_entries_personal_note,'')), 'B')     ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_entries_memorization,'')), 'C')      ||
    setweight(to_tsvector('simple', coalesce(user_class_attendance_entries_homework,'')), 'C')
  ) STORED,

  user_class_attendance_entries_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_class_attendance_entries_updated_at TIMESTAMP
);

-- ---------- FK ----------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucae_session') THEN
    ALTER TABLE user_class_attendance_entries
      ADD CONSTRAINT fk_ucae_session
      FOREIGN KEY (user_class_attendance_entries_session_id)
      REFERENCES class_attendance_sessions(class_attendance_sessions_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucae_user_class') THEN
    ALTER TABLE user_class_attendance_entries
      ADD CONSTRAINT fk_ucae_user_class
      FOREIGN KEY (user_class_attendance_entries_user_class_id)
      REFERENCES user_classes(user_classes_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_ucae_masjid') THEN
    ALTER TABLE user_class_attendance_entries
      ADD CONSTRAINT fk_ucae_masjid
      FOREIGN KEY (user_class_attendance_entries_masjid_id)
      REFERENCES masjids(masjid_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- ---------- 4) MIGRATE kolom status TEXT -> SMALLINT bila perlu ----------
-- ---------- 4) MIGRATE kolom status TEXT -> SMALLINT bila perlu ----------
DO $$
DECLARE
  v_datatype text;
BEGIN
  SELECT data_type
    INTO v_datatype
  FROM information_schema.columns
  WHERE table_schema='public'
    AND table_name='user_class_attendance_entries'
    AND column_name='user_class_attendance_entries_attendance_status';

  IF v_datatype = 'text' THEN
    -- Drop index yang mungkin bergantung ke status (aman jika belum ada)
    DROP INDEX IF EXISTS idx_ucae_session_status;
    DROP INDEX IF EXISTS idx_ucae_session_present_only;

    -- Drop semua CHECK di kolom status (kalau ada)
    PERFORM 1 FROM pg_constraint
     WHERE conrelid = 'user_class_attendance_entries'::regclass
       AND contype = 'c'
       AND pg_get_constraintdef(oid) ILIKE '%user_class_attendance_entries_attendance_status%';
    IF FOUND THEN
      DO $inner$
      DECLARE r RECORD;
      BEGIN
        FOR r IN
          SELECT conname
          FROM pg_constraint
          WHERE conrelid = 'user_class_attendance_entries'::regclass
            AND contype = 'c'
            AND pg_get_constraintdef(oid) ILIKE '%user_class_attendance_entries_attendance_status%'
        LOOP
          EXECUTE format('ALTER TABLE user_class_attendance_entries DROP CONSTRAINT %I', r.conname);
        END LOOP;
      END;
      $inner$;
    END IF;

    -- Convert TEXT -> SMALLINT (mapping: 0=absent,1=present,2=sick,3=leave)
    ALTER TABLE user_class_attendance_entries
      ALTER COLUMN user_class_attendance_entries_attendance_status
      TYPE SMALLINT
      USING (
        CASE lower(user_class_attendance_entries_attendance_status)
          WHEN 'absent' THEN 0
          WHEN 'present' THEN 1
          WHEN 'sick'   THEN 2
          WHEN 'leave'  THEN 3
          ELSE NULL
        END
      );
  END IF;

  -- Pastikan CHECK ada, tapi hanya kalau belum ada (cek nama constraint)
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'user_class_attendance_entries'::regclass
      AND conname  = 'chk_ucae_status_smallint'
  ) THEN
    ALTER TABLE user_class_attendance_entries
      ADD CONSTRAINT chk_ucae_status_smallint
      CHECK (user_class_attendance_entries_attendance_status IN (0,1,2,3));
  END IF;
END$$;


-- ---------- 5) Unique guard (idempotent) ----------
CREATE UNIQUE INDEX IF NOT EXISTS uidx_ucae_session_userclass
  ON user_class_attendance_entries (
    user_class_attendance_entries_session_id,
    user_class_attendance_entries_user_class_id
  );

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'uq_ucae_session_userclass'
      AND conrelid = 'user_class_attendance_entries'::regclass
  ) THEN
    ALTER TABLE user_class_attendance_entries
      ADD CONSTRAINT uq_ucae_session_userclass
      UNIQUE USING INDEX uidx_ucae_session_userclass;
  END IF;
END$$;

-- ---------- 6) Trigger updated_at ----------
CREATE OR REPLACE FUNCTION trg_set_timestamp_ucae()
RETURNS trigger AS $$
BEGIN
  NEW.user_class_attendance_entries_updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_ucae ON user_class_attendance_entries;
CREATE TRIGGER set_timestamp_ucae
BEFORE UPDATE ON user_class_attendance_entries
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_ucae();

-- ---------- 7) INDEXES ----------
-- 1) Timeline/aggregasi per masjid
CREATE INDEX IF NOT EXISTS idx_ucae_masjid_created_at
  ON user_class_attendance_entries (
    user_class_attendance_entries_masjid_id,
    user_class_attendance_entries_created_at DESC
  );

-- 2) Rekap per sesi (guru) - termasuk status (SMALLINT)
CREATE INDEX IF NOT EXISTS idx_ucae_session_status
  ON user_class_attendance_entries (
    user_class_attendance_entries_session_id,
    user_class_attendance_entries_attendance_status
  );

-- 3) Timeline progres per user_class (ortu/guru wali)
CREATE INDEX IF NOT EXISTS idx_ucae_userclass_created_at
  ON user_class_attendance_entries (
    user_class_attendance_entries_user_class_id,
    user_class_attendance_entries_created_at DESC
  );

-- 4) Kombinasi tenant+sesi
CREATE INDEX IF NOT EXISTS idx_ucae_masjid_session
  ON user_class_attendance_entries (
    user_class_attendance_entries_masjid_id,
    user_class_attendance_entries_session_id
  );

-- 5) BRIN per waktu
CREATE INDEX IF NOT EXISTS brin_ucae_created_at
  ON user_class_attendance_entries
  USING brin (user_class_attendance_entries_created_at);

-- 6) Full-text search
CREATE INDEX IF NOT EXISTS gin_ucae_search
  ON user_class_attendance_entries
  USING gin (user_class_attendance_entries_search);

-- 7) Partial index (hadir saja → status=0)
CREATE INDEX IF NOT EXISTS idx_ucae_session_present_only
  ON user_class_attendance_entries (user_class_attendance_entries_session_id)
  WHERE user_class_attendance_entries_attendance_status = 0;

-- 8) (Opsional) filter berdasar skor
CREATE INDEX IF NOT EXISTS idx_ucae_userclass_score
  ON user_class_attendance_entries (
    user_class_attendance_entries_user_class_id,
    user_class_attendance_entries_score
  );

