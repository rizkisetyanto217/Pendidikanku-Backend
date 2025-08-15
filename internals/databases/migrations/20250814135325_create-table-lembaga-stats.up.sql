CREATE TABLE IF NOT EXISTS lembaga_stats (
  lembaga_stats_lembaga_id UUID PRIMARY KEY
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lembaga_stats_active_classes   INT NOT NULL DEFAULT 0, -- kelas aktif
  lembaga_stats_active_sections  INT NOT NULL DEFAULT 0, -- section aktif
  lembaga_stats_active_students  INT NOT NULL DEFAULT 0, -- siswa aktif
  lembaga_stats_active_teachers  INT NOT NULL DEFAULT 0, -- guru aktif (role-based)

  lembaga_stats_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  lembaga_stats_updated_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lembaga_stats_updated_at
  ON lembaga_stats(lembaga_stats_updated_at);

CREATE TABLE IF NOT EXISTS user_class_attendance_semester_stats (
  user_class_attendance_semester_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_class_attendance_semester_stats_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_attendance_semester_stats_user_class_id UUID NOT NULL
    REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,

  user_class_attendance_semester_stats_section_id UUID NOT NULL
    REFERENCES class_sections(class_sections_id) ON DELETE CASCADE,

  user_class_attendance_semester_stats_period_start DATE NOT NULL,
  user_class_attendance_semester_stats_period_end   DATE NOT NULL,
  CONSTRAINT chk_ucass_period_range
    CHECK (user_class_attendance_semester_stats_period_start <= user_class_attendance_semester_stats_period_end),

  user_class_attendance_semester_stats_present_count INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_present_count >= 0),
  user_class_attendance_semester_stats_sick_count    INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_sick_count    >= 0),
  user_class_attendance_semester_stats_leave_count   INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_leave_count   >= 0),
  user_class_attendance_semester_stats_absent_count  INT NOT NULL DEFAULT 0 CHECK (user_class_attendance_semester_stats_absent_count  >= 0),

  -- generated dari 4 counter dasar
  user_class_attendance_semester_stats_total_sessions INT
    GENERATED ALWAYS AS (
      (user_class_attendance_semester_stats_present_count
     + user_class_attendance_semester_stats_sick_count
     + user_class_attendance_semester_stats_leave_count
     + user_class_attendance_semester_stats_absent_count)
    ) STORED,

  user_class_attendance_semester_stats_sum_score INT,

  -- JANGAN refer ke kolom generated lain (total_sessions). Hitung ulang penyebutnya.
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

  user_class_attendance_semester_stats_last_aggregated_at TIMESTAMP,
  user_class_attendance_semester_stats_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_class_attendance_semester_stats_updated_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ucass_tenant_userclass_section_period
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_user_class_id,
    user_class_attendance_semester_stats_section_id,
    user_class_attendance_semester_stats_period_start,
    user_class_attendance_semester_stats_period_end
  );

CREATE INDEX IF NOT EXISTS ix_ucass_masjid_section_period
  ON user_class_attendance_semester_stats (
    user_class_attendance_semester_stats_masjid_id,
    user_class_attendance_semester_stats_section_id,
    user_class_attendance_semester_stats_period_start,
    user_class_attendance_semester_stats_period_end
  );

CREATE INDEX IF NOT EXISTS ix_ucass_userclass
  ON user_class_attendance_semester_stats (user_class_attendance_semester_stats_user_class_id);
