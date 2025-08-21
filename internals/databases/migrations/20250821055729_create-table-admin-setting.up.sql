CREATE TABLE IF NOT EXISTS class_attendance_settings (
  class_attendance_setting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- pakai masjids(masjid_id), bukan masjids(id)
  class_attendance_setting_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- switches
  class_attendance_setting_enable_score               BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_score              BOOLEAN NOT NULL DEFAULT false,

  class_attendance_setting_enable_grade_passed        BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_grade_passed       BOOLEAN NOT NULL DEFAULT false,

  class_attendance_setting_enable_material_personal   BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_material_personal  BOOLEAN NOT NULL DEFAULT false,

  class_attendance_setting_enable_personal_note       BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_personal_note      BOOLEAN NOT NULL DEFAULT false,

  class_attendance_setting_enable_memorization        BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_memorization       BOOLEAN NOT NULL DEFAULT false,

  class_attendance_setting_enable_homework            BOOLEAN NOT NULL DEFAULT false,
  class_attendance_setting_require_homework           BOOLEAN NOT NULL DEFAULT false,

  -- audit minimum: kapan dibuat
  class_attendance_setting_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- index & constraints
CREATE INDEX IF NOT EXISTS idx_cas_masjid_id
  ON class_attendance_settings(class_attendance_setting_masjid_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_masjid_unique
ON class_attendance_settings(class_attendance_setting_masjid_id);

ALTER TABLE class_attendance_settings
ADD CONSTRAINT ck_cas_require_implies_enable CHECK (
  (NOT class_attendance_setting_require_score              OR class_attendance_setting_enable_score) AND
  (NOT class_attendance_setting_require_grade_passed       OR class_attendance_setting_enable_grade_passed) AND
  (NOT class_attendance_setting_require_material_personal  OR class_attendance_setting_enable_material_personal) AND
  (NOT class_attendance_setting_require_personal_note      OR class_attendance_setting_enable_personal_note) AND
  (NOT class_attendance_setting_require_memorization       OR class_attendance_setting_enable_memorization) AND
  (NOT class_attendance_setting_require_homework           OR class_attendance_setting_enable_homework)
);
