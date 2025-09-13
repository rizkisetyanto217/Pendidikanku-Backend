-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =========================================
-- USER PROGRESS REPORTS (HEADER)
-- =========================================
CREATE TABLE IF NOT EXISTS user_progress_reports (
  user_progress_reports_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant scope
  user_progress_reports_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- target siswa (masjid_students)
  user_progress_reports_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- opsional: kelas
  user_progress_reports_class_section_id UUID
    REFERENCES class_sections(class_sections_id) ON DELETE SET NULL,

  -- periode laporan
  user_progress_reports_period_type VARCHAR(10) NOT NULL DEFAULT 'monthly'
    CHECK (user_progress_reports_period_type IN ('weekly','monthly','custom')),
  user_progress_reports_period_start DATE NOT NULL,
  user_progress_reports_period_end   DATE NOT NULL,
  CHECK (user_progress_reports_period_start <= user_progress_reports_period_end),

  -- metadata
  user_progress_reports_title VARCHAR(150),
  user_progress_reports_summary TEXT,

  -- ringkasan attendance
  user_progress_reports_attendance_present  INT NOT NULL DEFAULT 0,
  user_progress_reports_attendance_absent   INT NOT NULL DEFAULT 0,
  user_progress_reports_attendance_excused  INT NOT NULL DEFAULT 0,
  user_progress_reports_attendance_late     INT NOT NULL DEFAULT 0,

  -- ringkasan tugas/nilai
  user_progress_reports_total_tasks     INT NOT NULL DEFAULT 0,
  user_progress_reports_completed_tasks INT NOT NULL DEFAULT 0,
  user_progress_reports_average_score   NUMERIC(5,2),
  user_progress_reports_rank_in_class   INT,

  -- snapshot JSON (data beku)
  user_progress_reports_attendance_snapshot JSONB,
  user_progress_reports_scores_snapshot     JSONB,
  user_progress_reports_behavior_snapshot   JSONB,

  -- catatan tambahan naratif
  user_progress_reports_additional_notes TEXT,

  -- jejak pembuat
  user_progress_reports_generated_by_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,
  user_progress_reports_generated_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- status & approval
  user_progress_reports_status VARCHAR(16) NOT NULL DEFAULT 'draft'
    CHECK (user_progress_reports_status IN ('draft','pending','final','archived')),
  user_progress_reports_approved_by_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,
  user_progress_reports_approved_at TIMESTAMPTZ,

  -- audit
  user_progress_reports_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_progress_reports_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_progress_reports_deleted_at TIMESTAMPTZ
);

-- Indexes untuk reports
CREATE INDEX IF NOT EXISTS idx_user_progress_reports_masjid_student_period
  ON user_progress_reports(user_progress_reports_masjid_id, user_progress_reports_student_id, user_progress_reports_period_start, user_progress_reports_period_end);

CREATE INDEX IF NOT EXISTS idx_user_progress_reports_class_period
  ON user_progress_reports(user_progress_reports_class_section_id, user_progress_reports_period_start, user_progress_reports_period_end);

CREATE INDEX IF NOT EXISTS idx_user_progress_reports_status
  ON user_progress_reports(user_progress_reports_status);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_progress_reports_student_period_exact
  ON user_progress_reports(user_progress_reports_student_id, user_progress_reports_period_start, user_progress_reports_period_end)
  WHERE user_progress_reports_deleted_at IS NULL;

-- =========================================
-- LINK: REPORT ↔ USER NOTES
-- =========================================
CREATE TABLE IF NOT EXISTS user_progress_report_user_notes (
  user_progress_report_user_notes_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_progress_report_user_notes_report_id UUID NOT NULL
    REFERENCES user_progress_reports(user_progress_reports_id) ON DELETE CASCADE,

  user_progress_report_user_notes_user_note_id UUID
    REFERENCES user_notes(user_note_id) ON DELETE SET NULL,

  -- salinan beku
  user_progress_report_user_notes_title_copy   VARCHAR(150),
  user_progress_report_user_notes_content_copy TEXT NOT NULL,
  user_progress_report_user_notes_type_name    VARCHAR(80),
  user_progress_report_user_notes_labels_copy  TEXT[] DEFAULT '{}',
  user_progress_report_user_notes_priority     VARCHAR(8)
    CHECK (user_progress_report_user_notes_priority IN ('low','med','high')),

  user_progress_report_user_notes_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_progress_report_user_notes_report
  ON user_progress_report_user_notes(user_progress_report_user_notes_report_id);

-- =========================================
-- RECEIPTS (ACKNOWLEDGEMENT ORANG TUA / WALI)
-- =========================================
CREATE TABLE IF NOT EXISTS user_progress_report_receipts (
  user_progress_report_receipts_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_progress_report_receipts_report_id UUID NOT NULL
    REFERENCES user_progress_reports(user_progress_reports_id) ON DELETE CASCADE,

  -- parent/wali (masjid_students)
  user_progress_report_receipts_parent_student_id UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE SET NULL,

  user_progress_report_receipts_acknowledged_at TIMESTAMPTZ,
  user_progress_report_receipts_comment TEXT,

  user_progress_report_receipts_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_progress_report_receipts_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_progress_report_receipts_report_parent_student
  ON user_progress_report_receipts(user_progress_report_receipts_report_id, user_progress_report_receipts_parent_student_id);

-- =========================================
-- APPROVAL / ACTION LOG
-- =========================================
CREATE TABLE IF NOT EXISTS user_progress_report_approvals (
  user_progress_report_approvals_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_progress_report_approvals_report_id UUID NOT NULL
    REFERENCES user_progress_reports(user_progress_reports_id) ON DELETE CASCADE,

  user_progress_report_approvals_action VARCHAR(16) NOT NULL
    CHECK (user_progress_report_approvals_action IN ('submit','approve','reject','finalize','archive')),

  user_progress_report_approvals_actor_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  user_progress_report_approvals_note TEXT,
  user_progress_report_approvals_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_progress_report_approvals_report
  ON user_progress_report_approvals(user_progress_report_approvals_report_id);
