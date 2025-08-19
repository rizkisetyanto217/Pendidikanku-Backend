-- =========================================================
-- FRESH INSTALL: Subjects, Class-Subjects, CSST, Attendance Sessions
-- (Idempotent: aman di-run berkali-kali)
-- =========================================================

-- ---------- EXTENSIONS ----------
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- ---------- TABLE ----------
CREATE TABLE IF NOT EXISTS subjects (
  subjects_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  subjects_code       VARCHAR(40)  NOT NULL,   -- unik per masjid (case-insensitive) bila belum dihapus
  subjects_name       VARCHAR(120) NOT NULL,
  subjects_desc       TEXT,

  subjects_is_active  BOOLEAN NOT NULL DEFAULT TRUE,

  subjects_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  subjects_updated_at TIMESTAMP,
  subjects_deleted_at TIMESTAMP
);

-- ---------- INDEXES ----------
-- Unik per masjid untuk row yang BELUM dihapus (soft delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_code_per_masjid
  ON subjects (subjects_masjid_id, lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;

-- Daftar aktif per masjid (abaikan yang sudah dihapus)
CREATE INDEX IF NOT EXISTS idx_subjects_active
  ON subjects(subjects_masjid_id)
  WHERE subjects_is_active = true AND subjects_deleted_at IS NULL;

-- Trigram untuk pencarian nama, hanya pada row yang belum dihapus
CREATE INDEX IF NOT EXISTS gin_subjects_name_trgm
  ON subjects USING gin (subjects_name gin_trgm_ops)
  WHERE subjects_deleted_at IS NULL;

-- (opsional) Index bantu untuk filter tenant (sering dipakai di WHERE)
CREATE INDEX IF NOT EXISTS idx_subjects_masjid_alive
  ON subjects(subjects_masjid_id)
  WHERE subjects_deleted_at IS NULL;

-- (opsional) Index bantu untuk pencarian case-insensitive code (non-unique)
-- berguna jika sering SELECT ... WHERE lower(subjects_code) = lower($1)
CREATE INDEX IF NOT EXISTS idx_subjects_code_ci_alive
  ON subjects (lower(subjects_code))
  WHERE subjects_deleted_at IS NULL;



-- ---------- TABLE ----------
CREATE TABLE IF NOT EXISTS class_subjects (
  class_subjects_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subjects_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_subjects_class_id   UUID NOT NULL REFERENCES classes(class_id)   ON DELETE CASCADE,
  class_subjects_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,

  -- metadata kurikulum (opsional)
  class_subjects_order_index INT,                       -- urutan di rapor (NULL = tak diatur)
  class_subjects_hours_per_week INT,                    -- jp/minggu
  class_subjects_min_passing_score INT CHECK (class_subjects_min_passing_score BETWEEN 0 AND 100),
  class_subjects_weight_on_report INT,                  -- bobot di rapor
  class_subjects_is_core BOOLEAN NOT NULL DEFAULT FALSE,
  class_subjects_academic_year TEXT,                    -- "2025/2026" (opsional)
  class_subjects_desc TEXT,

  class_subjects_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  class_subjects_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_subjects_updated_at TIMESTAMPTZ,
  class_subjects_deleted_at TIMESTAMPTZ                -- ⬅️ soft delete
);

-- ---------- SOFT-DELETE COLUMN BACKFILL (if older table) ----------
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='class_subjects'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='class_subjects' AND column_name='class_subjects_deleted_at'
  ) THEN
    ALTER TABLE class_subjects
      ADD COLUMN class_subjects_deleted_at TIMESTAMPTZ;
  END IF;
END$$;

-- ---------- ENSURE TIMESTAMPTZ (migrate from timestamp without time zone if needed) ----------
DO $$
DECLARE
  v_created_type text;
  v_updated_type text;
  v_deleted_type text;
BEGIN
  SELECT data_type INTO v_created_type
  FROM information_schema.columns
  WHERE table_name='class_subjects' AND column_name='class_subjects_created_at';

  IF v_created_type = 'timestamp without time zone' THEN
    ALTER TABLE class_subjects
      ALTER COLUMN class_subjects_created_at TYPE timestamptz
      USING class_subjects_created_at AT TIME ZONE 'UTC';
  END IF;

  SELECT data_type INTO v_updated_type
  FROM information_schema.columns
  WHERE table_name='class_subjects' AND column_name='class_subjects_updated_at';

  IF v_updated_type = 'timestamp without time zone' THEN
    ALTER TABLE class_subjects
      ALTER COLUMN class_subjects_updated_at TYPE timestamptz
      USING class_subjects_updated_at AT TIME ZONE 'UTC';
  END IF;

  SELECT data_type INTO v_deleted_type
  FROM information_schema.columns
  WHERE table_name='class_subjects' AND column_name='class_subjects_deleted_at';

  IF v_deleted_type = 'timestamp without time zone' THEN
    ALTER TABLE class_subjects
      ALTER COLUMN class_subjects_deleted_at TYPE timestamptz
      USING class_subjects_deleted_at AT TIME ZONE 'UTC';
  END IF;
END$$;

-- ---------- CHECK NUMERIC (idempotent guards) ----------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cs_order_index_nonneg') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT chk_cs_order_index_nonneg
      CHECK (class_subjects_order_index IS NULL OR class_subjects_order_index >= 0);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cs_hours_per_week_nonneg') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT chk_cs_hours_per_week_nonneg
      CHECK (class_subjects_hours_per_week IS NULL OR class_subjects_hours_per_week >= 0);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cs_weight_on_report_nonneg') THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT chk_cs_weight_on_report_nonneg
      CHECK (class_subjects_weight_on_report IS NULL OR class_subjects_weight_on_report >= 0);
  END IF;
END$$;

-- ---------- (Opsional) TENANT-SAFE FK komposit ke classes (class_id, class_masjid_id) ----------
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema='public' AND table_name='classes' AND column_name='class_masjid_id'
  ) AND NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_cs_class_masjid_pair'
  ) THEN
    ALTER TABLE class_subjects
      ADD CONSTRAINT fk_cs_class_masjid_pair
      FOREIGN KEY (class_subjects_class_id, class_subjects_masjid_id)
      REFERENCES classes (class_id, class_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- ---------- UNIQUE (soft-delete aware) ----------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_class_subjects') THEN
    DROP INDEX IF EXISTS uq_class_subjects;
  END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_subjects
ON class_subjects (
  class_subjects_masjid_id,
  class_subjects_class_id,
  class_subjects_subject_id,
  COALESCE(class_subjects_academic_year,'')
)
WHERE class_subjects_deleted_at IS NULL;

-- ---------- INDEXES (soft-delete aware) ----------
-- List mapel per kelas (aktif per tahun)
CREATE INDEX IF NOT EXISTS idx_cs_masjid_class_year_active
  ON class_subjects (
    class_subjects_masjid_id,
    class_subjects_class_id,
    COALESCE(class_subjects_academic_year,'')
  )
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

-- Cari semua kelas yang mengajarkan satu subject (aktif per tahun)
CREATE INDEX IF NOT EXISTS idx_cs_masjid_subject_year_active
  ON class_subjects (
    class_subjects_masjid_id,
    class_subjects_subject_id,
    COALESCE(class_subjects_academic_year,'')
  )
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

-- Rekap cepat per masjid (aktif)
CREATE INDEX IF NOT EXISTS idx_cs_masjid_active
  ON class_subjects (class_subjects_masjid_id)
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

-- Urutan rapor per kelas (aktif)
CREATE INDEX IF NOT EXISTS idx_cs_class_order
  ON class_subjects (class_subjects_class_id, class_subjects_order_index)
  WHERE class_subjects_is_active = TRUE
    AND class_subjects_deleted_at IS NULL;

-- Filter umum "alive" untuk tenant queries
CREATE INDEX IF NOT EXISTS idx_cs_masjid_alive
  ON class_subjects (class_subjects_masjid_id)
  WHERE class_subjects_deleted_at IS NULL;

-- Pencarian teks untuk deskripsi (opsional)
CREATE INDEX IF NOT EXISTS gin_cs_desc_trgm
  ON class_subjects USING gin (class_subjects_desc gin_trgm_ops)
  WHERE class_subjects_deleted_at IS NULL;

-- ---------- TRIGGER updated_at ----------
CREATE OR REPLACE FUNCTION trg_set_timestamp_class_subjects()
RETURNS trigger AS $$
BEGIN
  NEW.class_subjects_updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_class_subjects ON class_subjects;
CREATE TRIGGER set_timestamp_class_subjects
BEFORE UPDATE ON class_subjects
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_class_subjects();

-- =========================================================
-- NOTES:
-- - Jika sekolah tidak membedakan per tahun, biarkan academic_year NULL (unik tetap aman).
-- - Jika nanti ingin tanpa academic_year sama sekali, drop kolom + ganti UNIQUE ke 3 kolom (masjid,class,subject).
-- - Info buku disimpan di tabel class_books (child → parent), bukan di class_subjects.
-- =========================================================

-- =========================================================
-- CLASS SECTION SUBJECT TEACHERS (soft delete friendly)
-- =========================================================

-- 1) TABLE
CREATE TABLE IF NOT EXISTS class_section_subject_teachers (
  class_section_subject_teachers_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_section_subject_teachers_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_section_subject_teachers_section_id UUID NOT NULL REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,
  class_section_subject_teachers_subject_id UUID NOT NULL REFERENCES subjects(subjects_id) ON DELETE RESTRICT,
  class_section_subject_teachers_teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

  class_section_subject_teachers_is_active BOOLEAN NOT NULL DEFAULT TRUE,
  class_section_subject_teachers_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  class_section_subject_teachers_updated_at TIMESTAMP,
  class_section_subject_teachers_deleted_at TIMESTAMP     -- ⬅️ soft delete
);

-- 1b) Tambahkan kolom deleted_at jika belum ada (idempotent)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='class_section_subject_teachers'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='class_section_subject_teachers' AND column_name='class_section_subject_teachers_deleted_at'
  ) THEN
    ALTER TABLE class_section_subject_teachers
      ADD COLUMN class_section_subject_teachers_deleted_at TIMESTAMP;
  END IF;
END$$;

-- 2) TENANT-SAFE FK komposit ke class_sections (section_id, masjid_id)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_class_sec_subj_teachers_section_masjid'
  ) THEN
    ALTER TABLE class_section_subject_teachers
      ADD CONSTRAINT fk_class_sec_subj_teachers_section_masjid
      FOREIGN KEY (class_section_subject_teachers_section_id, class_section_subject_teachers_masjid_id)
      REFERENCES class_sections (class_sections_id, class_sections_masjid_id)
      ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END$$;

-- 3) UNIQUE aktif: cegah duplikasi penugasan guru untuk (section, subject, teacher) yang masih aktif & belum dihapus
--    (team teaching tetap diperbolehkan karena uniqueness menyertakan teacher_user_id)
DO $$
BEGIN
  -- drop index lama agar bisa ganti ke partial + deleted_at
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_class_sec_subj_teachers_active') THEN
    DROP INDEX IF EXISTS uq_class_sec_subj_teachers_active;
  END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sec_subj_teachers_active
ON class_section_subject_teachers (
  class_section_subject_teachers_section_id,
  class_section_subject_teachers_subject_id,
  class_section_subject_teachers_teacher_user_id
)
WHERE class_section_subject_teachers_is_active = TRUE
  AND class_section_subject_teachers_deleted_at IS NULL;

-- ALTERNATIF (tetap 1 guru saja per section+subject): tambahkan filter deleted_at juga
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sec_subj_teachers_active_single
-- ON class_section_subject_teachers (
--   class_section_subject_teachers_section_id,
--   class_section_subject_teachers_subject_id
-- )
-- WHERE class_section_subject_teachers_is_active = TRUE
--   AND class_section_subject_teachers_deleted_at IS NULL;

-- 4) INDEX umum (filter deleted_at agar ringan & sesuai query umum)
CREATE INDEX IF NOT EXISTS idx_class_sec_subj_teachers_section_alive
  ON class_section_subject_teachers(class_section_subject_teachers_section_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_sec_subj_teachers_subject_alive
  ON class_section_subject_teachers(class_section_subject_teachers_subject_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_sec_subj_teachers_masjid_alive
  ON class_section_subject_teachers(class_section_subject_teachers_masjid_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_sec_subj_teachers_teacher_alive
  ON class_section_subject_teachers(class_section_subject_teachers_teacher_user_id)
  WHERE class_section_subject_teachers_deleted_at IS NULL;

-- (opsional) kombinasi untuk query listing aktif per section+subject
CREATE INDEX IF NOT EXISTS idx_class_sec_subj_teachers_sec_subj_active_alive
  ON class_section_subject_teachers(
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_subject_id
  )
  WHERE class_section_subject_teachers_is_active = TRUE
    AND class_section_subject_teachers_deleted_at IS NULL;

-- 5) TRIGGER updated_at
CREATE OR REPLACE FUNCTION trg_set_timestamp_class_sec_subj_teachers()
RETURNS trigger AS $$
BEGIN
  NEW.class_section_subject_teachers_updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_class_sec_subj_teachers ON class_section_subject_teachers;
CREATE TRIGGER set_timestamp_class_sec_subj_teachers
BEFORE UPDATE ON class_section_subject_teachers
FOR EACH ROW EXECUTE FUNCTION trg_set_timestamp_class_sec_subj_teachers();

-- 6) VALIDASI TENANT: pastikan subject & section berada di masjid yang sama
--    (opsional) jika tabel subject/section juga punya deleted_at, validasi hanya pada yang belum dihapus
CREATE OR REPLACE FUNCTION fn_class_sec_subj_teachers_validate_tenant()
RETURNS TRIGGER AS $BODY$
DECLARE
  v_sec_masjid UUID;
  v_sub_masjid UUID;
  has_sec_deleted_at BOOLEAN := FALSE;
  has_sub_deleted_at BOOLEAN := FALSE;
BEGIN
  -- deteksi kolom deleted_at di class_sections
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='class_sections' AND column_name='class_sections_deleted_at'
  ) INTO has_sec_deleted_at;

  -- deteksi kolom deleted_at di subjects
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='subjects' AND column_name='subjects_deleted_at'
  ) INTO has_sub_deleted_at;

  -- validasi section (hanya yang belum dihapus jika kolomnya ada)
  IF has_sec_deleted_at THEN
    SELECT class_sections_masjid_id INTO v_sec_masjid
    FROM class_sections
    WHERE class_sections_id = NEW.class_section_subject_teachers_section_id
      AND class_sections_deleted_at IS NULL;
  ELSE
    SELECT class_sections_masjid_id INTO v_sec_masjid
    FROM class_sections
    WHERE class_sections_id = NEW.class_section_subject_teachers_section_id;
  END IF;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Section % tidak ditemukan / sudah dihapus', NEW.class_section_subject_teachers_section_id;
  END IF;
  IF v_sec_masjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: section(%) != row_masjid(%)',
      v_sec_masjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  -- validasi subject (hanya yang belum dihapus jika kolomnya ada)
  IF has_sub_deleted_at THEN
    SELECT subjects_masjid_id INTO v_sub_masjid
    FROM subjects
    WHERE subjects_id = NEW.class_section_subject_teachers_subject_id
      AND subjects_deleted_at IS NULL;
  ELSE
    SELECT subjects_masjid_id INTO v_sub_masjid
    FROM subjects
    WHERE subjects_id = NEW.class_section_subject_teachers_subject_id;
  END IF;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Subject % tidak ditemukan / sudah dihapus', NEW.class_section_subject_teachers_subject_id;
  END IF;
  IF v_sub_masjid IS DISTINCT FROM NEW.class_section_subject_teachers_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: subject(%) != row_masjid(%)',
      v_sub_masjid, NEW.class_section_subject_teachers_masjid_id;
  END IF;

  RETURN NEW;
END
$BODY$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_class_sec_subj_teachers_validate_tenant') THEN
    DROP TRIGGER trg_class_sec_subj_teachers_validate_tenant ON class_section_subject_teachers;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_class_sec_subj_teachers_validate_tenant
  AFTER INSERT OR UPDATE OF
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_subject_id
  ON class_section_subject_teachers
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_class_sec_subj_teachers_validate_tenant();
END$$;


-- =========================================================
-- CLASS ATTENDANCE SESSIONS (CAS) — Refactor tanpa "csst"
-- =========================================================

-- 0) (Opsional, migrasi) rename kolom lama jika masih ada
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_csst_id'
  )
  AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='class_attendance_sessions'
      AND column_name='class_attendance_sessions_class_section_subject_teachers_id'
  ) THEN
    ALTER TABLE class_attendance_sessions
      RENAME COLUMN class_attendance_sessions_csst_id
      TO class_attendance_sessions_class_section_subject_teachers_id;
  END IF;
END$$;

-- 1) TABLE (fresh install)
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_sessions_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_attendance_sessions_section_id UUID NOT NULL,
  class_attendance_sessions_masjid_id  UUID NOT NULL,

  -- Integrasi mapel & penugasan
  class_attendance_sessions_subject_id UUID,  -- boleh NULL saat awal
  class_attendance_sessions_class_section_subject_teachers_id UUID, -- opsional, validasi via trigger

  class_attendance_sessions_date  DATE NOT NULL DEFAULT CURRENT_DATE,
  class_attendance_sessions_title TEXT,
  class_attendance_sessions_general_info TEXT NOT NULL,
  class_attendance_sessions_note  TEXT,
  class_attendance_sessions_teacher_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  class_attendance_sessions_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  class_attendance_sessions_updated_at TIMESTAMP,
  class_attendance_sessions_deleted_at TIMESTAMP
);

-- 2) FK
-- (a) Komposit ke class_sections (tenant-safe)
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

-- (b) Ke subjects
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cas_subject') THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_subject
      FOREIGN KEY (class_attendance_sessions_subject_id)
      REFERENCES subjects(subjects_id)
      ON UPDATE CASCADE ON DELETE RESTRICT;
  END IF;
END$$;

-- (c) Ke class_section_subject_teachers (nama & PK yang benar)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_csst') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_csst;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teachers'
  ) THEN
    ALTER TABLE class_attendance_sessions
      ADD CONSTRAINT fk_cas_class_section_subject_teachers
      FOREIGN KEY (class_attendance_sessions_class_section_subject_teachers_id)
      REFERENCES class_section_subject_teachers(class_section_subject_teachers_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- 3) INDEXES
-- Unik: satu (masjid, section, subject, date)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_section_subject_date
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_subject_id,
    class_attendance_sessions_date
  );

-- Query umum
CREATE INDEX IF NOT EXISTS idx_cas_section
  ON class_attendance_sessions(class_attendance_sessions_section_id);

CREATE INDEX IF NOT EXISTS idx_cas_masjid
  ON class_attendance_sessions(class_attendance_sessions_masjid_id);

CREATE INDEX IF NOT EXISTS idx_cas_date
  ON class_attendance_sessions(class_attendance_sessions_date DESC);

CREATE INDEX IF NOT EXISTS idx_cas_subject
  ON class_attendance_sessions(class_attendance_sessions_subject_id);

CREATE INDEX IF NOT EXISTS idx_cas_css_teachers
  ON class_attendance_sessions(class_attendance_sessions_class_section_subject_teachers_id);

CREATE INDEX IF NOT EXISTS idx_cas_masjid_section_subject_date
  ON class_attendance_sessions(
    class_attendance_sessions_masjid_id,
    class_attendance_sessions_section_id,
    class_attendance_sessions_subject_id,
    class_attendance_sessions_date DESC
  );

-- (Opsional) Jika tetap mau guard "satu sesi per section per date" TANPA soft-delete:
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_section_date_plain
--   ON class_attendance_sessions (class_attendance_sessions_section_id, class_attendance_sessions_date);

-- 4) VALIDASI konsistensi penugasan guru (DEFERRABLE TRIGGER)
CREATE OR REPLACE FUNCTION fn_cas_validate_csst()
RETURNS TRIGGER AS $BODY$
DECLARE
  v_section UUID;
  v_subject UUID;
  v_masjid  UUID;
  v_teacher UUID;
BEGIN
  IF NEW.class_attendance_sessions_class_section_subject_teachers_id IS NULL THEN
    RETURN NEW; -- tidak perlu validasi
  END IF;

  SELECT
    class_section_subject_teachers_section_id,
    class_section_subject_teachers_subject_id,
    class_section_subject_teachers_masjid_id,
    class_section_subject_teachers_teacher_user_id
  INTO v_section, v_subject, v_masjid, v_teacher
  FROM class_section_subject_teachers
  WHERE class_section_subject_teachers_id = NEW.class_attendance_sessions_class_section_subject_teachers_id;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'ClassSectionSubjectTeachers % tidak ditemukan',
      NEW.class_attendance_sessions_class_section_subject_teachers_id;
  END IF;

  IF v_section IS DISTINCT FROM NEW.class_attendance_sessions_section_id THEN
    RAISE EXCEPTION 'Section mismatch: CSST(%) != CAS(%)', v_section, NEW.class_attendance_sessions_section_id;
  END IF;

  IF v_subject IS DISTINCT FROM NEW.class_attendance_sessions_subject_id THEN
    RAISE EXCEPTION 'Subject mismatch: CSST(%) != CAS(%)', v_subject, NEW.class_attendance_sessions_subject_id;
  END IF;

  IF v_masjid IS DISTINCT FROM NEW.class_attendance_sessions_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: CSST(%) != CAS(%)', v_masjid, NEW.class_attendance_sessions_masjid_id;
  END IF;

  IF NEW.class_attendance_sessions_teacher_user_id IS NOT NULL
     AND NEW.class_attendance_sessions_teacher_user_id IS DISTINCT FROM v_teacher THEN
    RAISE EXCEPTION 'Teacher mismatch: CAS(%) != CSST(%)',
      NEW.class_attendance_sessions_teacher_user_id, v_teacher;
  END IF;

  RETURN NEW;
END
$BODY$ LANGUAGE plpgsql;

-- Recreate constraint trigger dengan kolom yang benar
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_validate_csst') THEN
    DROP TRIGGER trg_cas_validate_csst ON class_attendance_sessions;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_cas_validate_csst
    AFTER INSERT OR UPDATE OF
      class_attendance_sessions_class_section_subject_teachers_id,
      class_attendance_sessions_section_id,
      class_attendance_sessions_subject_id,
      class_attendance_sessions_masjid_id,
      class_attendance_sessions_teacher_user_id
    ON class_attendance_sessions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_cas_validate_csst();
END$$;

-- 5) (Opsional) Setelah migrasi penuh, kamu bisa enforce NOT NULL:
-- ALTER TABLE class_attendance_sessions
--   ALTER COLUMN class_attendance_sessions_subject_id SET NOT NULL;




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