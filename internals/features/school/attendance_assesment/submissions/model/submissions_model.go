// file: internals/features/school/submissions/model/submission_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// enum status agar aman di kode
type SubmissionStatus string

const (
	SubmissionStatusDraft       SubmissionStatus = "draft"
	SubmissionStatusSubmitted   SubmissionStatus = "submitted"
	SubmissionStatusResubmitted SubmissionStatus = "resubmitted"
	SubmissionStatusGraded      SubmissionStatus = "graded"
	SubmissionStatusReturned    SubmissionStatus = "returned"
)

type Submission struct {
	// primary key
	SubmissionID uuid.UUID `json:"submissions_id" gorm:"column:submissions_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// keterkaitan tenant & entitas
	SubmissionMasjidID     uuid.UUID `json:"submissions_masjid_id" gorm:"column:submissions_masjid_id;type:uuid;not null"`
	SubmissionAssessmentID uuid.UUID `json:"submissions_assessment_id" gorm:"column:submissions_assessment_id;type:uuid;not null"`
	SubmissionUserID       uuid.UUID `json:"submissions_user_id" gorm:"column:submissions_user_id;type:uuid;not null"`

	// isi & status pengumpulan
	SubmissionText   *string          `json:"submissions_text,omitempty" gorm:"column:submissions_text;type:text"`
	SubmissionStatus SubmissionStatus `json:"submissions_status" gorm:"column:submissions_status;type:varchar(24);not null;default:'submitted'"`

	SubmissionSubmittedAt *time.Time `json:"submissions_submitted_at,omitempty" gorm:"column:submissions_submitted_at;type:timestamptz"`
	SubmissionIsLate      *bool      `json:"submissions_is_late,omitempty" gorm:"column:submissions_is_late"`

	// penilaian
	SubmissionScore    *float64   `json:"submissions_score,omitempty" gorm:"column:submissions_score;type:numeric(5,2)"`
	SubmissionFeedback *string    `json:"submissions_feedback,omitempty" gorm:"column:submissions_feedback;type:text"`
	SubmissionGradedBy *uuid.UUID `json:"submissions_graded_by_teacher_id,omitempty" gorm:"column:submissions_graded_by_teacher_id;type:uuid"`
	SubmissionGradedAt *time.Time `json:"submissions_graded_at,omitempty" gorm:"column:submissions_graded_at;type:timestamptz"`

	// timestamps & soft delete
	SubmissionCreatedAt time.Time      `json:"submissions_created_at" gorm:"column:submissions_created_at;type:timestamptz;autoCreateTime"`
	SubmissionUpdatedAt time.Time      `json:"submissions_updated_at" gorm:"column:submissions_updated_at;type:timestamptz;autoUpdateTime"`
	SubmissionDeletedAt gorm.DeletedAt `json:"submissions_deleted_at" gorm:"column:submissions_deleted_at;index"`
}

// TableName override
func (Submission) TableName() string {
	return "submissions"
}
