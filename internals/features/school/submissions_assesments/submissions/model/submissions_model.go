// file: internals/features/assessments/submissions/model/submission_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Sesuaikan dengan CHECK: 'draft','submitted','resubmitted','graded','returned'
type SubmissionStatus string

const (
	SubmissionStatusDraft       SubmissionStatus = "draft"
	SubmissionStatusSubmitted   SubmissionStatus = "submitted"
	SubmissionStatusResubmitted SubmissionStatus = "resubmitted"
	SubmissionStatusGraded      SubmissionStatus = "graded"
	SubmissionStatusReturned    SubmissionStatus = "returned"
)

type SubmissionModel struct {
	SubmissionID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:submission_id" json:"submission_id"`
	SubmissionSchoolID uuid.UUID `gorm:"type:uuid;not null;column:submission_school_id" json:"submission_school_id"`

	SubmissionAssessmentID uuid.UUID `gorm:"type:uuid;not null;column:submission_assessment_id" json:"submission_assessment_id"`
	SubmissionStudentID    uuid.UUID `gorm:"type:uuid;not null;column:submission_student_id" json:"submission_student_id"`

	// ✅ attempt ke berapa (1..n) — sesuai SQL terbaru
	SubmissionAttemptCount int `gorm:"type:int;not null;default:1;column:submission_attempt_count" json:"submission_attempt_count"`

	// Isi & status pengumpulan
	SubmissionText   *string          `gorm:"type:text;column:submission_text" json:"submission_text,omitempty"`
	SubmissionStatus SubmissionStatus `gorm:"type:varchar(24);not null;default:'submitted';column:submission_status" json:"submission_status"`

	SubmissionSubmittedAt *time.Time `gorm:"type:timestamptz;column:submission_submitted_at" json:"submission_submitted_at,omitempty"`

	// ✅ non-nullable + default false (bukan pointer)
	SubmissionIsLate bool `gorm:"not null;default:false;column:submission_is_late" json:"submission_is_late"`

	// Nilai akhir
	SubmissionScore *float64 `gorm:"type:numeric(5,2);column:submission_score" json:"submission_score,omitempty"`

	// Breakdown nilai per komponen dalam bentuk JSONB
	SubmissionScores datatypes.JSONMap `gorm:"type:jsonb;column:submission_scores" json:"submission_scores,omitempty"`

	// Berapa quiz/komponen yang sudah benar-benar selesai
	SubmissionQuizFinished int `gorm:"type:smallint;not null;default:0;column:submission_quiz_finished" json:"submission_quiz_finished"`

	SubmissionFeedback *string `gorm:"type:text;column:submission_feedback" json:"submission_feedback,omitempty"`

	SubmissionGradedByTeacherID *uuid.UUID `gorm:"type:uuid;column:submission_graded_by_teacher_id" json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt          *time.Time `gorm:"type:timestamptz;column:submission_graded_at" json:"submission_graded_at,omitempty"`

	SubmissionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:submission_created_at" json:"submission_created_at"`
	SubmissionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:submission_updated_at" json:"submission_updated_at"`
	SubmissionDeletedAt gorm.DeletedAt `gorm:"column:submission_deleted_at;index" json:"submission_deleted_at,omitempty"`
}

func (SubmissionModel) TableName() string { return "submissions" }
