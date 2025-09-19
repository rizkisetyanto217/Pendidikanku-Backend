// file: internals/features/school/assessments/dto/assessment_dto.go
package dto

import (
	"masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	"time"

	"github.com/google/uuid"
)

/*
	==============================
	  CREATE (POST /assessments)

==============================
*/
type CreateAssessmentRequest struct {
	AssessmentsMasjidID                     uuid.UUID  `json:"assessments_masjid_id" validate:"required"`
	AssessmentsClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id" validate:"omitempty"`
	AssessmentsTypeID                       *uuid.UUID `json:"assessments_type_id" validate:"omitempty"`

	AssessmentsTitle       string  `json:"assessments_title" validate:"required,max=180"`
	AssessmentsDescription *string `json:"assessments_description" validate:"omitempty"`

	AssessmentsStartAt *time.Time `json:"assessments_start_at" validate:"omitempty"`
	AssessmentsDueAt   *time.Time `json:"assessments_due_at" validate:"omitempty"`

	AssessmentsMaxScore *float64 `json:"assessments_max_score" validate:"omitempty,gte=0,lte=100"`

	AssessmentsIsPublished     *bool `json:"assessments_is_published" validate:"omitempty"`
	AssessmentsAllowSubmission *bool `json:"assessments_allow_submission" validate:"omitempty"`

	AssessmentsCreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id" validate:"omitempty"`
}

/*
	==============================
	  PATCH (PATCH /assessments/:id)

==============================
*/
type PatchAssessmentRequest struct {
	AssessmentsTitle       *string `json:"assessments_title" validate:"omitempty,max=180"`
	AssessmentsDescription *string `json:"assessments_description" validate:"omitempty"`

	AssessmentsStartAt *time.Time `json:"assessments_start_at" validate:"omitempty"`
	AssessmentsDueAt   *time.Time `json:"assessments_due_at" validate:"omitempty"`

	AssessmentsMaxScore *float64 `json:"assessments_max_score" validate:"omitempty,gte=0,lte=100"`

	AssessmentsIsPublished     *bool `json:"assessments_is_published" validate:"omitempty"`
	AssessmentsAllowSubmission *bool `json:"assessments_allow_submission" validate:"omitempty"`

	AssessmentsClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id" validate:"omitempty"`
	AssessmentsTypeID                       *uuid.UUID `json:"assessments_type_id" validate:"omitempty"`

	AssessmentsCreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id" validate:"omitempty"`
}

/*
	==============================
	  RESPONSE DTOs

==============================
*/
type AssessmentResponse struct {
	AssessmentsID                           uuid.UUID  `json:"assessments_id"`
	AssessmentsMasjidID                     uuid.UUID  `json:"assessments_masjid_id"`
	AssessmentsClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id,omitempty"`
	AssessmentsTypeID                       *uuid.UUID `json:"assessments_type_id,omitempty"`

	AssessmentsTitle       string  `json:"assessments_title"`
	AssessmentsDescription *string `json:"assessments_description,omitempty"`

	AssessmentsStartAt *time.Time `json:"assessments_start_at,omitempty"`
	AssessmentsDueAt   *time.Time `json:"assessments_due_at,omitempty"`

	AssessmentsMaxScore float64 `json:"assessments_max_score"`

	AssessmentsIsPublished     bool `json:"assessments_is_published"`
	AssessmentsAllowSubmission bool `json:"assessments_allow_submission"`

	AssessmentsCreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id,omitempty"`

	AssessmentsCreatedAt time.Time  `json:"assessments_created_at"`
	AssessmentsUpdatedAt time.Time  `json:"assessments_updated_at"`
	AssessmentsDeletedAt *time.Time `json:"assessments_deleted_at,omitempty"`
}

type ListAssessmentResponse struct {
	Data   []AssessmentResponse `json:"data"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// toResponse memetakan model -> DTO respons
func ToResponse(m *model.AssessmentModel) AssessmentResponse {
	var deletedAt *time.Time
	if m.AssessmentsDeletedAt.Valid {
		t := m.AssessmentsDeletedAt.Time
		deletedAt = &t
	}

	return AssessmentResponse{
		AssessmentsID:                           m.AssessmentsID,
		AssessmentsMasjidID:                     m.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: m.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       m.AssessmentsTypeID,

		AssessmentsTitle:       m.AssessmentsTitle,
		AssessmentsDescription: m.AssessmentsDescription,

		AssessmentsStartAt: m.AssessmentsStartAt,
		AssessmentsDueAt:   m.AssessmentsDueAt,

		AssessmentsMaxScore: m.AssessmentsMaxScore,

		AssessmentsIsPublished:     m.AssessmentsIsPublished,
		AssessmentsAllowSubmission: m.AssessmentsAllowSubmission,

		AssessmentsCreatedByTeacherID: m.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: m.AssessmentsCreatedAt,
		AssessmentsUpdatedAt: m.AssessmentsUpdatedAt,
		AssessmentsDeletedAt: deletedAt,
	}
}
