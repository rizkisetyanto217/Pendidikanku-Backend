-- =========================================================
-- lembaga_stats (global snapshot, tanpa term)
-- =========================================================
CREATE TABLE IF NOT EXISTS lembaga_stats (
  lembaga_stats_school_id UUID PRIMARY KEY
    REFERENCES schools(school_id) ON DELETE CASCADE,

  lembaga_stats_active_classes  INT NOT NULL DEFAULT 0,
  lembaga_stats_active_sections INT NOT NULL DEFAULT 0,
  lembaga_stats_active_students INT NOT NULL DEFAULT 0,
  lembaga_stats_active_teachers INT NOT NULL DEFAULT 0,

  lembaga_stats_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lembaga_stats_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_lembaga_stats_nonneg') THEN
    ALTER TABLE lembaga_stats
      ADD CONSTRAINT chk_lembaga_stats_nonneg
      CHECK (
        lembaga_stats_active_classes  >= 0 AND
        lembaga_stats_active_sections >= 0 AND
        lembaga_stats_active_students >= 0 AND
        lembaga_stats_active_teachers >= 0
      );
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_lembaga_stats_updated_at
  ON lembaga_stats(lembaga_stats_updated_at);



-- =========================================================
-- (MIGRASI) Bersihkan peninggalan lama yang pakai user_classes
-- =========================================================
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.constraint_column_usage
    WHERE constraint_name = 'fk_ucass_user_class'
  ) THEN
    ALTER TABLE user_class_attendance_semester_stats
      DROP CONSTRAINT fk_ucass_user_class;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'user_class_attendance_semester_stats'
      AND column_name = 'user_class_attendance_semester_stats_user_class_id'
  ) THEN
    ALTER TABLE user_class_attendance_semester_stats
      DROP COLUMN user_class_attendance_semester_stats_user_class_id;
  END IF;
END$$;



-- =========================================================
-- user_class_attendance_semester_stats (tanpa user_classes)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_attendance_semester_stats (
  user_class_attendance_semester_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_attendance_semester_stats_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- Hanya relasi ke SECTION (komposit tenant-safe)
  user_class_attendance_semester_stats_section_id UUID NOT NULL,

  -- Referensi opsional ke academic_terms (komposit tenant-safe)
  user_class_attendance_semester_stats_term_id UUID,

  -- Snapshot periode (tetap disimpan untuk audit/jejak) — inclusive dates
  user_class_attendance_semester_stats_period_start DATE NOT NULL,
  user_class_attendance_semester_stats_period_end   DATE NOT NULL,
  CONSTRAINT chk_ucass_period_range
  CHECK (user_class_attendance_semester_stats_period_start
         <= user_class_attendance_semester_stats_period_end),

  -- Counters
  user_class_attendance_semester_stats_present_count INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_present_count >= 0),
  user_class_attendance_semester_stats_sick_count    INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_sick_count    >= 0),
  user_class_attendance_semester_stats_leave_count   INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_leave_count   >= 0),
  user_class_attendance_semester_stats_absent_count  INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_absent_count  >= 0),

  user_class_attendance_semester_stats_sum_score INT,

  user_class_attendance_semester_stats_total_sessions INT
    GENERATED ALWAYS AS (
      (user_class_attendance_semester_stats_present_count
     + user_class_attendance_semester_stats_sick_count
     + user_class_attendance_semester_stats_leave_count
     + user_class_attendance_semester_stats_absent_count)
    ) STORED,

  user_class_attendance_semester_stats_avg_score NUMERIC(6,3)
    GENERATED ALWAYS AS (
      CASE
        WHEN user_class_attendance_semester_stats_sum_score IS NULL THEN NULL
        ELSE (
          user_class_attendance_semester_stats_sum_score::NUMERIC
          / NULLIF(
              (user_class_attendance_semester_stats_present_count
             + user_class_attendance_semester_stats_sick_count
             + user_class_attendance_semester_stats_leave_count
             + user_class_attendance_semester_stats_absent_count),
              0
            )
        )
      END
    ) STORED,

  user_class_attendance_semester_stats_grade_passed_count INT,
  user_class_attendance_semester_stats_grade_failed_count INT,

  user_class_attendance_semester_stats_last_aggregated_at TIMESTAMPTZ,
  user_class_attendance_semester_stats_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_attendance_semester_stats_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  /* ================== FK KOMPOSIT (tenant-safe) ================== */
  CONSTRAINT fk_ucass_section
    FOREIGN KEY (user_class_attendance_semester_stats_section_id,
                 user_class_attendance_semester_stats_school_id)
    REFERENCES class_sections (class_section_id, class_section_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_ucass_term
    FOREIGN KEY (user_class_attendance_semester_stats_term_id,
                 user_class_attendance_semester_stats_school_id)
    REFERENCES academic_terms (academic_term_id, academic_term_school_id)
    ON UPDATE CASCADE ON DELETE RESTRICT
);



-- =========================================================
-- TRIGGER: validasi tenant & periode vs term (tanpa user_classes)
-- =========================================================
CREATE OR REPLACE FUNCTION fn_ucass_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_sec_school UUID;
  v_term_school UUID;
  v_term_period DATERANGE;
  v_row_period  DATERANGE;
BEGIN
  -- class_sections → pastikan tenant cocok
  SELECT class_section_school_id
    INTO v_sec_school
  FROM class_sections
  WHERE class_section_id = NEW.user_class_attendance_semester_stats_section_id
    AND class_section_deleted_at IS NULL;

  IF v_sec_school IS NULL THEN
    RAISE EXCEPTION 'Section % tidak ditemukan/terhapus',
      NEW.user_class_attendance_semester_stats_section_id;
  END IF;

  IF NEW.user_class_attendance_semester_stats_school_id <> v_sec_school THEN
    RAISE EXCEPTION 'School mismatch: row(%) vs section(%)',
      NEW.user_class_attendance_semester_stats_school_id, v_sec_school;
  END IF;

  -- Jika ada term → cek tenant & cakupan periode
  IF NEW.user_class_attendance_semester_stats_term_id IS NOT NULL THEN
    SELECT academic_term_school_id, academic_term_period
      INTO v_term_school, v_term_period
    FROM academic_terms
    WHERE academic_term_id = NEW.user_class_attendance_semester_stats_term_id
      AND academic_term_deleted_at IS NULL;

    IF v_term_school IS NULL THEN
      RAISE EXCEPTION 'Term % tidak ditemukan/terhapus',
        NEW.user_class_attendance_semester_stats_term_id;
    END IF;

    IF v_term_school <> NEW.user_class_attendance_semester_stats_school_id THEN
      RAISE EXCEPTION 'School mismatch: term(%) != row(%)',
        v_term_school, NEW.user_class_attendance_semester_stats_school_id;
    END IF;

    -- Row range: [start, end] → konversi ke half-open [start, end+1) untuk bandingkan ke academic_term_period
    v_row_period := daterange(
      NEW.user_class_attendance_semester_stats_period_start,
      (NEW.user_class_attendance_semester_stats_period_end + 1),
      '[)'
    );

    IF NOT (v_row_period <@ v_term_period) THEN
      RAISE EXCEPTION 'Periode row [% .. %] di luar term',
        NEW.user_class_attendance_semester_stats_period_start,
        NEW.user_class_attendance_semester_stats_period_end;
    END IF;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_ucass_validate_links') THEN
    DROP TRIGGER trg_ucass_validate_links ON user_class_attendance_semester_stats;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_ucass_validate_links
    AFTER INSERT OR UPDATE OF
      user_class_attendance_semester_stats_school_id,
      user_class_attendance_semester_stats_section_id,
      user_class_attendance_semester_stats_term_id,
      user_class_attendance_semester_stats_period_start,
      user_class_attendance_semester_stats_period_end
    ON user_class_attendance_semester_stats
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_ucass_validate_links();
END$$;
