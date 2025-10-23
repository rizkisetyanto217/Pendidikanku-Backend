// File: internals/features/finance/spp/dto/bill_batch_dto.go
package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	// ganti path sesuai modelmu
	billing "masjidku_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// BILL BATCHES — DTO
////////////////////////////////////////////////////////////////////////////////

// Create: wajib isi salah satu -> ClassID ATAU SectionID
type BillBatchCreateDTO struct {
	BillBatchMasjidID uuid.UUID `json:"bill_batch_masjid_id" validate:"required"`

	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	BillBatchMonth int16 `json:"bill_batch_month" validate:"required,min=1,max=12"`
	BillBatchYear  int16 `json:"bill_batch_year"  validate:"required,min=2000,max=2100"`

	BillBatchTermID *uuid.UUID `json:"bill_batch_term_id,omitempty"`

	BillBatchTitle   string     `json:"bill_batch_title" validate:"required"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`
}

// Update (partial): tetap jaga XOR class/section saat apply
type BillBatchUpdateDTO struct {
	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`
	BillBatchTermID    *uuid.UUID `json:"bill_batch_term_id,omitempty"`
	BillBatchTitle     *string    `json:"bill_batch_title,omitempty"`
	BillBatchDueDate   *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote      *string    `json:"bill_batch_note,omitempty"`
}

// Response
type BillBatchResponse struct {
	BillBatchID        uuid.UUID  `json:"bill_batch_id"`
	BillBatchMasjidID  uuid.UUID  `json:"bill_batch_masjid_id"`
	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	BillBatchMonth   int16      `json:"bill_batch_month"`
	BillBatchYear    int16      `json:"bill_batch_year"`
	BillBatchTermID  *uuid.UUID `json:"bill_batch_term_id,omitempty"`
	BillBatchTitle   string     `json:"bill_batch_title"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`

	BillBatchCreatedAt time.Time  `json:"bill_batch_created_at"`
	BillBatchUpdatedAt time.Time  `json:"bill_batch_updated_at"`
	BillBatchDeletedAt *time.Time `json:"bill_batch_deleted_at,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// MAPPERS — Model <-> DTO
////////////////////////////////////////////////////////////////////////////////

// Model -> Response
func ToBillBatchResponse(m billing.BillBatch) BillBatchResponse {
	return BillBatchResponse{
		BillBatchID:        m.BillBatchID,
		BillBatchMasjidID:  m.BillBatchMasjidID,
		BillBatchClassID:   m.BillBatchClassID,
		BillBatchSectionID: m.BillBatchSectionID,
		BillBatchMonth:     m.BillBatchMonth,
		BillBatchYear:      m.BillBatchYear,
		BillBatchTermID:    m.BillBatchTermID,
		BillBatchTitle:     m.BillBatchTitle,
		BillBatchDueDate:   m.BillBatchDueDate,
		BillBatchNote:      m.BillBatchNote,
		BillBatchCreatedAt: m.BillBatchCreatedAt,
		BillBatchUpdatedAt: m.BillBatchUpdatedAt,
		BillBatchDeletedAt: toPtrTimeFromDeletedAt(m.BillBatchDeletedAt),
	}
}

// CreateDTO -> Model
func BillBatchCreateDTOToModel(d BillBatchCreateDTO) billing.BillBatch {
	return billing.BillBatch{
		BillBatchMasjidID:  d.BillBatchMasjidID,
		BillBatchClassID:   d.BillBatchClassID,
		BillBatchSectionID: d.BillBatchSectionID,
		BillBatchMonth:     d.BillBatchMonth,
		BillBatchYear:      d.BillBatchYear,
		BillBatchTermID:    d.BillBatchTermID,
		BillBatchTitle:     d.BillBatchTitle,
		BillBatchDueDate:   d.BillBatchDueDate,
		BillBatchNote:      d.BillBatchNote,
	}
}

// UpdateDTO -> Model (apply partial) + guard XOR
func ApplyBillBatchUpdate(m *billing.BillBatch, d BillBatchUpdateDTO) error {
	if d.BillBatchClassID != nil {
		m.BillBatchClassID = d.BillBatchClassID
	}
	if d.BillBatchSectionID != nil {
		m.BillBatchSectionID = d.BillBatchSectionID
	}
	if d.BillBatchTermID != nil {
		m.BillBatchTermID = d.BillBatchTermID
	}
	if d.BillBatchTitle != nil {
		m.BillBatchTitle = *d.BillBatchTitle
	}
	if d.BillBatchDueDate != nil {
		m.BillBatchDueDate = d.BillBatchDueDate
	}
	if d.BillBatchNote != nil {
		m.BillBatchNote = d.BillBatchNote
	}

	// XOR guard di level DTO-apply (selain di hooks model)
	if (m.BillBatchClassID == nil && m.BillBatchSectionID == nil) ||
		(m.BillBatchClassID != nil && m.BillBatchSectionID != nil) {
		return fmt.Errorf("exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}
	return nil
}

// Helpers list mapping
func ToBillBatchResponses(list []billing.BillBatch) []BillBatchResponse {
	out := make([]BillBatchResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToBillBatchResponse(v))
	}
	return out
}
