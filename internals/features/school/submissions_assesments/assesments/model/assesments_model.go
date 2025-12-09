// file: internals/features/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================
// Enum: Submission Mode
// =========================

type AssessmentSubmissionMode string

const (
	SubmissionModeDate    AssessmentSubmissionMode = "date"
	SubmissionModeSession AssessmentSubmissionMode = "session"
)

// =========================
// Enum: Assessment Kind
// =========================

type AssessmentKind string

const (
	AssessmentKindQuiz             AssessmentKind = "quiz"
	AssessmentKindAssignmentUpload AssessmentKind = "assignment_upload"
	AssessmentKindOffline          AssessmentKind = "offline"
	AssessmentKindSurvey           AssessmentKind = "survey"
)

// =========================
// Enum: Assessment Type Category
// (training / daily_exam / exam)
// =========================

type AssessmentTypeCategory string

const (
	AssessmentTypeCategoryTraining  AssessmentTypeCategory = "training"
	AssessmentTypeCategoryDailyExam AssessmentTypeCategory = "daily_exam"
	AssessmentTypeCategoryExam      AssessmentTypeCategory = "exam"
)

// =========================
// Model: assessments
// =========================

type AssessmentModel struct {
	// PK & Tenant
	AssessmentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_id" json:"assessment_id"`
	AssessmentSchoolID uuid.UUID `gorm:"type:uuid;not null;column:assessment_school_id" json:"assessment_school_id"`

	// Relasi ke CSST (single FK)
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_class_section_subject_teacher_id" json:"assessment_class_section_subject_teacher_id,omitempty"`

	// Tipe penilaian (kategori akademik)
	AssessmentTypeID *uuid.UUID `gorm:"type:uuid;column:assessment_type_id" json:"assessment_type_id,omitempty"`

	// Snapshot kategori type (training / daily_exam / exam)
	AssessmentTypeCategorySnapshot AssessmentTypeCategory `gorm:"type:assessment_type_enum;column:assessment_type_category_snapshot" json:"assessment_type_category_snapshot"`

	// Identitas
	AssessmentSlug        *string `gorm:"type:varchar(160);column:assessment_slug" json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `gorm:"type:varchar(180);not null;column:assessment_title" json:"assessment_title"`
	AssessmentDescription *string `gorm:"type:text;column:assessment_description" json:"assessment_description,omitempty"`

	// Jadwal by date
	AssessmentStartAt     *time.Time `gorm:"type:timestamptz;column:assessment_start_at" json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `gorm:"type:timestamptz;column:assessment_due_at" json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `gorm:"type:timestamptz;column:assessment_published_at" json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `gorm:"type:timestamptz;column:assessment_closed_at" json:"assessment_closed_at,omitempty"`

	// Pengaturan dasar assessment
	AssessmentKind                 AssessmentKind `gorm:"type:assessment_kind_enum;not null;default:'quiz';column:assessment_kind" json:"assessment_kind"`
	AssessmentDurationMinutes      *int           `gorm:"type:int;column:assessment_duration_minutes" json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int            `gorm:"type:int;not null;default:1;column:assessment_total_attempts_allowed" json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64        `gorm:"type:numeric(5,2);not null;default:100;column:assessment_max_score" json:"assessment_max_score"`

	// total quiz/komponen quiz di assessment ini (global, sama untuk semua siswa)
	AssessmentQuizTotal int `gorm:"type:smallint;not null;default:0;column:assessment_quiz_total" json:"assessment_quiz_total"`

	// agregat submissions (diupdate dari service)
	AssessmentSubmissionsTotal       int `gorm:"type:int;not null;default:0;column:assessment_submissions_total" json:"assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `gorm:"type:int;not null;default:0;column:assessment_submissions_graded_total" json:"assessment_submissions_graded_total"`

	AssessmentIsPublished     bool `gorm:"not null;default:true;column:assessment_is_published" json:"assessment_is_published"`
	AssessmentAllowSubmission bool `gorm:"not null;default:true;column:assessment_allow_submission" json:"assessment_allow_submission"`

	// Flag apakah assessment type ini menghasilkan nilai (graded) — snapshot
	AssessmentTypeIsGradedSnapshot bool `gorm:"not null;default:false;column:assessment_type_is_graded_snapshot" json:"assessment_type_is_graded_snapshot"`

	// =========================
	// Snapshot aturan dari AssessmentType (per assessment)
	// HANYA grading & late policy (sesuai SQL)
	// =========================
	AssessmentAllowLateSubmissionSnapshot bool    `gorm:"not null;default:false;column:assessment_allow_late_submission_snapshot" json:"assessment_allow_late_submission_snapshot"`
	AssessmentLatePenaltyPercentSnapshot  float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_late_penalty_percent_snapshot" json:"assessment_late_penalty_percent_snapshot"`
	AssessmentPassingScorePercentSnapshot float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_passing_score_percent_snapshot" json:"assessment_passing_score_percent_snapshot"`

	// Audit pembuat (opsional)
	AssessmentCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_created_by_teacher_id" json:"assessment_created_by_teacher_id,omitempty"`

	// Mode pengumpulan (by date / by session)
	AssessmentSubmissionMode    AssessmentSubmissionMode `gorm:"type:text;not null;default:'date';column:assessment_submission_mode" json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID               `gorm:"type:uuid;column:assessment_announce_session_id" json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID               `gorm:"type:uuid;column:assessment_collect_session_id" json:"assessment_collect_session_id,omitempty"`

	// Timestamps & soft delete
	AssessmentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_created_at" json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_updated_at" json:"assessment_updated_at"`
	AssessmentDeletedAt gorm.DeletedAt `gorm:"column:assessment_deleted_at;index" json:"assessment_deleted_at,omitempty"`
}

func (AssessmentModel) TableName() string { return "assessments" }
