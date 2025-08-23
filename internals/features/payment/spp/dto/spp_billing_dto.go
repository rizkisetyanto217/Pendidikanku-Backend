// internals/features/lembaga/spp/billings/dto/spp_billing_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/payment/spp/model"
)

/* =============== REQUESTS =============== */

// Create
type CreateSppBillingRequest struct {
	SppBillingMasjidID *uuid.UUID `json:"spp_billing_masjid_id" validate:"omitempty"`
	SppBillingClassID  *uuid.UUID `json:"spp_billing_class_id"  validate:"omitempty"`

	// (Opsional) relasi ke academic_terms
	SppBillingTermID *uuid.UUID `json:"spp_billing_term_id" validate:"omitempty"`

	SppBillingMonth int16  `json:"spp_billing_month" validate:"required,min=1,max=12"`      // 1..12
	SppBillingYear  int16  `json:"spp_billing_year"  validate:"required,gte=2000,lte=2100"` // 2000..2100
	SppBillingTitle string `json:"spp_billing_title" validate:"required,min=3"`

	SppBillingDueDate *time.Time `json:"spp_billing_due_date" validate:"omitempty"`
	SppBillingNote    *string    `json:"spp_billing_note"     validate:"omitempty"`
}

func (r CreateSppBillingRequest) ToModel() *m.SppBillingModel {
	return &m.SppBillingModel{
		SppBillingMasjidID: r.SppBillingMasjidID,
		SppBillingClassID:  r.SppBillingClassID,
		SppBillingTermID:   r.SppBillingTermID,
		SppBillingMonth:    r.SppBillingMonth,
		SppBillingYear:     r.SppBillingYear,
		SppBillingTitle:    r.SppBillingTitle,
		SppBillingDueDate:  r.SppBillingDueDate,
		SppBillingNote:     r.SppBillingNote,
	}
}

// Update (partial)
type UpdateSppBillingRequest struct {
	SppBillingMasjidID *uuid.UUID `json:"spp_billing_masjid_id" validate:"omitempty"`
	SppBillingClassID  *uuid.UUID `json:"spp_billing_class_id"  validate:"omitempty"`

	// (Opsional) relasi ke academic_terms
	SppBillingTermID *uuid.UUID `json:"spp_billing_term_id" validate:"omitempty"`

	SppBillingMonth *int16  `json:"spp_billing_month" validate:"omitempty,min=1,max=12"`
	SppBillingYear  *int16  `json:"spp_billing_year"  validate:"omitempty,gte=2000,lte=2100"`
	SppBillingTitle *string `json:"spp_billing_title" validate:"omitempty,min=1"`

	SppBillingDueDate *time.Time `json:"spp_billing_due_date" validate:"omitempty"`
	SppBillingNote    *string    `json:"spp_billing_note"     validate:"omitempty"`
}

// Terapkan perubahan ke model existing (untuk PUT)
func (r UpdateSppBillingRequest) ApplyTo(mo *m.SppBillingModel) {
	if r.SppBillingMasjidID != nil {
		mo.SppBillingMasjidID = r.SppBillingMasjidID
	}
	if r.SppBillingClassID != nil {
		mo.SppBillingClassID = r.SppBillingClassID
	}
	if r.SppBillingTermID != nil {
		mo.SppBillingTermID = r.SppBillingTermID
	}
	if r.SppBillingMonth != nil {
		mo.SppBillingMonth = *r.SppBillingMonth
	}
	if r.SppBillingYear != nil {
		mo.SppBillingYear = *r.SppBillingYear
	}
	if r.SppBillingTitle != nil {
		mo.SppBillingTitle = *r.SppBillingTitle
	}
	if r.SppBillingDueDate != nil {
		mo.SppBillingDueDate = r.SppBillingDueDate
	}
	if r.SppBillingNote != nil {
		mo.SppBillingNote = r.SppBillingNote
	}
}

// List / Query params
type ListSppBillingQuery struct {
	MasjidID *uuid.UUID `query:"masjid_id" validate:"omitempty"`
	ClassID  *uuid.UUID `query:"class_id"  validate:"omitempty"`
	TermID   *uuid.UUID `query:"term_id"   validate:"omitempty"`

	Month *int `query:"month" validate:"omitempty,min=1,max=12"`
	Year  *int `query:"year"  validate:"omitempty,gte=2000,lte=2100"`

	DueFrom *time.Time `query:"due_from" validate:"omitempty"`
	DueTo   *time.Time `query:"due_to"   validate:"omitempty,gtefield=DueFrom"`

	Q      *string `query:"q"      validate:"omitempty"`              // optional: cari di title/note, jika dipakai di query
	Limit  int     `query:"limit"  validate:"omitempty,gte=1,lte=100"`
	Offset int     `query:"offset" validate:"omitempty,gte=0"`
}

/* =============== RESPONSES =============== */

type SppBillingResponse struct {
	SppBillingID uuid.UUID `json:"spp_billing_id"`

	SppBillingMasjidID *uuid.UUID `json:"spp_billing_masjid_id,omitempty"`
	SppBillingClassID  *uuid.UUID `json:"spp_billing_class_id,omitempty"`
	SppBillingTermID   *uuid.UUID `json:"spp_billing_term_id,omitempty"`

	SppBillingMonth int16 `json:"spp_billing_month"`
	SppBillingYear  int16 `json:"spp_billing_year"`

	SppBillingTitle   string     `json:"spp_billing_title"`
	SppBillingDueDate *time.Time `json:"spp_billing_due_date,omitempty"`
	SppBillingNote    *string    `json:"spp_billing_note,omitempty"`

	SppBillingCreatedAt time.Time  `json:"spp_billing_created_at"`
	SppBillingUpdatedAt *time.Time `json:"spp_billing_updated_at,omitempty"`
}

type SppBillingListResponse struct {
	Items []SppBillingResponse `json:"items"`
	Total int64                `json:"total"`
}

/* =============== MAPPERS =============== */

func FromModel(x m.SppBillingModel) SppBillingResponse {
	return SppBillingResponse{
		SppBillingID:        x.SppBillingID,
		SppBillingMasjidID:  x.SppBillingMasjidID,
		SppBillingClassID:   x.SppBillingClassID,
		SppBillingTermID:    x.SppBillingTermID,
		SppBillingMonth:     x.SppBillingMonth,
		SppBillingYear:      x.SppBillingYear,
		SppBillingTitle:     x.SppBillingTitle,
		SppBillingDueDate:   x.SppBillingDueDate,
		SppBillingNote:      x.SppBillingNote,
		SppBillingCreatedAt: x.SppBillingCreatedAt,
		SppBillingUpdatedAt: x.SppBillingUpdatedAt,
	}
}

func FromModels(list []m.SppBillingModel, total int64) SppBillingListResponse {
	out := make([]SppBillingResponse, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return SppBillingListResponse{Items: out, Total: total}
}
