// File: internals/features/finance/spp/dto/bill_batch_dto.go
package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	// ganti path sesuai modelmu
	billing "madinahsalam_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// BILL BATCHES — DTO (versi terbaru, support periodic & one-off)
////////////////////////////////////////////////////////////////////////////////

// Create: wajib isi salah satu -> ClassID ATAU SectionID
// NOTE: Controller akan override BillBatchSchoolID dari :school_id path.
// Aturan:
// - PERIODIC  => option_code kosong => month & year WAJIB
// - ONE-OFF   => option_code terisi => month & year OPSIONAL
type BillBatchCreateDTO struct {
	BillBatchSchoolID uuid.UUID `json:"bill_batch_school_id" validate:"required"`

	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	// YM nullable supaya one-off bisa tanpa YM
	BillBatchMonth *int16 `json:"bill_batch_month,omitempty" validate:"omitempty,min=1,max=12"`
	BillBatchYear  *int16 `json:"bill_batch_year,omitempty"  validate:"omitempty,min=2000,max=2100"`

	BillBatchTermID *uuid.UUID `json:"bill_batch_term_id,omitempty"`

	// Katalog + kode
	BillBatchGeneralBillingKindID *uuid.UUID `json:"bill_batch_general_billing_kind_id,omitempty"`
	BillBatchBillCode             string     `json:"bill_batch_bill_code" validate:"required"` // default SPP jika kosong di controller
	BillBatchOptionCode           *string    `json:"bill_batch_option_code,omitempty"`         // WAJIB untuk one-off

	BillBatchTitle   string     `json:"bill_batch_title" validate:"required"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`
}

// Update (partial): tetap jaga XOR class/section saat apply.
// YM boleh diubah (nullable), code/kind/option juga boleh diubah.
type BillBatchUpdateDTO struct {
	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	BillBatchMonth  *int16     `json:"bill_batch_month,omitempty" validate:"omitempty,min=1,max=12"`
	BillBatchYear   *int16     `json:"bill_batch_year,omitempty"  validate:"omitempty,min=2000,max=2100"`
	BillBatchTermID *uuid.UUID `json:"bill_batch_term_id,omitempty"`

	BillBatchGeneralBillingKindID *uuid.UUID `json:"bill_batch_general_billing_kind_id,omitempty"`
	BillBatchBillCode             *string    `json:"bill_batch_bill_code,omitempty"`
	BillBatchOptionCode           *string    `json:"bill_batch_option_code,omitempty"`

	BillBatchTitle   *string    `json:"bill_batch_title,omitempty"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`
}

// Response — gunakan pointer untuk YM agar null merefleksikan one-off tanpa YM
type BillBatchResponse struct {
	BillBatchID        uuid.UUID  `json:"bill_batch_id"`
	BillBatchSchoolID  uuid.UUID  `json:"bill_batch_school_id"`
	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	BillBatchMonth  *int16     `json:"bill_batch_month,omitempty"`
	BillBatchYear   *int16     `json:"bill_batch_year,omitempty"`
	BillBatchTermID *uuid.UUID `json:"bill_batch_term_id,omitempty"`

	BillBatchGeneralBillingKindID *uuid.UUID `json:"bill_batch_general_billing_kind_id,omitempty"`
	BillBatchBillCode             string     `json:"bill_batch_bill_code"`
	BillBatchOptionCode           *string    `json:"bill_batch_option_code,omitempty"`

	BillBatchTitle   string     `json:"bill_batch_title"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`

	// denormalized totals (read-only di response)
	BillBatchTotalAmountIDR    int `json:"bill_batch_total_amount_idr"`
	BillBatchTotalPaidIDR      int `json:"bill_batch_total_paid_idr"`
	BillBatchTotalStudents     int `json:"bill_batch_total_students"`
	BillBatchTotalStudentsPaid int `json:"bill_batch_total_students_paid"`

	BillBatchCreatedAt time.Time  `json:"bill_batch_created_at"`
	BillBatchUpdatedAt time.Time  `json:"bill_batch_updated_at"`
	BillBatchDeletedAt *time.Time `json:"bill_batch_deleted_at,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// MAPPERS — Model <-> DTO
////////////////////////////////////////////////////////////////////////////////

func ToBillBatchResponse(m billing.BillBatch) BillBatchResponse {
	return BillBatchResponse{
		BillBatchID:                   m.BillBatchID,
		BillBatchSchoolID:             m.BillBatchSchoolID,
		BillBatchClassID:              m.BillBatchClassID,
		BillBatchSectionID:            m.BillBatchSectionID,
		BillBatchMonth:                m.BillBatchMonth,
		BillBatchYear:                 m.BillBatchYear,
		BillBatchTermID:               m.BillBatchTermID,
		BillBatchGeneralBillingKindID: m.BillBatchGeneralBillingKindID,
		BillBatchBillCode:             m.BillBatchBillCode,
		BillBatchOptionCode:           m.BillBatchOptionCode,
		BillBatchTitle:                m.BillBatchTitle,
		BillBatchDueDate:              m.BillBatchDueDate,
		BillBatchNote:                 m.BillBatchNote,

		BillBatchTotalAmountIDR:    m.BillBatchTotalAmountIDR,
		BillBatchTotalPaidIDR:      m.BillBatchTotalPaidIDR,
		BillBatchTotalStudents:     m.BillBatchTotalStudents,
		BillBatchTotalStudentsPaid: m.BillBatchTotalStudentsPaid,

		BillBatchCreatedAt: m.BillBatchCreatedAt,
		BillBatchUpdatedAt: m.BillBatchUpdatedAt,
		BillBatchDeletedAt: toPtrTimeFromDeletedAt(m.BillBatchDeletedAt),
	}
}

// CreateDTO -> Model
func BillBatchCreateDTOToModel(d BillBatchCreateDTO) billing.BillBatch {
	return billing.BillBatch{
		BillBatchSchoolID:             d.BillBatchSchoolID,
		BillBatchClassID:              d.BillBatchClassID,
		BillBatchSectionID:            d.BillBatchSectionID,
		BillBatchMonth:                d.BillBatchMonth,
		BillBatchYear:                 d.BillBatchYear,
		BillBatchTermID:               d.BillBatchTermID,
		BillBatchGeneralBillingKindID: d.BillBatchGeneralBillingKindID,
		BillBatchBillCode:             d.BillBatchBillCode,
		BillBatchOptionCode:           d.BillBatchOptionCode,
		BillBatchTitle:                d.BillBatchTitle,
		BillBatchDueDate:              d.BillBatchDueDate,
		BillBatchNote:                 d.BillBatchNote,
	}
}

// UpdateDTO -> Model (apply partial) + guard XOR + rules periodic/one-off ringan
func ApplyBillBatchUpdate(m *billing.BillBatch, d BillBatchUpdateDTO) error {
	// scope
	if d.BillBatchClassID != nil {
		m.BillBatchClassID = d.BillBatchClassID
	}
	if d.BillBatchSectionID != nil {
		m.BillBatchSectionID = d.BillBatchSectionID
	}

	// YM & term
	if d.BillBatchMonth != nil {
		m.BillBatchMonth = d.BillBatchMonth
	}
	if d.BillBatchYear != nil {
		m.BillBatchYear = d.BillBatchYear
	}
	if d.BillBatchTermID != nil {
		m.BillBatchTermID = d.BillBatchTermID
	}

	// katalog & kode
	if d.BillBatchGeneralBillingKindID != nil {
		m.BillBatchGeneralBillingKindID = d.BillBatchGeneralBillingKindID
	}
	if d.BillBatchBillCode != nil {
		m.BillBatchBillCode = safeStr(*d.BillBatchBillCode, "SPP")
	}
	if d.BillBatchOptionCode != nil {
		// empty string => treat as nil untuk kembali ke periodic
		if trimmed := trimOrNil(*d.BillBatchOptionCode); trimmed == nil {
			m.BillBatchOptionCode = nil
		} else {
			m.BillBatchOptionCode = trimmed
		}
	}

	// info
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

func safeStr(s string, def string) string {
	if v := trimOrEmpty(s); v != "" {
		return v
	}
	return def
}

func trimOrEmpty(s string) string {
	return string([]byte(s)) // keep as-is; controller bisa trimming jika perlu
}

func trimOrNil(s string) *string {
	t := s
	if len(t) == 0 {
		return nil
	}
	return &t
}

// Helpers list mapping
func ToBillBatchResponses(list []billing.BillBatch) []BillBatchResponse {
	out := make([]BillBatchResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToBillBatchResponse(v))
	}
	return out
}

// ===============================================
// REQUEST: Create BillBatch + Generate StudentBills
// ===============================================
//
// Catatan:
//   - XOR scope tetap sama.
//   - Untuk PERIODIC, set Labeling.OptionCode untuk student bills tapi biarkan
//     BillBatchOptionCode batch-nya kosong (ikut aturan batch periodic).
//   - Untuk ONE-OFF, set BillBatchOptionCode pada batch (di CreateDTO) dan
//     Labeling.OptionCode sebaiknya sama agar konsisten di student bills.
type BillBatchGenerateDTO struct {
	BillBatchClassID   *uuid.UUID `json:"bill_batch_class_id,omitempty"`
	BillBatchSectionID *uuid.UUID `json:"bill_batch_section_id,omitempty"`

	BillBatchMonth  *int16     `json:"bill_batch_month,omitempty" validate:"omitempty,min=1,max=12"`
	BillBatchYear   *int16     `json:"bill_batch_year,omitempty"  validate:"omitempty,min=2000,max=2100"`
	BillBatchTermID *uuid.UUID `json:"bill_batch_term_id,omitempty"`

	BillBatchGeneralBillingKindID *uuid.UUID `json:"bill_batch_general_billing_kind_id,omitempty"`
	BillBatchBillCode             string     `json:"bill_batch_bill_code" validate:"required"`
	BillBatchOptionCode           *string    `json:"bill_batch_option_code,omitempty"`

	BillBatchTitle   string     `json:"bill_batch_title" validate:"required"`
	BillBatchDueDate *time.Time `json:"bill_batch_due_date,omitempty"`
	BillBatchNote    *string    `json:"bill_batch_note,omitempty"`

	// Pilihan generate:
	SelectedStudentIDs []uuid.UUID `json:"selected_student_ids,omitempty"`
	OnlyActiveStudents bool        `json:"only_active_students"`

	// Labeling untuk student_bills yang di-generate
	Labeling struct {
		OptionCode  string  `json:"option_code" validate:"required"`
		OptionLabel *string `json:"option_label,omitempty"`
	} `json:"labeling" validate:"required"`
}

// ===============================================
// RESPONSE: Create+Generate
// ===============================================
type BillBatchGenerateResponse struct {
	BillBatch BillBatchResponse `json:"bill_batch"`
	Inserted  int               `json:"inserted"`
	Skipped   int               `json:"skipped"`
}
