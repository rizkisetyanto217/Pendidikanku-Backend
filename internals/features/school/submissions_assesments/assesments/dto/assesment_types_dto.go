// file: internals/features/school/assessments/dto/assessment_type_dto.go
package dto

import (
	"strings"
	"time"

	model "schoolku_backend/internals/features/school/submissions_assesments/assesments/model"

	"github.com/google/uuid"
)

/* ==============================
   REQUESTS
============================== */

// Create (POST /assessment-types)
// Catatan: assessment_type_school_id tetap diisi dari token di controller.
type CreateAssessmentTypeRequest struct {
	AssessmentTypeSchoolID      uuid.UUID `json:"assessment_type_school_id" validate:"required"`
	AssessmentTypeKey           string    `json:"assessment_type_key" validate:"required,max=32"`
	AssessmentTypeName          string    `json:"assessment_type_name" validate:"required,max=120"`
	AssessmentTypeWeightPercent float64   `json:"assessment_type_weight_percent" validate:"gte=0,lte=100"`
	AssessmentTypeIsActive      *bool     `json:"assessment_type_is_active" validate:"omitempty"`
}

// Patch (PATCH /assessment-types/:id) — partial update
type PatchAssessmentTypeRequest struct {
	AssessmentTypeName          *string  `json:"assessment_type_name" validate:"omitempty,max=120"`
	AssessmentTypeWeightPercent *float64 `json:"assessment_type_weight_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypeIsActive      *bool    `json:"assessment_type_is_active" validate:"omitempty"`
}

// List filter (GET /assessment-types)
type ListAssessmentTypeFilter struct {
	AssessmentTypeSchoolID uuid.UUID `query:"school_id" validate:"required"` // diisi dari token di controller
	Active                 *bool     `query:"active" validate:"omitempty"`
	Q                      *string   `query:"q" validate:"omitempty,max=120"`
	Limit                  int       `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset                 int       `query:"offset" validate:"omitempty,min=0"`
	SortBy                 *string   `query:"sort_by" validate:"omitempty,oneof=name created_at"`
	SortDir                *string   `query:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

/* ==============================
   RESPONSES
============================== */

type AssessmentTypeResponse struct {
	AssessmentTypeID            uuid.UUID `json:"assessment_type_id"`
	AssessmentTypeSchoolID      uuid.UUID `json:"assessment_type_school_id"`
	AssessmentTypeKey           string    `json:"assessment_type_key"`
	AssessmentTypeName          string    `json:"assessment_type_name"`
	AssessmentTypeWeightPercent float64   `json:"assessment_type_weight_percent"`
	AssessmentTypeIsActive      bool      `json:"assessment_type_is_active"`
	AssessmentTypeCreatedAt     time.Time `json:"assessment_type_created_at"`
	AssessmentTypeUpdatedAt     time.Time `json:"assessment_type_updated_at"`
}

type ListAssessmentTypeResponse struct {
	Data   []AssessmentTypeResponse `json:"data"`
	Total  int64                    `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

/* ==============================
   MAPPERS / HELPERS
============================== */

func (r CreateAssessmentTypeRequest) Normalize() CreateAssessmentTypeRequest {
	r.AssessmentTypeKey = strings.TrimSpace(r.AssessmentTypeKey)
	r.AssessmentTypeName = strings.TrimSpace(r.AssessmentTypeName)
	return r
}

func (r CreateAssessmentTypeRequest) ToModel() model.AssessmentTypeModel {
	// Default active = true agar tidak menimpa default DB dengan false (zero value)
	isActive := true
	if r.AssessmentTypeIsActive != nil {
		isActive = *r.AssessmentTypeIsActive
	}
	return model.AssessmentTypeModel{
		AssessmentTypeSchoolID:      r.AssessmentTypeSchoolID,
		AssessmentTypeKey:           r.AssessmentTypeKey,
		AssessmentTypeName:          r.AssessmentTypeName,
		AssessmentTypeWeightPercent: r.AssessmentTypeWeightPercent,
		AssessmentTypeIsActive:      isActive,
	}
}

func (p PatchAssessmentTypeRequest) Apply(m *model.AssessmentTypeModel) {
	if p.AssessmentTypeName != nil {
		name := strings.TrimSpace(*p.AssessmentTypeName)
		m.AssessmentTypeName = name
	}
	if p.AssessmentTypeWeightPercent != nil {
		m.AssessmentTypeWeightPercent = *p.AssessmentTypeWeightPercent
	}
	if p.AssessmentTypeIsActive != nil {
		m.AssessmentTypeIsActive = *p.AssessmentTypeIsActive
	}
}

func FromModel(m model.AssessmentTypeModel) AssessmentTypeResponse {
	return AssessmentTypeResponse{
		AssessmentTypeID:            m.AssessmentTypeID,
		AssessmentTypeSchoolID:      m.AssessmentTypeSchoolID,
		AssessmentTypeKey:           m.AssessmentTypeKey,
		AssessmentTypeName:          m.AssessmentTypeName,
		AssessmentTypeWeightPercent: m.AssessmentTypeWeightPercent,
		AssessmentTypeIsActive:      m.AssessmentTypeIsActive,
		AssessmentTypeCreatedAt:     m.AssessmentTypeCreatedAt,
		AssessmentTypeUpdatedAt:     m.AssessmentTypeUpdatedAt,
	}
}

func FromModels(items []model.AssessmentTypeModel) []AssessmentTypeResponse {
	out := make([]AssessmentTypeResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModel(it))
	}
	return out
}
