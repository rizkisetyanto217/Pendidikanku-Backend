// file: internals/features/school/assessments/dto/assessment_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	"github.com/google/uuid"
)

/* ==============================
   CREATE (POST /assessments)
============================== */

type CreateAssessmentRequest struct {
	// Tenant
	AssessmentMasjidID uuid.UUID `json:"assessment_masjid_id" validate:"required"`

	// Relasi
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty"`

	// Identitas
	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       string  `json:"assessment_title" validate:"required,max=180"`
	AssessmentDescription *string `json:"assessment_description" validate:"omitempty"`

	// Jadwal
	AssessmentStartAt     *time.Time `json:"assessment_start_at" validate:"omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at" validate:"omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at" validate:"omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at" validate:"omitempty"`

	// Pengaturan
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,min=1,max=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,min=1,max=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool    `json:"assessment_is_published" validate:"omitempty"`
	AssessmentAllowSubmission      *bool    `json:"assessment_allow_submission" validate:"omitempty"`

	// Audit pembuat
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty"`
}

func (r *CreateAssessmentRequest) Normalize() {
	if r.AssessmentSlug != nil {
		s := strings.TrimSpace(*r.AssessmentSlug)
		if s == "" {
			r.AssessmentSlug = nil
		} else {
			r.AssessmentSlug = &s
		}
	}
	r.AssessmentTitle = strings.TrimSpace(r.AssessmentTitle)
	if r.AssessmentDescription != nil {
		d := strings.TrimSpace(*r.AssessmentDescription)
		if d == "" {
			r.AssessmentDescription = nil
		} else {
			r.AssessmentDescription = &d
		}
	}
}

func (r CreateAssessmentRequest) ToModel() model.AssessmentModel {
	// Defaults
	isPublished := true
	if r.AssessmentIsPublished != nil {
		isPublished = *r.AssessmentIsPublished
	}
	allowSubmission := true
	if r.AssessmentAllowSubmission != nil {
		allowSubmission = *r.AssessmentAllowSubmission
	}
	maxScore := 100.0
	if r.AssessmentMaxScore != nil {
		maxScore = *r.AssessmentMaxScore
	}
	totalAttempts := 1
	if r.AssessmentTotalAttemptsAllowed != nil {
		totalAttempts = *r.AssessmentTotalAttemptsAllowed
	}

	return model.AssessmentModel{
		AssessmentMasjidID:                     r.AssessmentMasjidID,
		AssessmentClassSectionSubjectTeacherID: r.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       r.AssessmentTypeID,

		AssessmentSlug:        r.AssessmentSlug,
		AssessmentTitle:       r.AssessmentTitle,
		AssessmentDescription: r.AssessmentDescription,

		AssessmentStartAt:     r.AssessmentStartAt,
		AssessmentDueAt:       r.AssessmentDueAt,
		AssessmentPublishedAt: r.AssessmentPublishedAt,
		AssessmentClosedAt:    r.AssessmentClosedAt,

		AssessmentDurationMinutes:      r.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: totalAttempts,
		AssessmentMaxScore:             maxScore,
		AssessmentIsPublished:          isPublished,
		AssessmentAllowSubmission:      allowSubmission,

		AssessmentCreatedByTeacherID: r.AssessmentCreatedByTeacherID,
	}
}

/* ==============================
   PATCH (PATCH /assessments/:id)
============================== */

type PatchAssessmentRequest struct {
	// Identitas
	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       *string `json:"assessment_title" validate:"omitempty,max=180"`
	AssessmentDescription *string `json:"assessment_description" validate:"omitempty"`

	// Jadwal
	AssessmentStartAt     *time.Time `json:"assessment_start_at" validate:"omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at" validate:"omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at" validate:"omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at" validate:"omitempty"`

	// Pengaturan
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,min=1,max=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,min=1,max=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool    `json:"assessment_is_published" validate:"omitempty"`
	AssessmentAllowSubmission      *bool    `json:"assessment_allow_submission" validate:"omitempty"`

	// Relasi
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty"`

	// Audit pembuat
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty"`
}

func (p *PatchAssessmentRequest) Normalize() {
	if p.AssessmentSlug != nil {
		s := strings.TrimSpace(*p.AssessmentSlug)
		if s == "" {
			p.AssessmentSlug = nil
		} else {
			p.AssessmentSlug = &s
		}
	}
	if p.AssessmentTitle != nil {
		t := strings.TrimSpace(*p.AssessmentTitle)
		if t == "" {
			p.AssessmentTitle = nil
		} else {
			p.AssessmentTitle = &t
		}
	}
	if p.AssessmentDescription != nil {
		d := strings.TrimSpace(*p.AssessmentDescription)
		if d == "" {
			p.AssessmentDescription = nil
		} else {
			p.AssessmentDescription = &d
		}
	}
}

// Apply menerapkan PATCH ke model (tanpa menyimpan ke DB)
func (p PatchAssessmentRequest) Apply(m *model.AssessmentModel) {
	if p.AssessmentSlug != nil {
		m.AssessmentSlug = p.AssessmentSlug
	}
	if p.AssessmentTitle != nil {
		m.AssessmentTitle = strings.TrimSpace(*p.AssessmentTitle)
	}
	if p.AssessmentDescription != nil {
		m.AssessmentDescription = p.AssessmentDescription
	}

	if p.AssessmentStartAt != nil {
		m.AssessmentStartAt = p.AssessmentStartAt
	}
	if p.AssessmentDueAt != nil {
		m.AssessmentDueAt = p.AssessmentDueAt
	}
	if p.AssessmentPublishedAt != nil {
		m.AssessmentPublishedAt = p.AssessmentPublishedAt
	}
	if p.AssessmentClosedAt != nil {
		m.AssessmentClosedAt = p.AssessmentClosedAt
	}

	if p.AssessmentDurationMinutes != nil {
		m.AssessmentDurationMinutes = p.AssessmentDurationMinutes
	}
	if p.AssessmentTotalAttemptsAllowed != nil {
		m.AssessmentTotalAttemptsAllowed = *p.AssessmentTotalAttemptsAllowed
	}
	if p.AssessmentMaxScore != nil {
		m.AssessmentMaxScore = *p.AssessmentMaxScore
	}
	if p.AssessmentIsPublished != nil {
		m.AssessmentIsPublished = *p.AssessmentIsPublished
	}
	if p.AssessmentAllowSubmission != nil {
		m.AssessmentAllowSubmission = *p.AssessmentAllowSubmission
	}

	if p.AssessmentClassSectionSubjectTeacherID != nil {
		m.AssessmentClassSectionSubjectTeacherID = p.AssessmentClassSectionSubjectTeacherID
	}
	if p.AssessmentTypeID != nil {
		m.AssessmentTypeID = p.AssessmentTypeID
	}
	if p.AssessmentCreatedByTeacherID != nil {
		m.AssessmentCreatedByTeacherID = p.AssessmentCreatedByTeacherID
	}
}

/* ==============================
   RESPONSES
============================== */

type AssessmentResponse struct {
	AssessmentID                           uuid.UUID  `json:"assessment_id"`
	AssessmentMasjidID                     uuid.UUID  `json:"assessment_masjid_id"`
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id,omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id,omitempty"`

	AssessmentSlug        *string `json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `json:"assessment_title"`
	AssessmentDescription *string `json:"assessment_description,omitempty"`

	AssessmentStartAt     *time.Time `json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at,omitempty"`

	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `json:"assessment_max_score"`
	AssessmentIsPublished          bool    `json:"assessment_is_published"`
	AssessmentAllowSubmission      bool    `json:"assessment_allow_submission"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id,omitempty"`

	AssessmentCreatedAt time.Time  `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time  `json:"assessment_updated_at"`
	AssessmentDeletedAt *time.Time `json:"assessment_deleted_at,omitempty"`
}

type ListAssessmentResponse struct {
	Data   []AssessmentResponse `json:"data"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

func FromModelAssesment(m model.AssessmentModel) AssessmentResponse {
	var deletedAt *time.Time
	if m.AssessmentDeletedAt.Valid {
		t := m.AssessmentDeletedAt.Time
		deletedAt = &t
	}
	return AssessmentResponse{
		AssessmentID:                           m.AssessmentID,
		AssessmentMasjidID:                     m.AssessmentMasjidID,
		AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       m.AssessmentTypeID,

		AssessmentSlug:        m.AssessmentSlug,
		AssessmentTitle:       m.AssessmentTitle,
		AssessmentDescription: m.AssessmentDescription,

		AssessmentStartAt:     m.AssessmentStartAt,
		AssessmentDueAt:       m.AssessmentDueAt,
		AssessmentPublishedAt: m.AssessmentPublishedAt,
		AssessmentClosedAt:    m.AssessmentClosedAt,

		AssessmentDurationMinutes:      m.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:             m.AssessmentMaxScore,
		AssessmentIsPublished:          m.AssessmentIsPublished,
		AssessmentAllowSubmission:      m.AssessmentAllowSubmission,

		AssessmentCreatedByTeacherID: m.AssessmentCreatedByTeacherID,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
		AssessmentDeletedAt: deletedAt,
	}
}

func FromModelsAssesments(items []model.AssessmentModel) []AssessmentResponse {
	out := make([]AssessmentResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelAssesment(it))
	}
	return out
}
