// file: internals/features/school/assessments/dto/assessment_type_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ===== Requests =====

// CreateAssessmentTypeRequest
// - MasjidID wajib (tenant)
// - Key max 32, gunakan huruf/angka/underscore/dash (opsional regex)
type CreateAssessmentTypeRequest struct {
	MasjidID      uuid.UUID `json:"assessment_types_masjid_id" validate:"required"`
	Key           string    `json:"assessment_types_key" validate:"required,max=32"`       // tambah `alphanumunicode`/regex sesuai kebutuhan
	Name          string    `json:"assessment_types_name" validate:"required,max=120"`
	WeightPercent float32   `json:"assessment_types_weight_percent" validate:"gte=0,lte=100"`
	// Optional: default true kalau kosong di DB
	IsActive *bool `json:"assessment_types_is_active" validate:"omitempty"`
}

// UpdateAssessmentTypeRequest (partial)
// - Semua optional; hanya field non-nil yang di-update
type UpdateAssessmentTypeRequest struct {
	Name          *string  `json:"assessment_types_name" validate:"omitempty,max=120"`
	WeightPercent *float32 `json:"assessment_types_weight_percent" validate:"omitempty,gte=0,lte=100"`
	IsActive      *bool    `json:"assessment_types_is_active" validate:"omitempty"`
}

// Filter & List (Query Params)
type ListAssessmentTypeFilter struct {
	MasjidID uuid.UUID `query:"masjid_id" validate:"required"` // gunakan di handler: c.Query("masjid_id")
	Active   *bool     `query:"active" validate:"omitempty"`
	Q        *string   `query:"q" validate:"omitempty,max=120"` // cari di name/key (opsional)
	Limit    int       `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset   int       `query:"offset" validate:"omitempty,min=0"`
	// Optional: sort by name/created_at, asc/desc
	SortBy  *string `query:"sort_by" validate:"omitempty,oneof=name created_at"`
	SortDir *string `query:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

// ===== Responses =====

type AssessmentTypeResponse struct {
	ID            uuid.UUID `json:"assessment_types_id"`
	MasjidID      uuid.UUID `json:"assessment_types_masjid_id"`
	Key           string    `json:"assessment_types_key"`
	Name          string    `json:"assessment_types_name"`
	WeightPercent float32   `json:"assessment_types_weight_percent"`
	IsActive      bool      `json:"assessment_types_is_active"`
	CreatedAt     time.Time `json:"assessment_types_created_at"`
	UpdatedAt     time.Time `json:"assessment_types_updated_at"`
}

type ListAssessmentTypeResponse struct {
	Data  []AssessmentTypeResponse `json:"data"`
	Total int64                    `json:"total"`
	Limit int                      `json:"limit"`
	Offset int                     `json:"offset"`
}
