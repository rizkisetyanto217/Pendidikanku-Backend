package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateAssessmentRequest
type CreateAssessmentRequest struct {
	MasjidID                     uuid.UUID  `json:"assessments_masjid_id" validate:"required"`
	ClassSectionID               *uuid.UUID `json:"assessments_class_section_id" validate:"omitempty"`
	ClassSubjectsID              *uuid.UUID `json:"assessments_class_subjects_id" validate:"omitempty"`
	ClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id" validate:"omitempty"`

	TypeID *uuid.UUID `json:"assessments_type_id" validate:"required"`

	Title       string  `json:"assessments_title" validate:"required,max=180"`
	Description *string `json:"assessments_description" validate:"omitempty"`

	StartAt *time.Time `json:"assessments_start_at" validate:"omitempty"`
	DueAt   *time.Time `json:"assessments_due_at" validate:"omitempty"`

	MaxScore float32 `json:"assessments_max_score" validate:"gte=0,lte=100"`

	IsPublished     *bool `json:"assessments_is_published" validate:"omitempty"`
	AllowSubmission *bool `json:"assessments_allow_submission" validate:"omitempty"`

	CreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id" validate:"omitempty"`
}

// PatchAssessmentRequest (partial update, PATCH)
type PatchAssessmentRequest struct {
	Title       *string `json:"assessments_title" validate:"omitempty,max=180"`
	Description *string `json:"assessments_description" validate:"omitempty"`

	StartAt *time.Time `json:"assessments_start_at" validate:"omitempty"`
	DueAt   *time.Time `json:"assessments_due_at" validate:"omitempty"`

	MaxScore *float32 `json:"assessments_max_score" validate:"omitempty,gte=0,lte=100"`

	IsPublished     *bool `json:"assessments_is_published" validate:"omitempty"`
	AllowSubmission *bool `json:"assessments_allow_submission" validate:"omitempty"`

	ClassSectionID               *uuid.UUID `json:"assessments_class_section_id" validate:"omitempty"`
	ClassSubjectsID              *uuid.UUID `json:"assessments_class_subjects_id" validate:"omitempty"`
	ClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id" validate:"omitempty"`
	TypeID                       *uuid.UUID `json:"assessments_type_id" validate:"omitempty"`

	CreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id" validate:"omitempty"`
}

// Response DTOs
type AssessmentResponse struct {
	ID                           uuid.UUID  `json:"assessments_id"`
	MasjidID                     uuid.UUID  `json:"assessments_masjid_id"`
	ClassSectionID               *uuid.UUID `json:"assessments_class_section_id,omitempty"`
	ClassSubjectsID              *uuid.UUID `json:"assessments_class_subjects_id,omitempty"`
	ClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id,omitempty"`
	TypeID                       *uuid.UUID `json:"assessments_type_id,omitempty"`

	Title       string  `json:"assessments_title"`
	Description *string `json:"assessments_description,omitempty"`

	StartAt *time.Time `json:"assessments_start_at,omitempty"`
	DueAt   *time.Time `json:"assessments_due_at,omitempty"`

	MaxScore float32 `json:"assessments_max_score"`

	IsPublished     bool `json:"assessments_is_published"`
	AllowSubmission bool `json:"assessments_allow_submission"`

	CreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id,omitempty"`

	CreatedAt time.Time  `json:"assessments_created_at"`
	UpdatedAt time.Time  `json:"assessments_updated_at"`
	DeletedAt *time.Time `json:"assessments_deleted_at,omitempty"`
}

type ListAssessmentResponse struct {
	Data   []AssessmentResponse `json:"data"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}
