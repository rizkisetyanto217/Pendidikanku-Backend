// file: internals/features/school/assessments/dto/assessment_type_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

/* ==============================
   REQUESTS
============================== */

// Create (POST /assessment-types)
// Catatan: assessment_types_masjid_id tetap diisi dari token di controller.
type CreateAssessmentTypeRequest struct {
	AssessmentTypesMasjidID      uuid.UUID `json:"assessment_types_masjid_id" validate:"required"`
	AssessmentTypesKey           string    `json:"assessment_types_key" validate:"required,max=32"`
	AssessmentTypesName          string    `json:"assessment_types_name" validate:"required,max=120"`
	AssessmentTypesWeightPercent float32   `json:"assessment_types_weight_percent" validate:"gte=0,lte=100"`
	AssessmentTypesIsActive      *bool     `json:"assessment_types_is_active" validate:"omitempty"`
}

// Patch (PATCH /assessment-types/:id) — partial update
type PatchAssessmentTypeRequest struct {
	AssessmentTypesName          *string  `json:"assessment_types_name" validate:"omitempty,max=120"`
	AssessmentTypesWeightPercent *float32 `json:"assessment_types_weight_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypesIsActive      *bool    `json:"assessment_types_is_active" validate:"omitempty"`
}

// List filter (GET /assessment-types)
type ListAssessmentTypeFilter struct {
	AssessmentTypesMasjidID uuid.UUID `query:"masjid_id" validate:"required"` // diisi dari token di controller
	Active                  *bool     `query:"active" validate:"omitempty"`
	Q                       *string   `query:"q" validate:"omitempty,max=120"`
	Limit                   int       `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset                  int       `query:"offset" validate:"omitempty,min=0"`
	SortBy                  *string   `query:"sort_by" validate:"omitempty,oneof=name created_at"`
	SortDir                 *string   `query:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

/* ==============================
   RESPONSES
============================== */

type AssessmentTypeResponse struct {
	AssessmentTypesID            uuid.UUID `json:"assessment_types_id"`
	AssessmentTypesMasjidID      uuid.UUID `json:"assessment_types_masjid_id"`
	AssessmentTypesKey           string    `json:"assessment_types_key"`
	AssessmentTypesName          string    `json:"assessment_types_name"`
	AssessmentTypesWeightPercent float32   `json:"assessment_types_weight_percent"`
	AssessmentTypesIsActive      bool      `json:"assessment_types_is_active"`
	AssessmentTypesCreatedAt     time.Time `json:"assessment_types_created_at"`
	AssessmentTypesUpdatedAt     time.Time `json:"assessment_types_updated_at"`
}

type ListAssessmentTypeResponse struct {
	Data   []AssessmentTypeResponse `json:"data"`
	Total  int64                    `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}
