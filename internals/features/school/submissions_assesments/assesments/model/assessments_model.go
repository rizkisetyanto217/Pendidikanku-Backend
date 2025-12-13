// file: internals/features/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	quizModel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
)

//
// ========================================================
// ENUM DEFINITIONS
// ========================================================
//

// ----- Submission Mode -----
type AssessmentSubmissionMode string

const (
	SubmissionModeDate    AssessmentSubmissionMode = "date"
	SubmissionModeSession AssessmentSubmissionMode = "session"
)

// ----- Assessment Kind -----
type AssessmentKind string

const (
	AssessmentKindQuiz             AssessmentKind = "quiz"
	AssessmentKindAssignmentUpload AssessmentKind = "assignment_upload"
	AssessmentKindOffline          AssessmentKind = "offline"
	AssessmentKindSurvey           AssessmentKind = "survey"
)

// ----- Assessment Type Category (SNAPSHOT) -----
type AssessmentTypeCategory string

const (
	AssessmentTypeCategoryTraining  AssessmentTypeCategory = "training"
	AssessmentTypeCategoryDailyExam AssessmentTypeCategory = "daily_exam"
	AssessmentTypeCategoryExam      AssessmentTypeCategory = "exam"
)

// ----- Assessment Status -----
type AssessmentStatus string

const (
	AssessmentStatusDraft     AssessmentStatus = "draft"
	AssessmentStatusPublished AssessmentStatus = "published"
	AssessmentStatusArchived  AssessmentStatus = "archived"
)

//
// ========================================================
// MODEL: assessments
// ========================================================
//

type AssessmentModel struct {
	// PK & Tenant
	AssessmentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_id"`
	AssessmentSchoolID uuid.UUID `gorm:"type:uuid;not null;column:assessment_school_id"`

	// Relasi ke CSST
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_class_section_subject_teacher_id"`

	// Relasi ke AssessmentType (FK saja)
	AssessmentTypeID *uuid.UUID `gorm:"type:uuid;column:assessment_type_id"`

	// SNAPSHOT kategori type (training / daily_exam / exam)
	AssessmentTypeCategorySnapshot AssessmentTypeCategory `gorm:"type:assessment_type_enum;column:assessment_type_category_snapshot"`

	// Identitas
	AssessmentSlug        *string `gorm:"type:varchar(160);column:assessment_slug"`
	AssessmentTitle       string  `gorm:"type:varchar(180);not null;column:assessment_title"`
	AssessmentDescription *string `gorm:"type:text;column:assessment_description"`

	// Jadwal by date
	AssessmentStartAt     *time.Time `gorm:"type:timestamptz;column:assessment_start_at"`
	AssessmentDueAt       *time.Time `gorm:"type:timestamptz;column:assessment_due_at"`

	// Pengaturan
	AssessmentKind                 AssessmentKind   `gorm:"type:assessment_kind_enum;not null;default:'quiz';column:assessment_kind"`
	AssessmentStatus               AssessmentStatus `gorm:"type:assessment_status_enum;not null;default:'draft';column:assessment_status"`
	AssessmentTotalAttemptsAllowed int              `gorm:"type:int;not null;default:1;column:assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64          `gorm:"type:numeric(5,2);not null;default:100;column:assessment_max_score"`

	// Quiz Count
	AssessmentQuizTotal int `gorm:"type:smallint;not null;default:0;column:assessment_quiz_total"`

	// Aggregates
	AssessmentSubmissionsTotal       int `gorm:"type:int;not null;default:0;column:assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `gorm:"type:int;not null;default:0;column:assessment_submissions_graded_total"`

	// SNAPSHOT flags dari AssessmentType
	AssessmentTypeIsGradedSnapshot        bool    `gorm:"not null;default:false;column:assessment_type_is_graded_snapshot"`
	AssessmentAllowLateSubmissionSnapshot bool    `gorm:"not null;default:false;column:assessment_allow_late_submission_snapshot"`
	AssessmentLatePenaltyPercentSnapshot  float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_late_penalty_percent_snapshot"`
	AssessmentMinPassingScoreClassSubjectSnapshot     float64 `gorm:"type:numeric(5,2);not null;default:0;column:assesment_min_passing_score_class_subject_snapshot"`

	// Creator
	AssessmentCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_created_by_teacher_id"`

	// Submission mode: date or session
	AssessmentSubmissionMode    AssessmentSubmissionMode `gorm:"type:text;not null;default:'date';column:assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID               `gorm:"type:uuid;column:assessment_announce_session_id"`
	AssessmentCollectSessionID  *uuid.UUID               `gorm:"type:uuid;column:assessment_collect_session_id"`

	// Timestamps
	AssessmentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_created_at"`
	AssessmentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_updated_at"`
	AssessmentDeletedAt gorm.DeletedAt `gorm:"column:assessment_deleted_at;index"`

	// ========================================================
	// ⭐ RELASI QUIZZES (PRELOAD OPTIONAL)
	// ========================================================
	Quizzes []quizModel.QuizModel `gorm:"foreignKey:QuizAssessmentID;references:AssessmentID" json:"-"`
}

func (AssessmentModel) TableName() string { return "assessments" }
