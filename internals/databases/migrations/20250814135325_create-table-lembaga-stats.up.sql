-- =========================================================
-- lembaga_stats (global snapshot, tanpa term)
-- =========================================================
CREATE TABLE IF NOT EXISTS lembaga_stats (
  lembaga_stats_masjid_id UUID PRIMARY KEY
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lembaga_stats_active_classes  INT NOT NULL DEFAULT 0,
  lembaga_stats_active_sections INT NOT NULL DEFAULT 0,
  lembaga_stats_active_students INT NOT NULL DEFAULT 0,
  lembaga_stats_active_teachers INT NOT NULL DEFAULT 0,

  lembaga_stats_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lembaga_stats_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Guard non-negatif (idempotent)
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
-- user_class_attendance_semester_stats (per semester, pakai term)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_attendance_semester_stats (
  user_class_attendance_semester_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_attendance_semester_stats_masjid_id   UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_attendance_semester_stats_user_class_id UUID NOT NULL
    REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,

  user_class_attendance_semester_stats_section_id  UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  -- NEW: referensi resmi ke academic_terms (opsional untuk data lama)
  user_class_attendance_semester_stats_term_id UUID
    REFERENCES academic_terms(academic_terms_id) ON DELETE RESTRICT,

  -- Snapshot periode (tetap disimpan untuk audit/jejak)
  user_class_attendance_semester_stats_period_start DATE NOT NULL,
  user_class_attendance_semester_stats_period_end   DATE NOT NULL,
  CONSTRAINT chk_ucass_period_range
  CHECK (user_class_attendance_semester_stats_period_start
         <= user_class_attendance_semester_stats_period_end),

  user_class_attendance_semester_stats_present_count INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_present_count >= 0),
  user_class_attendance_semester_stats_sick_count    INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_sick_count    >= 0),
  user_class_attendance_semester_stats_leave_count   INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_leave_count   >= 0),
  user_class_attendance_semester_stats_absent_count  INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_absent_count  >= 0),

  user_class_attendance_semester_stats_total_sessions INT
    GENERATED ALWAYS AS (
      (user_class_attendance_semester_stats_present_count
     + user_class_attendance_semester_stats_sick_count
     + user_class_attendance_semester_stats_leave_count
     + user_class_attendance_semester_stats_absent_count)
    ) STORED,

  user_class_attendance_semester_stats_sum_score INT,

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
  user_class_attendance_semester_stats_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================
-- Uniques (dual-mode, idempotent)
-- ============================
-- 1) Baris TANPA term_id (kompatibilitas data lama) → unik per periode
DROP INDEX IF EXISTS uq_ucass_tenant_userclass_section_period;
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucass_tenant_userclass_section_period
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_user_class_id,
    user_class_attendance_semester_stats_section_id,
    user_class_attendance_semester_stats_period_start,
    user_class_attendance_semester_stats_period_end
  )
  WHERE user_class_attendance_semester_stats_term_id IS NULL;

-- 2) Baris DENGAN term_id → unik per term
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucass_tenant_userclass_section_term
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_user_class_id,
    user_class_attendance_semester_stats_section_id,
    user_class_attendance_semester_stats_term_id
  )
  WHERE user_class_attendance_semester_stats_term_id IS NOT NULL;

-- ============================
-- Index pendukung query
-- ============================
CREATE INDEX IF NOT EXISTS ix_ucass_userclass
  ON user_class_attendance_semester_stats (user_class_attendance_semester_stats_user_class_id);

CREATE INDEX IF NOT EXISTS ix_ucass_masjid_section_period
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_section_id,
    user_class_attendance_semester_stats_period_start,
    user_class_attendance_semester_stats_period_end
  );

CREATE INDEX IF NOT EXISTS ix_ucass_masjid_term
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_term_id
  )
  WHERE user_class_attendance_semester_stats_term_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS ix_ucass_term
  ON user_class_attendance_semester_stats (user_class_attendance_semester_stats_term_id)
  WHERE user_class_attendance_semester_stats_term_id IS NOT NULL;

-- ============================
-- Constraint trigger: validasi tenant & periode vs term
--  - Pastikan masjid konsisten antara row, user_class, section, dan term
--  - Pastikan [start,end] ⊆ academic_terms_period ([start,end) half-open)
-- ============================
CREATE OR REPLACE FUNCTION fn_ucass_validate_links()
RETURNS TRIGGER AS $$
DECLARE
  v_uc_masjid UUID;
  v_uc_class  UUID;
  v_sec_masjid UUID;
  v_sec_class  UUID;
  v_term_masjid UUID;
  v_term_period DATERANGE;
  v_row_period  DATERANGE;
BEGIN
  -- user_classes → ambil masjid & class
  SELECT user_classes_masjid_id, user_classes_class_id
    INTO v_uc_masjid, v_uc_class
  FROM user_classes
  WHERE user_classes_id = NEW.user_class_attendance_semester_stats_user_class_id;

  IF v_uc_masjid IS NULL THEN
    RAISE EXCEPTION 'user_class % tidak ditemukan', NEW.user_class_attendance_semester_stats_user_class_id;
  END IF;

  -- class_sections → ambil masjid & class
  SELECT class_sections_masjid_id, class_sections_class_id
    INTO v_sec_masjid, v_sec_class
  FROM class_sections
  WHERE class_sections_id = NEW.user_class_attendance_semester_stats_section_id
    AND class_sections_deleted_at IS NULL;

  IF v_sec_masjid IS NULL THEN
    RAISE EXCEPTION 'section % tidak ditemukan/terhapus',
      NEW.user_class_attendance_semester_stats_section_id;
  END IF;

  -- Tenant harus konsisten (row vs user_classes vs section)
  IF NEW.user_class_attendance_semester_stats_masjid_id <> v_uc_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: row(%) vs user_class(%)',
      NEW.user_class_attendance_semester_stats_masjid_id, v_uc_masjid;
  END IF;

  IF NEW.user_class_attendance_semester_stats_masjid_id <> v_sec_masjid THEN
    RAISE EXCEPTION 'Masjid mismatch: row(%) vs section(%)',
      NEW.user_class_attendance_semester_stats_masjid_id, v_sec_masjid;
  END IF;

  -- (opsional kuat) pastikan section.class == user_class.class
  IF v_uc_class IS NOT NULL AND v_sec_class IS NOT NULL AND v_uc_class <> v_sec_class THEN
    RAISE EXCEPTION 'Section.class(%) != UserClass.class(%)', v_sec_class, v_uc_class;
  END IF;

  -- Jika ada term → cek tenant & cakupan periode
  IF NEW.user_class_attendance_semester_stats_term_id IS NOT NULL THEN
    SELECT academic_terms_masjid_id, academic_terms_period
      INTO v_term_masjid, v_term_period
    FROM academic_terms
    WHERE academic_terms_id = NEW.user_class_attendance_semester_stats_term_id
      AND academic_terms_deleted_at IS NULL;

    IF v_term_masjid IS NULL THEN
      RAISE EXCEPTION 'Term % tidak ditemukan/terhapus',
        NEW.user_class_attendance_semester_stats_term_id;
    END IF;

    IF v_term_masjid <> NEW.user_class_attendance_semester_stats_masjid_id THEN
      RAISE EXCEPTION 'Masjid mismatch: term(%) != row(%)',
        v_term_masjid, NEW.user_class_attendance_semester_stats_masjid_id;
    END IF;

    -- Row range: [start, end] → ubah ke [start, end+1) agar setara dengan half-open
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
      user_class_attendance_semester_stats_masjid_id,
      user_class_attendance_semester_stats_user_class_id,
      user_class_attendance_semester_stats_section_id,
      user_class_attendance_semester_stats_term_id,
      user_class_attendance_semester_stats_period_start,
      user_class_attendance_semester_stats_period_end
    ON user_class_attendance_semester_stats
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_ucass_validate_links();
END$$;
